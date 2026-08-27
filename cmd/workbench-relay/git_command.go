package main

import (
	"context"
	"os"
	"os/exec"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

const (
	relayGitLocalTimeout   = 30 * time.Second
	relayGitNetworkTimeout = 45 * time.Second
	relayGitCleanupTimeout = 10 * time.Second
)

func relayGitCombinedOutput(ctx context.Context, timeout time.Duration, repo string, args ...string) ([]byte, error) {
	cmd, cancel := newRelayGitCommand(ctx, timeout, repo, args...)
	defer cancel()
	return cmd.CombinedOutput()
}

func relayGitOutput(ctx context.Context, timeout time.Duration, repo string, args ...string) ([]byte, error) {
	cmd, cancel := newRelayGitCommand(ctx, timeout, repo, args...)
	defer cancel()
	return cmd.Output()
}

func relayGitRun(ctx context.Context, timeout time.Duration, repo string, args ...string) error {
	cmd, cancel := newRelayGitCommand(ctx, timeout, repo, args...)
	defer cancel()
	return cmd.Run()
}

func newRelayGitCommand(ctx context.Context, timeout time.Duration, repo string, args ...string) (*exec.Cmd, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if timeout <= 0 {
		timeout = relayGitLocalTimeout
	}
	gitCtx, cancel := context.WithTimeout(ctx, timeout)
	argv := append([]string{"-C", repo}, args...)
	cmd := exec.CommandContext(gitCtx, "git", argv...)
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_PAGER=cat",
		"PAGER=cat",
	)
	core.ConfigureBoundedChildProcess(cmd)
	return cmd, cancel
}
