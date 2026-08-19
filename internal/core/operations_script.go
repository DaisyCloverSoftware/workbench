package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	defaultOperationsScriptTimeout = 15 * time.Minute
	maxOperationsScriptTimeout     = 60 * time.Minute
	maxOperationsScriptArgs        = 64
	maxOperationsScriptArgBytes    = 2048
	maxOperationsScriptOutputBytes = 2 << 20
)

type OperationsScriptRequest struct {
	Path           string   `json:"path"`
	Args           []string `json:"args,omitempty"`
	TimeoutSeconds int      `json:"timeout_seconds,omitempty"`
}

type OperationsScriptResult struct {
	Path         string   `json:"path"`
	Args         []string `json:"args,omitempty"`
	Commit       string   `json:"commit"`
	ScriptSHA256 string   `json:"script_sha256"`
	Output       string   `json:"output,omitempty"`
	ExitCode     int      `json:"exit_code"`
	Truncated    bool     `json:"truncated,omitempty"`
	Transport    string   `json:"transport"`
}

// RunOperationsScript executes one committed repository operations script
// without accepting arbitrary shell text. The script must live under
// scripts/ops/, be a regular Git blob at HEAD, and end in .sh. Workbench creates
// a detached disposable worktree at the exact commit before executing it with
// `bash --noprofile --norc <script> <literal argv...>`. Dirty working-tree
// changes therefore cannot alter the code that runs.
func RunOperationsScript(ctx context.Context, project string, req OperationsScriptRequest) (OperationsScriptResult, error) {
	rel, err := validateOperationsScriptRequest(req)
	if err != nil {
		return OperationsScriptResult{}, err
	}
	result := OperationsScriptResult{
		Path:      rel,
		Args:      append([]string(nil), req.Args...),
		ExitCode:  0,
		Transport: "git-worktree-bash",
	}

	root, err := operationsScriptGitRoot(ctx, project)
	if err != nil {
		return result, err
	}
	commit, err := operationsGitOutput(ctx, root, "rev-parse", "HEAD")
	if err != nil {
		return result, fmt.Errorf("resolve operations script commit: %w", err)
	}
	commit = strings.TrimSpace(commit)
	if len(commit) != 40 {
		return result, errors.New("operations script commit could not be resolved to a full Git SHA")
	}
	result.Commit = commit

	mode, objectType, err := operationsScriptTreeEntry(ctx, root, commit, rel)
	if err != nil {
		return result, err
	}
	if objectType != "blob" || mode == "120000" {
		return result, errors.New("operations script must be a regular Git-tracked file, not a symlink or non-blob entry")
	}

	base, err := os.MkdirTemp("", "workbench-operations-script-")
	if err != nil {
		return result, err
	}
	defer os.RemoveAll(base)
	checkout := filepath.Join(base, "checkout")
	add := exec.CommandContext(ctx, "git", "-C", root, "worktree", "add", "--detach", "--quiet", checkout, commit)
	configureChildProcess(add, false)
	if out, addErr := add.CombinedOutput(); addErr != nil {
		return result, fmt.Errorf("create disposable operations worktree: %s", strings.TrimSpace(string(out)))
	}
	defer func() {
		cleanup := exec.Command("git", "-C", root, "worktree", "remove", "--force", checkout)
		configureChildProcess(cleanup, false)
		_ = cleanup.Run()
	}()

	scriptPath := filepath.Join(checkout, filepath.FromSlash(rel))
	if !withinRoot(checkout, scriptPath) {
		return result, errors.New("operations script resolved outside disposable repository worktree")
	}
	info, err := os.Lstat(scriptPath)
	if err != nil {
		return result, errors.New("operations script is missing from disposable committed worktree")
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return result, errors.New("operations script must resolve to a regular file")
	}
	body, err := os.ReadFile(scriptPath)
	if err != nil {
		return result, err
	}
	digest := sha256.Sum256(body)
	result.ScriptSHA256 = hex.EncodeToString(digest[:])

	timeout := defaultOperationsScriptTimeout
	if req.TimeoutSeconds > 0 {
		timeout = time.Duration(req.TimeoutSeconds) * time.Second
	}
	if timeout > maxOperationsScriptTimeout {
		timeout = maxOperationsScriptTimeout
	}
	if timeout < time.Second {
		timeout = time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	argv := []string{"--noprofile", "--norc", scriptPath}
	argv = append(argv, req.Args...)
	cmd := exec.CommandContext(runCtx, "bash", argv...)
	cmd.Dir = checkout
	cmd.Env = append(os.Environ(),
		"WORKBENCH_OPERATION_SCRIPT=1",
		"WORKBENCH_OPERATION_COMMIT="+commit,
		"PAGER=cat",
		"GIT_PAGER=cat",
		"SYSTEMD_PAGER=cat",
		"SYSTEMD_COLORS=0",
		"NO_COLOR=1",
	)
	configureChildProcess(cmd, false)
	output := &limitedCapture{limit: maxOperationsScriptOutputBytes}
	cmd.Stdout, cmd.Stderr = output, output
	runErr := cmd.Run()
	out := strings.TrimSpace(output.String())
	result.Truncated = output.exceeded
	if output.exceeded {
		out += "\n… output truncated by Workbench …"
	}
	if LooksSecret(out) {
		result.Output = "[withheld by Workbench: operations script output resembled secret material]"
		result.ExitCode = operationsExitCode(runErr)
		return result, errors.New("operations script output was withheld because it resembled secret material")
	}
	result.Output = out
	result.ExitCode = operationsExitCode(runErr)
	if runErr != nil {
		if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
			return result, errors.New("operations script timed out")
		}
		return result, fmt.Errorf("operations script failed: %w", runErr)
	}
	return result, nil
}

func validateOperationsScriptRequest(req OperationsScriptRequest) (string, error) {
	raw := strings.TrimSpace(strings.ReplaceAll(req.Path, "\\", "/"))
	if raw == "" || strings.ContainsAny(raw, "\x00\r\n") {
		return "", errors.New("operations script path is required and must be one bounded relative path")
	}
	rel := path.Clean(raw)
	if rel == "." || rel != raw || strings.HasPrefix(rel, "/") || rel == ".." || strings.HasPrefix(rel, "../") {
		return "", errors.New("operations script path must be canonical and relative")
	}
	if !strings.HasPrefix(rel, "scripts/ops/") || strings.EqualFold(rel, "scripts/ops/") || !strings.HasSuffix(strings.ToLower(rel), ".sh") {
		return "", errors.New("operations script must be a .sh file beneath scripts/ops/")
	}
	if len(req.Args) > maxOperationsScriptArgs {
		return "", errors.New("operations script has too many arguments")
	}
	for _, arg := range req.Args {
		if len(arg) > maxOperationsScriptArgBytes || strings.ContainsAny(arg, "\x00\r\n") {
			return "", errors.New("operations script argument is invalid or too large")
		}
	}
	if LooksSecret(strings.Join(req.Args, "\n")) {
		return "", errors.New("operations script arguments appear to contain secret material")
	}
	return rel, nil
}

func operationsScriptGitRoot(ctx context.Context, project string) (string, error) {
	abs, err := filepath.Abs(project)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", errors.New("operations project path could not be canonicalised")
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return "", errors.New("operations project path is not a directory")
	}
	root, err := operationsGitOutput(ctx, resolved, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", errors.New("operations script project must be a Git repository")
	}
	root = strings.TrimSpace(root)
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", errors.New("operations Git root could not be canonicalised")
	}
	if filepath.Clean(canonicalRoot) != filepath.Clean(resolved) {
		return "", errors.New("operations script project must be the Git repository root")
	}
	return canonicalRoot, nil
}

func operationsScriptTreeEntry(ctx context.Context, root, commit, rel string) (string, string, error) {
	out, err := operationsGitOutput(ctx, root, "ls-tree", commit, "--", rel)
	if err != nil {
		return "", "", err
	}
	line := strings.TrimSpace(out)
	if line == "" {
		return "", "", errors.New("operations script is not tracked at the selected commit")
	}
	fields := strings.Fields(line)
	if len(fields) < 4 {
		return "", "", errors.New("operations script Git tree entry is malformed")
	}
	return fields[0], fields[1], nil
}

func operationsGitOutput(ctx context.Context, root string, args ...string) (string, error) {
	gitCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(gitCtx, "git", append([]string{"-C", root}, args...)...)
	configureChildProcess(cmd, false)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(out)), err
	}
	return strings.TrimSpace(string(out)), nil
}

func operationsExitCode(err error) int {
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}
	return -1
}

// Keep the implementation portable but make the expected execution dependency
// explicit to callers/tests on platforms where bash may not be installed.
func OperationsScriptSupported() bool {
	if runtime.GOOS == "windows" {
		_, err := exec.LookPath("bash")
		return err == nil
	}
	_, err := exec.LookPath("bash")
	return err == nil
}
