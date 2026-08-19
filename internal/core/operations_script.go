package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
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
	Commit         string   `json:"commit,omitempty"`
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

// operationsOriginAllowed is a narrow test seam. Production accepts only a
// credential-free github.com origin. Tests can replace this predicate while
// exercising the fetch path against a local bare repository.
var operationsOriginAllowed = isApprovedOperationsOrigin

// RunOperationsScript executes one committed repository operations script
// without accepting arbitrary shell text. The script must live under
// scripts/ops/, be a regular Git blob, and end in .sh. By default Workbench
// executes the exact local HEAD version. When Commit is supplied, it must be a
// full SHA currently advertised by one of the configured github.com origin's
// branch heads. Workbench fetches that exact branch into a disposable temporary
// repository and never switches, resets, pulls, merges or otherwise changes the
// registered developer checkout. In both modes a second detached disposable
// worktree is created before executing `bash --noprofile --norc <script>
// <literal argv...>` so dirty working-tree changes cannot alter the code that
// runs.
func RunOperationsScript(ctx context.Context, project string, req OperationsScriptRequest) (OperationsScriptResult, error) {
	rel, err := validateOperationsScriptRequest(req)
	if err != nil {
		return OperationsScriptResult{}, err
	}
	requestedCommit, err := normalizeOperationsCommit(req.Commit)
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
	sourceRoot, commit, transport, cleanupSource, err := prepareOperationsScriptSource(ctx, root, requestedCommit)
	if err != nil {
		return result, err
	}
	defer cleanupSource()
	result.Commit = commit
	result.Transport = transport

	mode, objectType, err := operationsScriptTreeEntry(ctx, sourceRoot, commit, rel)
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
	add := exec.CommandContext(ctx, "git", "-C", sourceRoot, "worktree", "add", "--detach", "--quiet", checkout, commit)
	configureChildProcess(add, false)
	if out, addErr := add.CombinedOutput(); addErr != nil {
		return result, fmt.Errorf("create disposable operations worktree: %s", strings.TrimSpace(string(out)))
	}
	defer func() {
		cleanup := exec.Command("git", "-C", sourceRoot, "worktree", "remove", "--force", checkout)
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
	if _, err := normalizeOperationsCommit(req.Commit); err != nil {
		return "", err
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

func normalizeOperationsCommit(raw string) (string, error) {
	commit := strings.ToLower(strings.TrimSpace(raw))
	if commit == "" {
		return "", nil
	}
	if len(commit) != 40 {
		return "", errors.New("operations script commit must be a full 40-character Git SHA")
	}
	for _, ch := range commit {
		if (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') {
			continue
		}
		return "", errors.New("operations script commit must be a full hexadecimal Git SHA")
	}
	return commit, nil
}

func prepareOperationsScriptSource(ctx context.Context, root, requestedCommit string) (string, string, string, func(), error) {
	if requestedCommit == "" {
		commit, err := operationsGitOutput(ctx, root, "rev-parse", "HEAD")
		if err != nil {
			return "", "", "", func() {}, fmt.Errorf("resolve operations script commit: %w", err)
		}
		commit, err = normalizeOperationsCommit(commit)
		if err != nil {
			return "", "", "", func() {}, errors.New("operations script commit could not be resolved to a full Git SHA")
		}
		return root, commit, "git-worktree-bash", func() {}, nil
	}

	originURL, err := operationsGitOutput(ctx, root, "remote", "get-url", "origin")
	if err != nil {
		return "", "", "", func() {}, errors.New("operations script remote commit requires the project's configured origin")
	}
	originURL = strings.TrimSpace(originURL)
	if !operationsOriginAllowed(originURL) {
		return "", "", "", func() {}, errors.New("operations script remote commit requires a credential-free github.com origin")
	}
	ref, err := operationsAdvertisedOriginHead(ctx, root, requestedCommit)
	if err != nil {
		return "", "", "", func() {}, err
	}

	base, err := os.MkdirTemp("", "workbench-operations-origin-")
	if err != nil {
		return "", "", "", func() {}, err
	}
	cleanup := func() { _ = os.RemoveAll(base) }
	source := filepath.Join(base, "source")
	if err := os.MkdirAll(source, 0o700); err != nil {
		cleanup()
		return "", "", "", func() {}, err
	}
	if _, err := operationsGitOutput(ctx, source, "init", "--quiet"); err != nil {
		cleanup()
		return "", "", "", func() {}, errors.New("initialise disposable operations source repository")
	}
	if _, err := operationsGitOutput(ctx, source, "remote", "add", "origin", originURL); err != nil {
		cleanup()
		return "", "", "", func() {}, errors.New("configure disposable operations source origin")
	}
	if _, err := operationsGitNetworkOutput(ctx, source, "fetch", "--quiet", "--depth=1", "--no-tags", "origin", ref); err != nil {
		cleanup()
		return "", "", "", func() {}, fmt.Errorf("fetch exact operations origin commit: %w", err)
	}
	resolved, err := operationsGitOutput(ctx, source, "rev-parse", "FETCH_HEAD")
	if err != nil {
		cleanup()
		return "", "", "", func() {}, errors.New("resolve fetched operations origin commit")
	}
	resolved, err = normalizeOperationsCommit(resolved)
	if err != nil || resolved != requestedCommit {
		cleanup()
		return "", "", "", func() {}, errors.New("fetched operations origin ref did not resolve to the requested commit")
	}
	return source, requestedCommit, "github-origin-commit-worktree-bash", cleanup, nil
}

func operationsAdvertisedOriginHead(ctx context.Context, root, commit string) (string, error) {
	out, err := operationsGitNetworkOutput(ctx, root, "ls-remote", "--heads", "origin")
	if err != nil {
		return "", errors.New("inspect configured operations origin heads")
	}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) != 2 || !strings.EqualFold(fields[0], commit) || !strings.HasPrefix(fields[1], "refs/heads/") {
			continue
		}
		return fields[1], nil
	}
	return "", errors.New("requested operations script commit is not currently advertised by a configured origin branch head")
}

func isApprovedOperationsOrigin(raw string) bool {
	remote := strings.TrimSpace(raw)
	if remote == "" || strings.ContainsAny(remote, "\x00\r\n") {
		return false
	}
	if strings.HasPrefix(remote, "git@github.com:") {
		return validOperationsGitHubRepoPath(strings.TrimPrefix(remote, "git@github.com:"))
	}
	u, err := url.Parse(remote)
	if err != nil || !strings.EqualFold(u.Hostname(), "github.com") || u.Port() != "" || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		if u.User != nil {
			return false
		}
	case "ssh":
		if u.User == nil || u.User.Username() != "git" {
			return false
		}
		if _, hasPassword := u.User.Password(); hasPassword {
			return false
		}
	default:
		return false
	}
	return validOperationsGitHubRepoPath(u.Path)
}

func validOperationsGitHubRepoPath(raw string) bool {
	repoPath := strings.Trim(strings.TrimSpace(raw), "/")
	repoPath = strings.TrimSuffix(repoPath, ".git")
	parts := strings.Split(repoPath, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || parts[0] == "." || parts[0] == ".." || parts[1] == "." || parts[1] == ".." {
		return false
	}
	for _, part := range parts {
		for _, ch := range part {
			if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' || ch == '.' {
				continue
			}
			return false
		}
	}
	return true
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

func operationsGitNetworkOutput(ctx context.Context, root string, args ...string) (string, error) {
	gitCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(gitCtx, "git", append([]string{"-C", root}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_PAGER=cat", "PAGER=cat")
	configureChildProcess(cmd, false)
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		return text, err
	}
	return text, nil
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
