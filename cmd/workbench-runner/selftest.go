package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

// selftest verifies Workbench's own runner control plane without depending on
// any external AI provider being authenticated, entitled, online, or below its
// quota. It proves the isolated-worktree -> Workbench review-commit lifecycle
// and verifies that the source checkout remains clean.
func selftest() error {
	dir, keep, err := createSelftestRepository(".workbench-selftest-")
	if err != nil {
		return err
	}
	if !keep {
		defer os.RemoveAll(dir)
	}

	marker := fmt.Sprintf("WORKBENCH_SELFTEST_OK_%d", time.Now().UTC().UnixNano())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	ws, err := core.CreateTaskWorkspace(ctx, dir, "runner-control-plane-selftest")
	if err != nil {
		return fmt.Errorf("create isolated task workspace: %w", err)
	}
	if err := os.WriteFile(filepath.Join(ws.Workspace, "WORKBENCH_SELFTEST_OK.txt"), []byte(marker+"\n"), 0o644); err != nil {
		return fmt.Errorf("write isolated marker: %w", err)
	}
	review, err := core.FinalizeTaskWorkspace(ctx, ws)
	if err != nil {
		return fmt.Errorf("finalize isolated task workspace: %w", err)
	}
	if !review.Changed || review.Published || strings.TrimSpace(review.Branch) == "" || strings.TrimSpace(review.Commit) == "" {
		return fmt.Errorf("unexpected review result: changed=%t published=%t branch=%q commit=%q", review.Changed, review.Published, review.Branch, review.Commit)
	}
	if status, err := runSelftestGit(ctx, dir, "status", "--porcelain"); err != nil {
		return err
	} else if strings.TrimSpace(status) != "" {
		return fmt.Errorf("source checkout was dirtied by isolated finalization: %q", strings.TrimSpace(status))
	}
	if _, err := os.Stat(filepath.Join(dir, "WORKBENCH_SELFTEST_OK.txt")); !errors.Is(err, os.ErrNotExist) {
		if err == nil {
			return errors.New("isolated marker leaked into the source checkout")
		}
		return fmt.Errorf("inspect source marker: %w", err)
	}
	got, err := runSelftestGit(ctx, dir, "show", review.Commit+":WORKBENCH_SELFTEST_OK.txt")
	if err != nil {
		return fmt.Errorf("read prepared review marker: %w", err)
	}
	if strings.TrimSpace(got) != marker {
		return fmt.Errorf("review marker content mismatch: got %q, want %q", strings.TrimSpace(got), marker)
	}
	if _, ok, err := core.OpenTaskWorkspace(dir, "runner-control-plane-selftest"); err != nil || ok {
		return fmt.Errorf("successful task workspace was not cleaned up: present=%t err=%v", ok, err)
	}

	fmt.Println("Workbench deterministic self-test")
	fmt.Println("SELFTEST PASSED")
	fmt.Println("  Git repository baseline: ok")
	fmt.Println("  isolated task worktree: ok")
	fmt.Println("  Workbench-owned review commit: ok")
	fmt.Println("  source checkout remained clean: yes")
	fmt.Println("  external AI workers invoked: no")
	fmt.Println("  live worker check: workbench-runner live-selftest")
	if keep {
		fmt.Println("  retained repository:", dir)
	} else {
		fmt.Println("  cleanup: disposable repository removed")
	}
	return nil
}

// liveSelftest deliberately exercises currently configured real AI workers. It
// is separate from selftest because provider authentication, entitlement,
// quotas and outages are not Workbench control-plane health signals.
func liveSelftest() error {
	dir, keep, err := createSelftestRepository(".workbench-live-selftest-")
	if err != nil {
		return err
	}
	if !keep {
		defer os.RemoveAll(dir)
	}

	marker := fmt.Sprintf("WORKBENCH_LIVE_SELFTEST_OK_%d", time.Now().UTC().UnixNano())
	intent := fmt.Sprintf("Create a file named WORKBENCH_SELFTEST_OK.txt in the repository root containing exactly this single line followed by a newline: %s. Do not modify any other file. Verify the file exists, then report completion.", marker)
	task := core.Task{
		ID:          "runner-live-selftest",
		Origin:      "workbench-runner live-selftest",
		Title:       "Live cluster worker self-test",
		Intent:      intent,
		ProjectPath: dir,
		Status:      core.TaskQueued,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	fmt.Println("Workbench live worker self-test")
	fmt.Println("  workspace:", dir)
	fmt.Println("  policy: zero-marginal/included workers first; metered disabled; scarce Work protected")
	fmt.Println("  delegating one harmless file creation through the real runner route...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	resp := core.ExecuteRunnerRequest(ctx, core.RunnerRequest{Task: task, AvoidWorkUsage: true, AllowMeteredAPI: false})
	if strings.TrimSpace(resp.Error) != "" {
		return fmt.Errorf("runner route failed: %s (attempts: %s)", resp.Error, strings.Join(resp.Attempts, "; "))
	}
	if strings.TrimSpace(resp.Result.Attention) != "" {
		return fmt.Errorf("worker unexpectedly requested human attention: %s", resp.Result.Attention)
	}
	if resp.ProviderCost == core.CostScarce {
		return errors.New("routing invariant violated: live self-test consumed scarce agentic capacity")
	}
	if status, err := runSelftestGit(ctx, dir, "status", "--porcelain"); err != nil {
		return err
	} else if strings.TrimSpace(status) != "" {
		return fmt.Errorf("live worker dirtied the source checkout: %q", strings.TrimSpace(status))
	}
	if _, err := os.Stat(filepath.Join(dir, "WORKBENCH_SELFTEST_OK.txt")); !errors.Is(err, os.ErrNotExist) {
		if err == nil {
			return errors.New("live worker marker leaked into the source checkout instead of a Workbench review branch")
		}
		return fmt.Errorf("inspect live source marker: %w", err)
	}
	commit, err := findLiveSelftestReviewCommit(ctx, dir)
	if err != nil {
		return err
	}
	got, err := runSelftestGit(ctx, dir, "show", commit+":WORKBENCH_SELFTEST_OK.txt")
	if err != nil {
		return fmt.Errorf("read live prepared review marker: %w", err)
	}
	if strings.TrimSpace(got) != marker {
		return fmt.Errorf("live review marker content mismatch: got %q, want %q", strings.TrimSpace(got), marker)
	}

	fmt.Println("LIVE SELFTEST PASSED")
	fmt.Printf("  worker: %s (%s)\n", resp.ProviderName, resp.ProviderCost)
	if len(resp.Attempts) > 0 {
		fmt.Println("  attempts:")
		for _, a := range resp.Attempts {
			fmt.Println("   -", a)
		}
	}
	fmt.Println("  verification: exact requested file exists in the Workbench-owned review commit")
	fmt.Println("  source checkout remained clean: yes")
	fmt.Println("  scarce Work/Codex used: no")
	if keep {
		fmt.Println("  retained repository:", dir)
	} else {
		fmt.Println("  cleanup: disposable repository removed")
	}
	return nil
}

func createSelftestRepository(prefix string) (string, bool, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false, err
	}
	root := strings.TrimSpace(os.Getenv("WORKBENCH_RUNNER_ROOT"))
	if root == "" {
		root = filepath.Join(home, "src")
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return "", false, err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", false, fmt.Errorf("prepare runner root: %w", err)
	}
	if _, err := exec.LookPath("git"); err != nil {
		return "", false, errors.New("git is required for Workbench runner self-tests")
	}

	dir, err := os.MkdirTemp(root, prefix)
	if err != nil {
		return "", false, fmt.Errorf("create isolated selftest repo: %w", err)
	}
	keep := strings.TrimSpace(os.Getenv("WORKBENCH_KEEP_SELFTEST")) == "1"
	cleanupOnError := true
	defer func() {
		if cleanupOnError && !keep {
			_ = os.RemoveAll(dir)
		}
	}()

	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Workbench runner self-test\n\nThis repository is disposable.\n"), 0o644); err != nil {
		return "", false, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for _, args := range [][]string{
		{"init", "-q"},
		{"add", "README.md"},
		{"-c", "user.name=Workbench Selftest", "-c", "user.email=workbench-selftest@example.invalid", "commit", "-q", "-m", "selftest baseline"},
	} {
		if _, err := runSelftestGit(ctx, dir, args...); err != nil {
			return "", false, err
		}
	}
	cleanupOnError = false
	return dir, keep, nil
}

func findLiveSelftestReviewCommit(ctx context.Context, dir string) (string, error) {
	refs, err := runSelftestGit(ctx, dir, "for-each-ref", "--format=%(refname) %(objectname)", "refs/heads/workbench/")
	if err != nil {
		return "", err
	}
	prefix := "refs/heads/workbench/runner-live-selftest-"
	var commits []string
	for _, line := range strings.Split(strings.TrimSpace(refs), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && strings.HasPrefix(fields[0], prefix) {
			commits = append(commits, fields[1])
		}
	}
	if len(commits) != 1 {
		return "", fmt.Errorf("expected one Workbench live-selftest review branch, found %d", len(commits))
	}
	return commits[0], nil
}

func runSelftestGit(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(out))
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), message)
	}
	return string(out), nil
}
