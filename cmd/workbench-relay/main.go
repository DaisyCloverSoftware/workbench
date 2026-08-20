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

const relayVersion = "0.9.36"

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
	smokeOnly := flag.Bool("smoke-only", false, "validate relay configuration without consuming relay work")
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
	if *smokeOnly {
		fmt.Println("smoke-only: configuration validated; relay queue was not polled")
		return
	}

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

	taggedIntent := "[relay:" + env.ID + "] " + operationIntent
	taskID, err := delegateOperationMCP(ctx, mcpURL, authFile, taggedIntent, project)
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
	if rec, ok, err := core.LoadRelayRecord(id); err == nil && ok && rec.AnswerApplied {
		return nil
	}
	out, err := readRefFile(repo, ref, path, 64<<10)
	if err != nil {
		return err
	}
	var ans answerEnvelope
	dec := json.NewDecoder(bytes.NewReader(out))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&ans); err != nil {
		return err
	}
	if ans.Version != 1 || ans.ID != id || strings.TrimSpace(ans.Answer) == "" || len(ans.Answer) > 16000 {
		return errors.New("relay answer id/version/answer invalid")
	}
	rec, ok, err := core.LoadRelayRecord(id)
	if err != nil || !ok || rec.WorkbenchTaskID == "" {
		return errors.New("relay answer has no matching Workbench task")
	}
	if _, err := resolveAttentionMCP(ctx, mcpURL, authFile, rec.WorkbenchTaskID, ans.Answer); err != nil {
		return err
	}
	rec.AnswerApplied = true
	if err := core.SaveRelayRecord(rec); err != nil {
		return err
	}
	fmt.Printf("applied relay answer %s -> task %s\n", id, rec.WorkbenchTaskID)
	return nil
}

func recordError(id, sourcePath, msg string) error {
	rec := core.RelayRecord{RelayID: id, Source: "github-git-relay", SourcePath: sourcePath, Error: msg}
	_ = core.SaveRelayRecord(rec)
	return errors.New(msg)
}

func syncOutbox(ctx context.Context, repo, remote, branch, mcpURL, authFile, resultMode string, publicTransport bool) error {
	records, err := core.ListRelayRecords()
	if err != nil {
		return err
	}
	for _, rec := range records {
		out := outboxEnvelope{Version: 1, ID: rec.RelayID, WorkbenchTask: rec.WorkbenchTaskID, UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
		if rec.WorkbenchTaskID == "" {
			out.Status = core.TaskFailed
			out.Error = publicSafe(rec.Error, publicTransport)
		} else {
			task, err := getTaskMCP(ctx, mcpURL, authFile, rec.WorkbenchTaskID)
			if err != nil {
				out.Status = core.TaskFailed
				out.Error = publicSafe(err.Error(), publicTransport)
			} else {
				out.Status = task.Status
				out.ConsumesWork = task.ConsumesWork
				out.UpdatedAt = task.UpdatedAt.Format(time.RFC3339Nano)
				if resultMode == "report" && !publicTransport {
					out.Report = sanitizeReport(task.Output)
					out.Error = sanitizeReport(task.Error)
					out.Attention = sanitizeReport(task.AttentionQuestion)
				}
			}
		}
		if publicTransport || resultMode != "report" {
			out.Report = ""
			out.Attention = ""
			if out.Error != "" {
				out.Error = publicSafe(out.Error, true)
			}
		}
		if core.LooksSecret(out.Report) || core.LooksSecret(out.Attention) || core.LooksSecret(out.Error) {
			out.Report = ""
			out.Attention = ""
			out.Error = ""
			out.DetailWithheld = true
		}
		if err := publishOutbox(ctx, repo, remote, branch, out); err != nil {
			return err
		}
	}
	return nil
}

func publishOutbox(ctx context.Context, repo, remote, branch string, out outboxEnvelope) error {
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	path := "relay/outbox/" + out.ID + ".json"
	current, _ := os.ReadFile(filepath.Join(repo, path))
	if bytes.Equal(current, b) {
		return nil
	}
	if err := os.MkdirAll(filepath.Join(repo, "relay", "outbox"), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(repo, path), b, 0o644); err != nil {
		return err
	}
	return commitAndPush(ctx, repo, remote, branch, "relay: update task status", path)
}

func commitAndPush(ctx context.Context, repo, remote, branch, message string, paths ...string) error {
	if len(paths) == 0 {
		return nil
	}
	for _, p := range paths {
		cmd := exec.CommandContext(ctx, "git", "-C", repo, "add", "--", p)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git add failed: %s", strings.TrimSpace(string(out)))
		}
	}
	cmd := exec.CommandContext(ctx, "git", "-C", repo, "diff", "--cached", "--quiet", "--")
	if err := cmd.Run(); err == nil {
		return nil
	}
	cmd = exec.CommandContext(ctx, "git", "-C", repo, "-c", "user.name=Workbench Relay", "-c", "user.email=workbench-relay@users.noreply.github.com", "commit", "-m", message, "--quiet")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit failed: %s", strings.TrimSpace(string(out)))
	}
	pushCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	cmd = exec.CommandContext(pushCtx, "git", "-C", repo, "push", "--quiet", remote, "HEAD:"+branch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func sanitizeReport(s string) string {
	s = strings.TrimSpace(s)
	const max = 12000
	if len(s) > max {
		s = s[:max] + "\n… truncated by Workbench relay …"
	}
	return s
}

func publicSafe(s string, public bool) string {
	if !public || s == "" {
		return sanitizeReport(s)
	}
	return "Workbench task failed; inspect the private runner state for details."
}

func callMCP(ctx context.Context, mcpURL, authFile, tool string, args map[string]any) (map[string]any, error) {
	if strings.TrimSpace(mcpURL) == "" {
		return nil, errors.New("local MCP URL is empty")
	}
	if strings.TrimSpace(authFile) == "" {
		return nil, errors.New("MCP auth file is empty")
	}
	authBytes, err := os.ReadFile(authFile)
	if err != nil {
		return nil, fmt.Errorf("read MCP auth file: %w", err)
	}
	auth := strings.TrimSpace(string(authBytes))
	if auth == "" || strings.ContainsAny(auth, "\r\n") {
		return nil, errors.New("MCP auth file contains an invalid Authorization value")
	}
	body := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      tool,
			"arguments": args,
		},
	}
	payload, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, mcpURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", auth)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MCP returned HTTP %d", resp.StatusCode)
	}
	limited := io.LimitReader(resp.Body, 2<<20)
	var rpc rpcResponse
	if err := json.NewDecoder(limited).Decode(&rpc); err != nil {
		return nil, err
	}
	if rpc.Error != nil || rpc.Result.IsError {
		return nil, errors.New("Workbench MCP tool call failed")
	}
	return rpc.Result.StructuredContent, nil
}

func delegateOperationMCP(ctx context.Context, mcpURL, authFile, intent, project string) (string, error) {
	result, err := callMCP(ctx, mcpURL, authFile, "delegate_operation", map[string]any{"intent": intent, "project_path": project})
	if err != nil {
		return "", err
	}
	id, _ := result["task_id"].(string)
	if id == "" {
		return "", errors.New("Workbench did not return a task id")
	}
	return id, nil
}

func resolveAttentionMCP(ctx context.Context, mcpURL, authFile, taskID, answer string) (map[string]any, error) {
	return callMCP(ctx, mcpURL, authFile, "resolve_attention", map[string]any{"task_id": taskID, "answer": answer})
}

func getTaskMCP(ctx context.Context, mcpURL, authFile, taskID string) (core.Task, error) {
	result, err := callMCP(ctx, mcpURL, authFile, "get_task", map[string]any{"task_id": taskID})
	if err != nil {
		return core.Task{}, err
	}
	raw, err := json.Marshal(result["task"])
	if err != nil {
		return core.Task{}, err
	}
	var task core.Task
	if err := json.Unmarshal(raw, &task); err != nil {
		return core.Task{}, err
	}
	return task, nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "workbench-relay:", err)
	os.Exit(1)
}
