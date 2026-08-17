package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

const maxPrivateControlResult = 256 << 10

type privateControlEnvelope struct {
	Version int             `json:"version"`
	ID      string          `json:"id"`
	Action  string          `json:"action"`
	Project string          `json:"project,omitempty"`
	Args    json.RawMessage `json:"args,omitempty"`
}

type privateControlOutbox struct {
	Version   int            `json:"version"`
	ID        string         `json:"id"`
	Action    string         `json:"action"`
	Status    string         `json:"status"`
	Result    map[string]any `json:"result,omitempty"`
	Error     string         `json:"error,omitempty"`
	UpdatedAt string         `json:"updated_at"`
}

// syncPrivateControl exposes a small, private-only control surface over the Git
// relay. It exists for personal ChatGPT plans where GitHub writes are supported
// but custom MCP write tools are not: a lead chat can use Workbench memory,
// compact context and bounded safe repository eyes/hands without turning the
// human into a clipboard or consuming an autonomous coding worker.
func syncPrivateControl(ctx context.Context, repo, remote, branch, ref, mcpURL, authFile string) error {
	paths, err := relayPaths(repo, ref, "relay/control")
	if err != nil {
		return err
	}
	files := map[string][]byte{}
	if current, readErr := readRefFile(repo, ref, privateChatGuidePath, 128<<10); readErr != nil || !bytes.Equal(current, privateChatGuide) {
		files[privateChatGuidePath] = append([]byte(nil), privateChatGuide...)
	}
	for _, path := range paths {
		id := strings.TrimSuffix(filepath.Base(path), ".json")
		if !validRelayID(id) {
			continue
		}
		outPath := "relay/control-outbox/" + id + ".json"
		if refFileExists(repo, ref, outPath) {
			continue
		}
		out := privateControlOutbox{Version: 1, ID: id, Status: "failed", UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
		raw, readErr := readRefFile(repo, ref, path, 64<<10)
		if readErr != nil {
			out.Error = readErr.Error()
		} else {
			env, decodeErr := decodePrivateControl(raw, id)
			if decodeErr != nil {
				out.Error = decodeErr.Error()
			} else {
				out.Action = env.Action
				result, execErr := executePrivateControlForRepo(ctx, env, repo, mcpURL, authFile)
				if execErr != nil {
					out.Error = execErr.Error()
				} else {
					out.Status = "completed"
					out.Result = result
				}
			}
		}
		b, marshalErr := marshalPrivateControlOutbox(out)
		if marshalErr != nil {
			return marshalErr
		}
		files[outPath] = b
	}
	return publishPrivateControlFiles(ctx, repo, remote, branch, files)
}

func decodePrivateControl(raw []byte, idFromPath string) (privateControlEnvelope, error) {
	var env privateControlEnvelope
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&env); err != nil {
		return env, fmt.Errorf("invalid private control JSON: %w", err)
	}
	var trailing any
	if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
		return env, errors.New("private control JSON must contain one object")
	}
	env.ID = strings.TrimSpace(env.ID)
	env.Action = strings.ToLower(strings.TrimSpace(env.Action))
	env.Project = strings.TrimSpace(env.Project)
	if env.Version != 1 || env.ID != idFromPath || !validRelayID(env.ID) {
		return env, errors.New("private control id/version mismatch")
	}
	if !isPrivateSafeHandsAction(env.Action) {
		switch env.Action {
		case "save_memory", "search_memory", "save_context", "get_context", "update_workbench":
		default:
			return env, fmt.Errorf("unsupported private control action %q", env.Action)
		}
	}
	if len(env.Args) == 0 {
		env.Args = json.RawMessage(`{}`)
	}
	return env, nil
}

// executePrivateControl is kept as the unit-test-friendly MCP-only entry point.
// Relay execution uses executePrivateControlForRepo so maintenance actions can
// reuse the already-configured private transport without accepting a remote URL.
func executePrivateControl(ctx context.Context, env privateControlEnvelope, mcpURL, authFile string) (map[string]any, error) {
	return executePrivateControlForRepo(ctx, env, "", mcpURL, authFile)
}

func executePrivateControlForRepo(ctx context.Context, env privateControlEnvelope, relayRepo, mcpURL, authFile string) (map[string]any, error) {
	if isPrivateSafeHandsAction(env.Action) {
		return executePrivateSafeHands(ctx, env, mcpURL, authFile)
	}

	switch env.Action {
	case "save_memory":
		var a struct {
			Scope   string   `json:"scope"`
			Kind    string   `json:"kind,omitempty"`
			Title   string   `json:"title"`
			Content string   `json:"content"`
			Tags    []string `json:"tags,omitempty"`
			Source  string   `json:"source,omitempty"`
		}
		if err := decodePrivateControlArgs(env.Args, &a); err != nil {
			return nil, err
		}
		a.Scope = strings.ToLower(strings.TrimSpace(a.Scope))
		a.Kind = strings.ToLower(strings.TrimSpace(a.Kind))
		if a.Scope != "global" && a.Scope != "project" {
			return nil, errors.New("save_memory scope must be global or project")
		}
		if a.Kind == "" {
			a.Kind = "fact"
		}
		if !validKnowledgeKind(a.Kind) {
			return nil, errors.New("save_memory kind must be fact, decision, constraint, pattern, routine or code")
		}
		args := map[string]any{"scope": a.Scope, "kind": a.Kind, "title": strings.TrimSpace(a.Title), "content": strings.TrimSpace(a.Content), "tags": a.Tags, "source": strings.TrimSpace(a.Source)}
		if a.Scope == "project" {
			project, err := resolveProject(env.Project)
			if err != nil {
				return nil, err
			}
			args["project_path"] = project
		}
		return callMCP(ctx, mcpURL, authFile, "save_memory", args)

	case "search_memory":
		var a struct {
			Query string `json:"query,omitempty"`
			Limit int    `json:"limit,omitempty"`
		}
		if err := decodePrivateControlArgs(env.Args, &a); err != nil {
			return nil, err
		}
		if a.Limit <= 0 || a.Limit > 20 {
			a.Limit = 20
		}
		args := map[string]any{"query": strings.TrimSpace(a.Query), "limit": a.Limit}
		if env.Project != "" {
			project, err := resolveProject(env.Project)
			if err != nil {
				return nil, err
			}
			args["project_path"] = project
		}
		return callMCP(ctx, mcpURL, authFile, "search_memory", args)

	case "save_context":
		var a struct {
			Objective   string   `json:"objective"`
			State       string   `json:"state"`
			Decisions   []string `json:"decisions,omitempty"`
			Constraints []string `json:"constraints,omitempty"`
			References  []string `json:"references,omitempty"`
			OpenThreads []string `json:"open_threads,omitempty"`
			NextAction  string   `json:"next_action,omitempty"`
		}
		if err := decodePrivateControlArgs(env.Args, &a); err != nil {
			return nil, err
		}
		project, err := resolveProject(env.Project)
		if err != nil {
			return nil, err
		}
		return callMCP(ctx, mcpURL, authFile, "save_context", map[string]any{
			"project_path": project,
			"objective": strings.TrimSpace(a.Objective),
			"state": strings.TrimSpace(a.State),
			"decisions": a.Decisions,
			"constraints": a.Constraints,
			"references": a.References,
			"open_threads": a.OpenThreads,
			"next_action": strings.TrimSpace(a.NextAction),
		})

	case "get_context":
		if err := decodePrivateControlArgs(env.Args, &struct{}{}); err != nil {
			return nil, err
		}
		project, err := resolveProject(env.Project)
		if err != nil {
			return nil, err
		}
		return callMCP(ctx, mcpURL, authFile, "get_context", map[string]any{"project_path": project})

	case "update_workbench":
		if env.Project != "" {
			return nil, errors.New("update_workbench does not accept a project")
		}
		if err := decodePrivateControlArgs(env.Args, &struct{}{}); err != nil {
			return nil, err
		}
		return schedulePrivateWorkbenchUpdate(relayRepo)
	}
	return nil, errors.New("unsupported private control action")
}

func decodePrivateControlArgs(raw json.RawMessage, dst any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid private control args: %w", err)
	}
	var trailing any
	if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("private control args must contain one object")
	}
	return nil
}

func validKnowledgeKind(kind string) bool {
	switch kind {
	case "fact", "decision", "constraint", "pattern", "routine", "code":
		return true
	default:
		return false
	}
}

func marshalPrivateControlOutbox(out privateControlOutbox) ([]byte, error) {
	if out.Result != nil {
		b, _ := json.Marshal(out.Result)
		if core.LooksSecret(string(b)) {
			out.Status = "failed"
			out.Result = nil
			out.Error = "private control result was withheld because it resembled secret material"
		}
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, err
	}
	if len(b) > maxPrivateControlResult {
		return nil, errors.New("private control result exceeds 256 KiB")
	}
	return append(b, '\n'), nil
}

func refFileExists(repo, ref, path string) bool {
	return exec.Command("git", "-C", repo, "cat-file", "-e", ref+":"+path).Run() == nil
}

func publishPrivateControlFiles(ctx context.Context, repo, remote, branch string, files map[string][]byte) error {
	if len(files) == 0 {
		return nil
	}
	stagePaths := make([]string, 0, len(files))
	for path := range files {
		stagePaths = append(stagePaths, path)
	}
	sort.Strings(stagePaths)

	for attempt := 0; attempt < 3; attempt++ {
		if err := fetchRemote(ctx, repo, remote, branch); err != nil {
			return err
		}
		ref := remote + "/" + branch
		tmp, err := os.MkdirTemp("", "workbench-relay-control-")
		if err != nil {
			return err
		}
		added := exec.Command("git", "-C", repo, "worktree", "add", "--detach", "--quiet", tmp, ref)
		if out, err := added.CombinedOutput(); err != nil {
			_ = os.RemoveAll(tmp)
			return fmt.Errorf("create private control publish worktree: %s", strings.TrimSpace(string(out)))
		}
		for path, content := range files {
			dest := filepath.Join(tmp, filepath.FromSlash(path))
			if old, readErr := os.ReadFile(dest); readErr == nil && bytes.Equal(old, content) {
				continue
			}
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				cleanupWorktree(repo, tmp)
				return err
			}
			if err := os.WriteFile(dest, content, 0o644); err != nil {
				cleanupWorktree(repo, tmp)
				return err
			}
		}
		addArgs := []string{"-C", tmp, "add", "--"}
		addArgs = append(addArgs, stagePaths...)
		if out, err := exec.Command("git", addArgs...).CombinedOutput(); err != nil {
			cleanupWorktree(repo, tmp)
			return fmt.Errorf("stage private relay files: %s", strings.TrimSpace(string(out)))
		}
		diffArgs := []string{"-C", tmp, "diff", "--cached", "--quiet", "--"}
		diffArgs = append(diffArgs, stagePaths...)
		diffCmd := exec.Command("git", diffArgs...)
		diffOut, diffErr := diffCmd.CombinedOutput()
		if diffErr == nil {
			cleanupWorktree(repo, tmp)
			return nil
		}
		if exitErr, ok := diffErr.(*exec.ExitError); !ok || exitErr.ExitCode() != 1 {
			cleanupWorktree(repo, tmp)
			return fmt.Errorf("check staged private relay files: %s", strings.TrimSpace(string(diffOut)))
		}
		commit := exec.Command("git", "-C", tmp, "-c", "user.name=Workbench Relay", "-c", "user.email=workbench-relay@users.noreply.github.com", "commit", "--quiet", "-m", "relay: update private Workbench state")
		if out, err := commit.CombinedOutput(); err != nil {
			cleanupWorktree(repo, tmp)
			return fmt.Errorf("commit private relay files: %s", strings.TrimSpace(string(out)))
		}
		pushCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
		push := exec.CommandContext(pushCtx, "git", "-C", tmp, "push", "--quiet", remote, "HEAD:refs/heads/"+branch)
		out, pushErr := push.CombinedOutput()
		cancel()
		cleanupWorktree(repo, tmp)
		if pushErr == nil {
			return nil
		}
		if attempt == 2 {
			return fmt.Errorf("push private relay files failed: %s", strings.TrimSpace(string(out)))
		}
		time.Sleep(time.Duration(attempt+1) * 250 * time.Millisecond)
	}
	return nil
}
