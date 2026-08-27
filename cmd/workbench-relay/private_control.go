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

const (
	maxPrivateControlResult     = 256 << 10
	maxPrivateControlErrorBytes = 32 << 10
	maxPrivateControlsPerPoll   = 12
)

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

type privateControlCandidate struct {
	path      string
	id        string
	env       privateControlEnvelope
	readErr   error
	decodeErr error
}

// syncPrivateControl exposes a small, private-only control surface over the Git
// relay. It exists for personal ChatGPT plans where GitHub writes are supported
// but custom MCP write tools are not: a lead chat can use Workbench memory,
// compact context and bounded safe repository eyes/hands without turning the
// human into a clipboard or consuming an autonomous coding worker.
//
// Results are deliberately published one request at a time. A slow, malformed,
// oversized, or secret-like request therefore cannot prevent unrelated project
// chats from receiving their own results. update_workbench is prioritised and
// returned immediately after its acknowledgement is pushed, giving the fixed
// updater grace time to restart the relay without losing that acknowledgement.
func syncPrivateControl(ctx context.Context, repo, remote, branch, ref, mcpURL, authFile string) error {
	paths, err := relayPaths(repo, ref, "relay/control")
	if err != nil {
		return err
	}
	outboxPaths, err := relayPaths(repo, ref, "relay/control-outbox")
	if err != nil {
		return err
	}
	paths = pendingPrivateControlPaths(paths, outboxPaths)

	metadata := map[string][]byte{}
	if current, readErr := readRefFile(repo, ref, privateChatGuidePath, 128<<10); readErr != nil || !bytes.Equal(current, privateChatGuide) {
		metadata[privateChatGuidePath] = append([]byte(nil), privateChatGuide...)
	}
	capabilities, err := privateChatCapabilitiesJSON()
	if err != nil {
		return err
	}
	if current, readErr := readRefFile(repo, ref, privateChatCapabilitiesPath, 128<<10); readErr != nil || !bytes.Equal(current, capabilities) {
		metadata[privateChatCapabilitiesPath] = capabilities
	}
	if err := publishPrivateControlFiles(ctx, repo, remote, branch, metadata); err != nil {
		return err
	}

	candidates := make([]privateControlCandidate, 0, len(paths))
	for _, controlPath := range paths {
		id := strings.TrimSuffix(filepath.Base(controlPath), ".json")
		if !validRelayID(id) {
			continue
		}

		candidate := privateControlCandidate{path: controlPath, id: id}
		raw, readErr := readRefFile(repo, ref, controlPath, 64<<10)
		candidate.readErr = readErr
		if readErr == nil {
			candidate.env, candidate.decodeErr = decodePrivateControl(raw, id)
		}
		candidates = append(candidates, candidate)
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		left := privateControlPriority(candidates[i].env.Action)
		right := privateControlPriority(candidates[j].env.Action)
		if left != right {
			return left < right
		}
		return candidates[i].path < candidates[j].path
	})
	if len(candidates) > maxPrivateControlsPerPoll {
		candidates = candidates[:maxPrivateControlsPerPoll]
	}

	for _, candidate := range candidates {
		out := executePrivateControlCandidate(ctx, candidate, repo, mcpURL, authFile)
		b, marshalErr := marshalPrivateControlOutbox(out)
		if marshalErr != nil {
			return marshalErr
		}
		outPath := "relay/control-outbox/" + candidate.id + ".json"
		if err := publishPrivateControlFiles(ctx, repo, remote, branch, map[string][]byte{outPath: b}); err != nil {
			return err
		}
		if candidate.env.Action == "update_workbench" && candidate.readErr == nil && candidate.decodeErr == nil {
			return nil
		}
	}
	return nil
}

func pendingPrivateControlPaths(controlPaths, outboxPaths []string) []string {
	completed := make(map[string]struct{}, len(outboxPaths))
	for _, outboxPath := range outboxPaths {
		id := strings.TrimSuffix(filepath.Base(outboxPath), ".json")
		if validRelayID(id) {
			completed[id] = struct{}{}
		}
	}

	pending := make([]string, 0, len(controlPaths))
	for _, controlPath := range controlPaths {
		id := strings.TrimSuffix(filepath.Base(controlPath), ".json")
		if !validRelayID(id) {
			continue
		}
		if _, done := completed[id]; done {
			continue
		}
		pending = append(pending, controlPath)
	}
	return pending
}

func executePrivateControlCandidate(ctx context.Context, candidate privateControlCandidate, repo, mcpURL, authFile string) privateControlOutbox {
	out := privateControlOutbox{
		Version:   1,
		ID:        candidate.id,
		Status:    "failed",
		UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	if candidate.readErr != nil {
		out.Error = candidate.readErr.Error()
		return out
	}
	if candidate.decodeErr != nil {
		out.Error = candidate.decodeErr.Error()
		return out
	}

	out.Action = candidate.env.Action
	result, execErr := executePrivateControlForRepo(ctx, candidate.env, repo, mcpURL, authFile)
	if execErr != nil {
		out.Error = execErr.Error()
		return out
	}
	out.Status = "completed"
	out.Result = result
	return out
}

func privateControlPriority(action string) int {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "update_workbench":
		return 0
	case "update_status":
		return 1
	default:
		return 10
	}
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
	if len(out.Error) > maxPrivateControlErrorBytes {
		out.Error = out.Error[:maxPrivateControlErrorBytes] + "\n… private control error truncated by Workbench …"
	}

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, err
	}
	if len(b) > maxPrivateControlResult {
		out.Status = "failed"
		out.Result = nil
		out.Error = "private control result exceeded 256 KiB and was withheld; request a narrower bounded result"
		b, err = json.MarshalIndent(out, "", "  ")
		if err != nil {
			return nil, err
		}
	}
	if len(b) > maxPrivateControlResult {
		return nil, errors.New("private control failure envelope unexpectedly exceeds 256 KiB")
	}
	return append(b, '\n'), nil
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
		if out, err := relayGitCombinedOutput(ctx, relayGitLocalTimeout, repo, "worktree", "add", "--detach", "--quiet", tmp, ref); err != nil {
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
		addArgs := []string{"add", "--"}
		addArgs = append(addArgs, stagePaths...)
		if out, err := relayGitCombinedOutput(ctx, relayGitLocalTimeout, tmp, addArgs...); err != nil {
			cleanupWorktree(repo, tmp)
			return fmt.Errorf("stage private relay files: %s", strings.TrimSpace(string(out)))
		}
		diffArgs := []string{"diff", "--cached", "--quiet", "--"}
		diffArgs = append(diffArgs, stagePaths...)
		diffOut, diffErr := relayGitCombinedOutput(ctx, relayGitLocalTimeout, tmp, diffArgs...)
		if diffErr == nil {
			cleanupWorktree(repo, tmp)
			return nil
		}
		if exitErr, ok := diffErr.(*exec.ExitError); !ok || exitErr.ExitCode() != 1 {
			cleanupWorktree(repo, tmp)
			return fmt.Errorf("check staged private relay files: %s", strings.TrimSpace(string(diffOut)))
		}
		if out, err := relayGitCombinedOutput(ctx, relayGitLocalTimeout, tmp, "-c", "user.name=Workbench Relay", "-c", "user.email=workbench-relay@users.noreply.github.com", "commit", "--quiet", "-m", "relay: update private Workbench state"); err != nil {
			cleanupWorktree(repo, tmp)
			return fmt.Errorf("commit private relay files: %s", strings.TrimSpace(string(out)))
		}
		out, pushErr := relayGitCombinedOutput(ctx, relayGitNetworkTimeout, tmp, "push", "--quiet", remote, "HEAD:refs/heads/"+branch)
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
