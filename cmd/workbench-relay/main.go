package main

import (
	"bytes"
	"context"
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

const relayVersion = "0.5.0-dev"

type envelope struct {
	Version   int    `json:"version"`
	ID        string `json:"id"`
	Project   string `json:"project"`
	Intent    string `json:"intent"`
	CreatedAt string `json:"created_at,omitempty"`
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
	remote := flag.String("remote", "origin", "git remote containing relay/inbox")
	branch := flag.String("branch", "main", "remote branch containing relay tasks")
	interval := flag.Duration("interval", 10*time.Second, "poll interval")
	mcpURL := flag.String("mcp-url", "http://127.0.0.1:8765/mcp", "local Workbench MCP endpoint")
	authFile := flag.String("auth-file", defaultAuthFile(), "file containing local MCP Authorization value")
	once := flag.Bool("once", false, "poll once then exit")
	flag.Parse()

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

	fmt.Printf("Workbench GitHub Relay %s\n", relayVersion)
	fmt.Printf("relay repo: %s\n", absRepo)
	fmt.Printf("source ref: %s/%s:relay/inbox\n", *remote, *branch)
	fmt.Printf("local MCP: %s\n", *mcpURL)

	for {
		if err := poll(context.Background(), absRepo, *remote, *branch, *mcpURL, *authFile); err != nil {
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

func poll(ctx context.Context, repo, remote, branch, mcpURL, authFile string) error {
	fetchCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(fetchCtx, "git", "-C", repo, "fetch", "--quiet", remote, branch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git fetch failed: %s", strings.TrimSpace(string(out)))
	}
	ref := remote + "/" + branch
	paths, err := inboxPaths(repo, ref)
	if err != nil {
		return err
	}
	for _, path := range paths {
		if err := processPath(ctx, repo, ref, path, mcpURL, authFile); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
		}
	}
	return nil
}

func inboxPaths(repo, ref string) ([]string, error) {
	cmd := exec.Command("git", "-C", repo, "ls-tree", "-r", "--name-only", ref, "--", "relay/inbox")
	out, err := cmd.Output()
	if err != nil {
		return nil, nil
	}
	var paths []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "relay/inbox/") && strings.HasSuffix(line, ".json") {
			paths = append(paths, line)
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func processPath(ctx context.Context, repo, ref, path, mcpURL, authFile string) error {
	idFromPath := strings.TrimSuffix(filepath.Base(path), ".json")
	if !validRelayID(idFromPath) {
		return errors.New("invalid relay filename")
	}
	if rec, ok, err := core.LoadRelayRecord(idFromPath); err == nil && ok && (rec.WorkbenchTaskID != "" || rec.Error != "") {
		return nil
	}

	cmd := exec.Command("git", "-C", repo, "show", ref+":"+path)
	out, err := cmd.Output()
	if err != nil {
		return err
	}
	if len(out) > 64<<10 {
		return recordError(idFromPath, path, "relay envelope exceeds 64 KiB")
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

	// The relay marker is deliberately part of the durable task intent so a
	// read-only ChatGPT MCP connection can find the resulting task with
	// list_tasks/get_task even when custom MCP write tools are unavailable.
	taggedIntent := "[relay:" + env.ID + "] " + intent
	taskID, err := delegateMCP(ctx, mcpURL, authFile, taggedIntent, project)
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
	fmt.Printf("accepted relay %s -> task %s (%s)\n", env.ID, taskID, filepath.Base(project))
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
	if name == "" || filepath.Base(name) != name || strings.ContainsAny(name, `/\\`) || name == "." || name == ".." {
		return "", errors.New("project must be one repository directory name under WORKBENCH_RUNNER_ROOT")
	}
	root := strings.TrimSpace(os.Getenv("WORKBENCH_RUNNER_ROOT"))
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		root = filepath.Join(home, "src")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	project := filepath.Join(root, name)
	st, err := os.Stat(project)
	if err != nil || !st.IsDir() {
		return "", fmt.Errorf("relay project not found under runner root: %s", project)
	}
	return project, nil
}

func delegateMCP(ctx context.Context, url, authFile, intent, project string) (string, error) {
	auth, err := os.ReadFile(authFile)
	if err != nil {
		return "", fmt.Errorf("read local MCP auth: %w", err)
	}
	payload, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "delegate_task",
			"arguments": map[string]any{
				"intent":       intent,
				"project_path": project,
			},
		},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", strings.TrimSpace(string(auth)))
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("local MCP returned HTTP %d", resp.StatusCode)
	}
	var rr rpcResponse
	if err := json.Unmarshal(body, &rr); err != nil {
		return "", err
	}
	if rr.Error != nil || rr.Result.IsError {
		return "", fmt.Errorf("local MCP rejected relay task")
	}
	v, _ := rr.Result.StructuredContent["task_id"].(string)
	v = strings.TrimSpace(v)
	if v == "" {
		return "", errors.New("local MCP returned no task_id")
	}
	return v, nil
}

func recordError(id, path, message string) error {
	_ = core.SaveRelayRecord(core.RelayRecord{RelayID: id, Source: "github-git-relay", SourcePath: path, Error: message})
	return errors.New(message)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "workbench-relay:", err)
	os.Exit(1)
}
