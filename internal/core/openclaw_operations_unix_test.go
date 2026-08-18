//go:build !windows

package core

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenClawOperationsSupervisorAutomaticallyContinuesProgressOnlyExit(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "fake-openclaw")
	body := `#!/usr/bin/env bash
set -euo pipefail
count_file="$PWD/.workbench-fake-count"
count=0
if [ -f "$count_file" ]; then count="$(cat "$count_file")"; fi
count=$((count+1))
printf '%s' "$count" > "$count_file"
if [ "$count" -eq 1 ]; then
  echo "Restart complete; still checking health."
  exit 0
fi
echo "Restart and health verification complete."
echo "WORKBENCH_OPERATION_COMPLETE: verified"
`
	if err := os.WriteFile(script, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}

	provider := Provider{ID: "openclaw", Name: "OpenClaw", Command: script, Installed: true, Authenticated: true, CanWrite: true, CanRunTools: true}
	task := Task{ID: "task-supervised-op", Mode: TaskModeOperations, ProjectPath: dir, Intent: "Restart the runner and verify health"}
	res, err := RunOpenClawOperationSupervised(context.Background(), provider, task, Preferences{})
	if err != nil {
		t.Fatalf("supervised operation failed: %v; output=%q", err, res.Output)
	}
	count, err := os.ReadFile(filepath.Join(dir, ".workbench-fake-count"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(count)) != "2" {
		t.Fatalf("invocations=%q want 2", count)
	}
	if strings.Contains(res.Output, operationCompletePrefix) || !strings.Contains(res.Output, "health verification complete") {
		t.Fatalf("unexpected final report: %q", res.Output)
	}
}
