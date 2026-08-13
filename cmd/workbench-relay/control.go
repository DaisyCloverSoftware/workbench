package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

type controlEnvelope struct {
	Version     int      `json:"version"`
	ID          string   `json:"id"`
	Action      string   `json:"action"`
	Project     string   `json:"project,omitempty"`
	Scope       string   `json:"scope,omitempty"`
	Kind        string   `json:"kind,omitempty"`
	Title       string   `json:"title,omitempty"`
	Summary     string   `json:"summary,omitempty"`
	Content     string   `json:"content,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Decisions   []string `json:"decisions,omitempty"`
	OpenLoops   []string `json:"open_loops,omitempty"`
	NextActions []string `json:"next_actions,omitempty"`
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	Triggers    []string `json:"triggers,omitempty"`
	Steps       []string `json:"steps,omitempty"`
	Code        string   `json:"code,omitempty"`
	Language    string   `json:"language,omitempty"`
	Query       string   `json:"query,omitempty"`
	Limit       int      `json:"limit,omitempty"`
	MaxItems    int      `json:"max_items,omitempty"`
	MaxChars    int      `json:"max_chars,omitempty"`
}

type controlOutboxEnvelope struct {
	Version   int            `json:"version"`
	ID        string         `json:"id"`
	Action    string         `json:"action"`
	Status    string         `json:"status"`
	Result    map[string]any `json:"result,omitempty"`
	Error     string         `json:"error,omitempty"`
	UpdatedAt string         `json:"updated_at"`
}

func processControlPath(ctx context.Context, repo, ref, path, mcpURL, authFile string) error {
	id := strings.TrimSuffix(filepath.Base(path), ".json")
	if !validRelayID(id) {
		return errors.New("invalid relay control filename")
	}
	raw, err := readRefFile(repo, ref, path, 96<<10)
	if err != nil {
		return err
	}
	digest := rawDigest(raw)
	if rec, ok, loadErr := core.LoadRelayControlRecord(id); loadErr == nil && ok && rec.Digest == digest {
		return nil
	}

	var env controlEnvelope
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&env); err != nil {
		return saveControlFailure(id, path, digest, "invalid", "invalid relay control JSON: "+err.Error())
	}
	env.Action = strings.ToLower(strings.TrimSpace(env.Action))
	if env.Version != 1 || env.ID != id || !validRelayID(env.ID) {
		return saveControlFailure(id, path, digest, env.Action, "relay control id/version mismatch")
	}

	tool, args, needsProject, err := controlCall(env)
	if err != nil {
		return saveControlFailure(id, path, digest, env.Action, err.Error())
	}
	if strings.TrimSpace(env.Project) != "" {
		project, err := resolveProject(env.Project)
		if err != nil {
			return saveControlFailure(id, path, digest, env.Action, err.Error())
		}
		args["project_path"] = project
	} else if needsProject {
		return saveControlFailure(id, path, digest, env.Action, "relay control action requires a project repository name")
	}

	result, callErr := callMCP(ctx, mcpURL, authFile, tool, args)
	response := controlOutboxEnvelope{
		Version:   1,
		ID:        id,
		Action:    env.Action,
		Status:    "completed",
		Result:    result,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	if callErr != nil {
		response.Status = "failed"
		response.Result = nil
		response.Error = callErr.Error()
	}
	b, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return err
	}
	if core.LooksSecret(string(b)) {
		response.Status = "failed"
		response.Result = nil
		response.Error = "control result withheld because probable secret material was detected"
		b, err = json.MarshalIndent(response, "", "  ")
		if err != nil {
			return err
		}
	}
	b = append(b, '\n')
	if err := core.SaveRelayControlRecord(core.RelayControlRecord{RelayID: id, SourcePath: path, Digest: digest, Action: env.Action, Response: b}); err != nil {
		return err
	}
	fmt.Printf("processed relay control %s (%s)\n", id, env.Action)
	return nil
}

func controlCall(env controlEnvelope) (string, map[string]any, bool, error) {
	scope := strings.ToLower(strings.TrimSpace(env.Scope))
	if scope == "" {
		scope = "project"
	}
	args := map[string]any{}
	switch env.Action {
	case "checkpoint":
		args["summary"] = env.Summary
		args["decisions"] = env.Decisions
		args["open_loops"] = env.OpenLoops
		args["next_actions"] = env.NextActions
		return "save_checkpoint", args, true, nil
	case "remember":
		args["scope"] = scope
		args["kind"] = env.Kind
		args["title"] = env.Title
		args["summary"] = env.Summary
		args["content"] = env.Content
		args["tags"] = env.Tags
		return "remember", args, scope != "global", nil
	case "routine":
		args["scope"] = scope
		args["name"] = env.Name
		args["description"] = env.Description
		args["triggers"] = env.Triggers
		args["steps"] = env.Steps
		args["code"] = env.Code
		args["language"] = env.Language
		args["tags"] = env.Tags
		return "save_routine", args, scope != "global", nil
	case "context":
		args["query"] = env.Query
		args["max_items"] = env.MaxItems
		args["max_chars"] = env.MaxChars
		return "get_context_pack", args, true, nil
	case "recall":
		args["query"] = env.Query
		args["limit"] = env.Limit
		return "recall_memory", args, strings.TrimSpace(env.Project) != "", nil
	case "routines":
		args["query"] = env.Query
		args["limit"] = env.Limit
		return "find_routines", args, strings.TrimSpace(env.Project) != "", nil
	default:
		return "", nil, false, fmt.Errorf("unsupported relay control action %q", env.Action)
	}
}

func saveControlFailure(id, path, digest, action, message string) error {
	if action == "" {
		action = "invalid"
	}
	response := controlOutboxEnvelope{
		Version:   1,
		ID:        id,
		Action:    action,
		Status:    "failed",
		Error:     message,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	b, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if saveErr := core.SaveRelayControlRecord(core.RelayControlRecord{RelayID: id, SourcePath: path, Digest: digest, Action: action, Response: b}); saveErr != nil {
		return saveErr
	}
	return nil
}

func syncControlOutbox(ctx context.Context, repo, remote, branch string) error {
	records, err := core.ListRelayControlRecords()
	if err != nil {
		return err
	}
	files := map[string][]byte{}
	for _, rec := range records {
		if !validRelayID(rec.RelayID) || len(rec.Response) == 0 {
			continue
		}
		files["relay/control-outbox/"+rec.RelayID+".json"] = append([]byte(nil), rec.Response...)
	}
	return publishOutbox(ctx, repo, remote, branch, files)
}

func rawDigest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
