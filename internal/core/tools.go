package core

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var safeCommandPrefixes = []string{
	"go test", "go vet", "go build",
	"npm test", "npm run test", "npm run build", "npm run lint",
	"pnpm test", "pnpm run test", "pnpm run build", "pnpm run lint",
	"yarn test", "yarn build", "yarn lint",
	"pytest", "python -m pytest", "python3 -m pytest",
	"dotnet test", "dotnet build",
	"cargo test", "cargo check", "cargo build",
}

var dangerousFragments = []string{
	"&&", "||", ";", "|", ">", "<", "`", "$(`, "\n", "\r",
	" push", " deploy", " publish", " release", " rm ", " rmdir ", " del ", " format ", " shutdown", " reboot", " curl ", " wget ", " invoke-webrequest",
}

var unsafeGitInspectionFragments = []string{
	"--output", "--ext-diff", "--textconv", "--exec", "--paginate", "--config-env",
}

func IsSafeCommand(line string) bool {
	s := strings.TrimSpace(strings.ToLower(line))
	if s == "" {
		return false
	}
	for _, bad := range dangerousFragments {
		if strings.Contains(s, bad) {
			return false
		}
	}
	if strings.HasPrefix(s, "git ") {
		return isSafeGitInspection(s)
	}
	for _, p := range safeCommandPrefixes {
		if s == p || strings.HasPrefix(s, p+" ") {
			return true
		}
	}
	return false
}

func isSafeGitInspection(s string) bool {
	allowed := false
	for _, prefix := range []string{"git status", "git diff", "git log", "git show"} {
		if s == prefix || strings.HasPrefix(s, prefix+" ") {
			allowed = true
			break
		}
	}
	if !allowed {
		// Branch inspection is useful, but the generic `git branch` command also
		// deletes, renames and edits branches. Permit only its explicit read form.
		return s == "git branch --show-current"
	}
	for _, bad := range unsafeGitInspectionFragments {
		if strings.Contains(s, bad) {
			return false
		}
	}
	return true
}

func RunSafeCommand(ctx context.Context, project, line string) (string, error) {
	if !IsSafeCommand(line) {
		return "", errors.New("command rejected by Workbench safe-command policy")
	}
	abs, err := filepath.Abs(project)
	if err != nil {
		return "", err
	}
	if st, err := os.Stat(abs); err != nil || !st.IsDir() {
		return "", fmt.Errorf("project path is not a directory: %s", abs)
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Minute)
	defer cancel()
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd.exe", "/D", "/S", "/C", line)
	} else {
		cmd = exec.CommandContext(ctx, "/bin/sh", "-lc", line)
	}
	cmd.Dir = abs
	cmd.Env = append(os.Environ(), "GIT_PAGER=cat", "PAGER=cat", "GIT_OPTIONAL_LOCKS=0")
	configureChildProcess(cmd, false)
	var buf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &buf
	err = cmd.Run()
	out := strings.TrimSpace(buf.String())
	if err != nil {
		return out, fmt.Errorf("command failed: %w", err)
	}
	return out, nil
}

func ApplyPatch(ctx context.Context, project, patch string) (string, error) {
	if strings.TrimSpace(patch) == "" {
		return "", errors.New("patch is empty")
	}
	if LooksSecret(patch) {
		return "", errors.New("patch appears to contain a secret; Workbench will not pass it into source control")
	}
	abs, err := filepath.Abs(project)
	if err != nil {
		return "", err
	}
	if st, err := os.Stat(abs); err != nil || !st.IsDir() {
		return "", fmt.Errorf("project path is not a directory: %s", abs)
	}
	f, err := os.CreateTemp("", "workbench-*.patch")
	if err != nil {
		return "", err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err := f.WriteString(patch); err != nil {
		_ = f.Close()
		return "", err
	}
	_ = f.Close()

	check := exec.CommandContext(ctx, "git", "apply", "--check", name)
	check.Dir = abs
	configureChildProcess(check, false)
	var checkBuf bytes.Buffer
	check.Stdout, check.Stderr = &checkBuf, &checkBuf
	if err := check.Run(); err != nil {
		return strings.TrimSpace(checkBuf.String()), fmt.Errorf("patch check failed: %w", err)
	}
	apply := exec.CommandContext(ctx, "git", "apply", name)
	apply.Dir = abs
	configureChildProcess(apply, false)
	var buf bytes.Buffer
	apply.Stdout, apply.Stderr = &buf, &buf
	if err := apply.Run(); err != nil {
		return strings.TrimSpace(buf.String()), err
	}
	out, _ := RunSafeCommand(ctx, abs, "git diff --stat")
	if strings.TrimSpace(out) == "" {
		out = "Patch applied successfully."
	}
	return out, nil
}
