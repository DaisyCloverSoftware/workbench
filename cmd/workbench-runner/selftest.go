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

func selftest() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	root := strings.TrimSpace(os.Getenv("WORKBENCH_RUNNER_ROOT"))
	if root == "" {
		root = filepath.Join(home, "src")
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		return fmt.Errorf("prepare runner root: %w", err)
	}

	dir, err := os.MkdirTemp(root, ".workbench-live-selftest-")
	if err != nil {
		return fmt.Errorf("create isolated selftest repo: %w", err)
	}
	keep := strings.TrimSpace(os.Getenv("WORKBENCH_KEEP_SELFTEST")) == "1"
	if !keep {
		defer os.RemoveAll(dir)
	}

	marker := fmt.Sprintf("WORKBENCH_SELFTEST_OK_%d", time.Now().UTC().UnixNano())
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Workbench isolated live self-test\n\nThis directory is disposable.\n"), 0644); err != nil {
		return err
	}
	if git, lookErr := exec.LookPath("git"); lookErr == nil {
		cmd := exec.Command(git, "init", "-q")
		cmd.Dir = dir
		_ = cmd.Run()
	}

	intent := fmt.Sprintf("Create a file named WORKBENCH_SELFTEST_OK.txt in the repository root containing exactly this single line followed by a newline: %s. Do not modify any other file. Verify the file exists, then report completion.", marker)
	task := core.Task{
		ID:          "runner-live-selftest",
		Origin:      "workbench-runner selftest",
		Title:       "Live cluster worker self-test",
		Intent:      intent,
		ProjectPath: dir,
		Status:      core.TaskQueued,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	fmt.Println("Workbench live self-test")
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

	got, err := os.ReadFile(filepath.Join(dir, "WORKBENCH_SELFTEST_OK.txt"))
	if err != nil {
		return fmt.Errorf("worker returned success but marker file is missing: %w", err)
	}
	if strings.TrimSpace(string(got)) != marker {
		return fmt.Errorf("marker content mismatch: got %q, want %q", strings.TrimSpace(string(got)), marker)
	}

	fmt.Println("SELFTEST PASSED")
	fmt.Printf("  worker: %s (%s)\n", resp.ProviderName, resp.ProviderCost)
	if len(resp.Attempts) > 0 {
		fmt.Println("  attempts:")
		for _, a := range resp.Attempts {
			fmt.Println("   -", a)
		}
	}
	fmt.Println("  verification: worker created the exact requested file inside the isolated repo")
	fmt.Println("  scarce Work/Codex used: no")
	if keep {
		fmt.Println("  retained workspace:", dir)
	} else {
		fmt.Println("  cleanup: isolated repo removed")
	}
	return nil
}
