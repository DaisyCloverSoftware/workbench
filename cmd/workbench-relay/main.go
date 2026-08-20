package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

const relayVersion = "0.9.48"

type envelope struct {
	Version   int    `json:"version"`
	ID        string `json:"id"`
	Project   string `json:"project"`
	Intent    string `json:"intent"`
	CreatedAt string `json:"created_at,omitempty"`
}

type answerEnvelope struct {
	Version int    `json:"version"`
	ID      string `json:"id"`
	Answer  string `json:"answer"`
}

type outboxEnvelope struct {
	Version        int             `json:"version"`
	ID             string          `json:"id"`
	WorkbenchTask  string          `json:"workbench_task_id,omitempty"`
	Status         core.TaskStatus `json:"status"`
	ConsumesWork   bool            `json:"consumes_work,omitempty"`
	Report         string          `json:"report,omitempty"`
	Error          string          `json:"error,omitempty"`
	Attention      string          `json:"attention,omitempty"`
	DetailWithheld bool            `json:"detail_withheld,omitempty"`
	UpdatedAt      string          `json:"updated_at"`
}

type rpcResponse struct {
	Result struct {
		StructuredContent map[string]any `json:"structuredContent"`
		IsError           bool           `json:"isError"`
	} `json:"result"`
	Error any `json:"error"`
}

func main() {
	repoDir := flag.String("repo-dir", "", "git clone used as the relay transport")
	remote := flag.String("remote", "origin", "git remote containing relay messages")
	branch := flag.String("branch", "main", "remote branch containing relay messages")
	interval := flag.Duration("interval", 10*time.Second, "poll interval")
	mcpURL := flag.String("mcp-url", "http://127.0.0.1:8765/mcp", "local Workbench MCP endpoint")
	authFile := flag.String("auth-file", defaultAuthFile(), "file containing local MCP Authorization value")
	resultMode := flag.String("result-mode", "status", "outbox detail: status or report")
	publicTransport := flag.Bool("public-transport", true, "publish only status-safe output suitable for a public relay repository")
	once := flag.Bool("once", false, "poll once then exit")
	flag.Parse()

	if *resultMode != "status" && *resultMode != "report" {
		fatal(errors.New("result-mode must be status or report"))
	}
	if *publicTransport && *resultMode == "report" {
		fatal(errors.New("report result mode requires a private relay transport; use --public-transport=false"))
	}
	if strings.TrimSpace(*repoDir) == "" {
		if wd, err := os.Getwd(); err == nil {
			*repoDir = wd
		}
	}
	absRepo, err := filepath.Abs(*repoDir)
	if err != nil {
		fatal(err)
	}
	if err := verifyRepo(absRepo, *remote); err != nil {
		fatal(err)
	}

	fmt.Printf("Workbench Git Relay %s\n", relayVersion)
	fmt.Printf("relay repo: %s\n", absRepo)
	fmt.Printf("source ref: %s/%s\n", *remote, *branch)
	fmt.Printf("local MCP: %s\n", *mcpURL)
	fmt.Printf("outbox mode: %s (public transport: %t)\n", *resultMode, *publicTransport)

	for {
		if err := poll(context.Background(), absRepo, *remote, *branch, *mcpURL, *authFile, *resultMode, *publicTransport); err != nil {
			fmt.Fprintln(os.Stderr, "relay poll:", err)
		}
		if *once {
			return
		}
		time.Sleep(*interval)
	}
}

func defaultAuthFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "workbench", "mcp-loopback-auth-value")
}

func verifyRepo(repo, remote string) error {
	if st, err := os.Stat(filepath.Join(repo, ".git")); err != nil || !st.IsDir() {
		return fmt.Errorf("relay repo is not a git clone: %s", repo)
	}
	cmd := exec.Command("git", "-C", repo, "remote", "get-url", remote)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git remote %q unavailable: %s", remote, strings.TrimSpace(string(out)))
	}
	return nil
}

func poll(ctx context.Context, repo, remote, branch, mcpURL, authFile, resultMode string, publicTransport bool) error {
	if err := fetchRemote(ctx, repo, remote, branch); err != nil {
		return err
	}
	ref := remote + "/" + branch

	paths, err := relayPaths(repo, ref, "relay/inbox")
	if err != nil {
		return err
	}
	for _, path := range paths {
		if err := processPath(ctx, repo, ref, path, mcpURL, authFile); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
		}
	}

	answers, err := relayPaths(repo, ref, "relay/answers")
	if err != nil {
		return err
	}
	for _, path := range answers {
		if err := processAnswerPath(ctx, repo, ref, path, mcpURL, authFile); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
		}
	}

	if !publicTransport {
		if err := syncPrivateControl(ctx, repo, remote, branch, ref, mcpURL, authFile); err != nil {
			return err
		}
	}
	return syncOutbox(ctx, repo, remote, branch, mcpURL, authFile, resultMode, publicTransport)
}

func fetchRemote(ctx context.Context, repo, remote, branch string) error {
	fetchCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(fetchCtx, "git", "-C", repo, "fetch", "--quiet", remote, branch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git fetch failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func relayPaths(repo, ref, prefix string) ([]string, error) {
	cmd := exec.Command("git", "-C", repo, "ls-tree", "-r", "--name-only", ref, "--", prefix)
	out, err := cmd.Output()
	if err != nil {
		return nil, nil
	}
	var paths []string
	needle := strings.TrimSuffix(prefix, "/") + "/"
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, needle) && strings.HasSuffix(line, ".json") {
			paths = append(paths, line)
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func readRefFile(repo, ref, path string, max int64) ([]byte, error) {
	cmd := exec.Command("git", "-C", repo, "show", ref+":"+path)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	if int64(len(out)) > max {
		return nil, fmt.Errorf("relay file exceeds %d bytes", max)
	}
	return out, nil
}

func processPath(ctx context.Context, repo, ref, path, mcpURL, authFile string) error {
	idFromPath := strings.TrimSuffix(filepath.Base(path), ".json")
	if !validRelayID(idFromPath) {
		return errors.New("invalid relay filename")
	}
	if rec, ok, err := core.LoadRelayRecord(idFromPath); err == nil && ok && (rec.WorkbenchTaskID != "" || rec.Error != "") {
		return nil
	}

	out, err := readRefFile(repo, ref, path, 64<<10)
	if err != nil {
		return err
	}
	var env envelope
	dec := json.NewDecoder(bytes.NewReader(out))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&env); err != nil {
		return recordError(idFromPath, path, "invalid relay JSON: "+err.Error())
	}
	if env.Version != 1 || env.ID != idFromPath || !validRelayID(env.ID) {
		return recordError(idFromPath, path, "relay id/version mismatch")
	}
	project, err := resolveProject(env.Project)
	if err != nil {
		return recordError(idFromPath, path, err.Error())
	}
	intent := strings.TrimSpace(env.Intent)
	if intent == "" || len(intent) > 32000 {
		return recordError(idFromPath, path, "relay intent is empty or too large")
	}
	if !strings.HasPrefix(intent, core.RelayOperationsIntentPrefix) {
		return recordError(idFromPath, path, "relay/inbox accepts machine-side operations only; ChatGPT owns source code, Git/GitHub, pull requests, CI and GitHub Actions")
	}
	operationIntent := strings.TrimSpace(strings.TrimPrefix(intent, core.RelayOperationsIntentPrefix))
	if operationIntent == "" {
		return recordError(idFromPath, path, "operations relay intent is empty")
	}

	taggedIntent := "[relay:" + env.ID + "] " + core.RelayOperationsIntentPrefix + " " + operationIntent
	taskID, err := delegateRelayTaskMCP(ctx, mcpURL, authFile, taggedIntent, project)
	rec := core.RelayRecord{RelayID: env.ID, Source: "github-git-relay", SourcePath: path, Project: project}
	if err != nil {
		rec.Error = err.Error()
		_ = core.SaveRelayRecord(rec)
		return err
	}
	rec.WorkbenchTaskID = taskID
	if err := core.SaveRelayRecord(rec); err != nil {
		return err
	}
	fmt.Printf("accepted operations relay %s -> task %s (%s)\n", env.ID, taskID, filepath.Base(project))
	return nil
}

func processAnswerPath(ctx context.Context, repo, ref, path, mcpURL, authFile string) error {
	id := strings.TrimSuffix(filepath.Base(path), ".json")
	if !validRelayID(id) {
		return errors.New("invalid relay answer filename")
	}
	rec, ok, err := core.LoadRelayRecord(id)
	if err != nil {
		return err
	}
	if !ok || strings.TrimSpace(rec.WorkbenchTaskID) == "" {
		return nil
	}
	out, err := readRefFile(repo, ref, path, 32<<10)
	if err != nil {
		return err
	}
	var env answerEnvelope
	dec := json.NewDecoder(bytes.NewReader(out))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&env); err != nil {
		return fmt.Errorf("invalid relay answer JSON: %w", err)
	}
	answer := strings.TrimSpace(env.Answer)
	if env.Version != 1 || env.ID != id || answer == "" || len(answer) > 16000 {
		return errors.New("relay answer id/version/content is invalid")
	}
	digest := textDigest(answer)
	if digest == rec.LastAnswerDigest {
		return nil
	}
	task, err := getTaskMCP(ctx, mcpURL, authFile, rec.WorkbenchTaskID)
	if err != nil {
		return err
	}
	if task.Status != core.TaskNeedsAttention {
		return nil
	}
	if err := resolveAttentionMCP(ctx, mcpURL, authFile, rec.WorkbenchTaskID, answer); err != nil {
		return err
	}
	rec.LastAnswerDigest = digest
	if err := core.SaveRelayRecord(rec); err != nil {
		return err
	}
	fmt.Printf("resumed relay %s after human answer\n", id)
	return nil
}

func validRelayID(id string) bool {
	if len(id) < 8 || len(id) > 80 {
		return false
	}
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

func resolveProject(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("project is required")
	}
	if strings.HasPrefix(strings.ToLower(name), core.RunnerProjectPrefix) {
		if !core.IsRunnerProjectReference(name) {
			return "", errors.New("project contains an invalid runner reference")
		}
		return core.ResolveRunnerProject(name)
	}
	if filepath.Base(name) != name || strings.ContainsAny(name, `/\\:`) || name == "." || name == ".." {
		return "", errors.New("project must be one repository directory name or a scoped runner project reference")
	}
	return core.ResolveRunnerProject(name)
}

func delegateRelayTaskMCP(ctx context.Context, url, authFile, intent, project string) (string, error) {
	result, err := callMCP(ctx, url, authFile, "delegate_task", map[string]any{"intent": intent, "project_path": project})
	if err != nil {
		return "", err
	}
	v, _ := result["task_id"].(string)
	v = strings.TrimSpace(v)
	if v == "" {
		return "", errors.New("local MCP returned no task_id")
	}
	return v, nil
}

func getTaskMCP(ctx context.Context, url, authFile, taskID string) (core.Task, error) {
	result, err := callMCP(ctx, url, authFile, "get_task", map[string]any{"task_id": taskID})
	if err != nil {
		return core.Task{}, err
	}
	b, _ := json.Marshal(result)
	var task core.Task
	if err := json.Unmarshal(b, &task); err != nil {
		return core.Task{}, err
	}
	if strings.TrimSpace(task.ID) == "" {
		return core.Task{}, errors.New("local MCP returned an invalid task")
	}
	return task, nil
}

func resolveAttentionMCP(ctx context.Context, url, authFile, taskID, answer string) error {
	_, err := callMCP(ctx, url, authFile, "resolve_attention", map[string]any{"task_id": taskID, "answer": answer})
	return err
}

func callMCP(ctx context.Context, url, authFile, tool string, args map[string]any) (map[string]any, error) {
	auth, err := os.ReadFile(authFile)
	if err != nil {
		return nil, fmt.Errorf("read local MCP auth: %w", err)
	}
	payload, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      tool,
			"arguments": args,
		},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", strings.TrimSpace(string(auth)))
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("local MCP returned HTTP %d", resp.StatusCode)
	}
	var rr rpcResponse
	if err := json.Unmarshal(body, &rr); err != nil {
		return nil, err
	}
	if rr.Error != nil || rr.Result.IsError {
		return nil, fmt.Errorf("local MCP rejected %s", tool)
	}
	return rr.Result.StructuredContent, nil
}

func syncOutbox(ctx context.Context, repo, remote, branch, mcpURL, authFile, resultMode string, publicTransport bool) error {
	records, err := core.ListRelayRecords()
	if err != nil {
		return err
	}
	files := map[string][]byte{}
	for _, rec := range records {
		if !validRelayID(rec.RelayID) {
			continue
		}
		out := outboxEnvelope{Version: 1, ID: rec.RelayID, UpdatedAt: rec.UpdatedAt.UTC().Format(time.RFC3339Nano)}
		if !publicTransport {
			out.WorkbenchTask = rec.WorkbenchTaskID
		}
		if rec.Error != "" {
			out.Status = core.TaskFailed
			if !publicTransport && resultMode == "report" {
				if core.LooksSecret(rec.Error) {
					out.DetailWithheld = true
				} else {
					out.Error = rec.Error
				}
			}
		} else if rec.WorkbenchTaskID != "" {
			task, taskErr := getTaskMCP(ctx, mcpURL, authFile, rec.WorkbenchTaskID)
			if taskErr != nil {
				continue
			}
			out = buildOutbox(rec.RelayID, task, resultMode, publicTransport)
		}
		if out.Status == "" {
			continue
		}
		b, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return err
		}
		b = append(b, '\n')
		files["relay/outbox/"+rec.RelayID+".json"] = b
	}
	return publishOutbox(ctx, repo, remote, branch, files)
}

func buildOutbox(id string, task core.Task, resultMode string, publicTransport bool) outboxEnvelope {
	out := outboxEnvelope{
		Version:   1,
		ID:        id,
		Status:    task.Status,
		UpdatedAt: task.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if !publicTransport {
		out.WorkbenchTask = task.ID
		out.ConsumesWork = task.ConsumesWork
	}
	if !publicTransport && resultMode == "report" {
		detail := strings.Join([]string{task.Output, task.Error, task.AttentionQuestion}, "\n")
		if core.LooksSecret(detail) {
			out.DetailWithheld = true
		} else {
			out.Report = task.Output
			out.Error = task.Error
			out.Attention = task.AttentionQuestion
		}
	}
	return out
}

func publishOutbox(ctx context.Context, repo, remote, branch string, files map[string][]byte) error {
	if len(files) == 0 {
		return nil
	}
	for attempt := 0; attempt < 3; attempt++ {
		if err := fetchRemote(ctx, repo, remote, branch); err != nil {
			return err
		}
		ref := remote + "/" + branch
		tmp, err := os.MkdirTemp("", "workbench-relay-publish-")
		if err != nil {
			return err
		}
		added := exec.Command("git", "-C", repo, "worktree", "add", "--detach", "--quiet", tmp, ref)
		if out, err := added.CombinedOutput(); err != nil {
			_ = os.RemoveAll(tmp)
			return fmt.Errorf("create relay publish worktree: %s", strings.TrimSpace(string(out)))
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
		if out, err := exec.Command("git", "-C", tmp, "add", "relay/outbox").CombinedOutput(); err != nil {
			cleanupWorktree(repo, tmp)
			return fmt.Errorf("stage relay outbox: %s", strings.TrimSpace(string(out)))
		}
		diffCmd := exec.Command("git", "-C", tmp, "diff", "--cached", "--quiet", "--", "relay/outbox")
		diffOut, diffErr := diffCmd.CombinedOutput()
		if diffErr == nil {
			cleanupWorktree(repo, tmp)
			return nil
		}
		if exitErr, ok := diffErr.(*exec.ExitError); !ok || exitErr.ExitCode() != 1 {
			cleanupWorktree(repo, tmp)
			return fmt.Errorf("check staged relay outbox: %s", strings.TrimSpace(string(diffOut)))
		}
		commit := exec.Command("git", "-C", tmp, "-c", "user.name=Workbench Relay", "-c", "user.email=workbench-relay@users.noreply.github.com", "commit", "--quiet", "-m", "relay: update task status")
		if out, err := commit.CombinedOutput(); err != nil {
			cleanupWorktree(repo, tmp)
			return fmt.Errorf("commit relay outbox: %s", strings.TrimSpace(string(out)))
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
			return fmt.Errorf("push relay outbox failed: %s", strings.TrimSpace(string(out)))
		}
		time.Sleep(time.Duration(attempt+1) * 250 * time.Millisecond)
	}
	return nil
}

func cleanupWorktree(repo, dir string) {
	_ = exec.Command("git", "-C", repo, "worktree", "remove", "--force", dir).Run()
	_ = os.RemoveAll(dir)
}

func textDigest(s string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(s)))
	return hex.EncodeToString(sum[:])
}

func recordError(id, path, message string) error {
	_ = core.SaveRelayRecord(core.RelayRecord{RelayID: id, Source: "github-git-relay", SourcePath: path, Error: message})
	return errors.New(message)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "workbench-relay:", err)
	os.Exit(1)
}
