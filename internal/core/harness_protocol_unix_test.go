//go:build !windows

package core

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunHarnessAdapterExecutesFixedFileWithoutShellExpansion(t *testing.T) {
	dir := t.TempDir()
	capture := filepath.Join(dir, "job.json")
	adapter := filepath.Join(dir, "adapter; touch SHOULD_NOT_EXIST")
	script := `#!/bin/sh
set -eu
cat > "$HARNESS_JOB_CAPTURE"
printf '%s\n' '{"version":1,"task_id":"task-structured","status":"completed","report":"structured adapter completed"}'
`
	if err := os.WriteFile(adapter, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HARNESS_JOB_CAPTURE", capture)
	project := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := RunHarnessAdapter(ctx, adapter, Task{
		ID:          "task-structured",
		ProjectPath: project,
		Intent:      "Change the repository safely",
	}, "bounded worker prompt")
	if err != nil {
		t.Fatal(err)
	}
	if res.Output != "structured adapter completed" || res.Attention != "" || res.WorkerUnavailable != "" {
		t.Fatalf("unexpected structured adapter result: %#v", res)
	}
	if _, err := os.Stat(filepath.Join(project, "SHOULD_NOT_EXIST")); !os.IsNotExist(err) {
		t.Fatalf("adapter filename was interpreted by a shell: %v", err)
	}
	body, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, want := range []string{`"version":1`, `"task_id":"task-structured"`, `"project_path":"` + strings.ReplaceAll(project, `\`, `\\`) + `"`, `"network_access":false`, `"publish":false`, `"deploy":false`, `"secrets":false`} {
		if !strings.Contains(text, want) {
			t.Fatalf("captured harness job missing %q: %s", want, text)
		}
	}
}

func TestRunHarnessAdapterRejectsMismatchedTaskIdentity(t *testing.T) {
	dir := t.TempDir()
	adapter := filepath.Join(dir, "adapter")
	if err := os.WriteFile(adapter, []byte("#!/bin/sh\nprintf '%s\\n' '{\"version\":1,\"task_id\":\"other-task\",\"status\":\"completed\"}'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := RunHarnessAdapter(context.Background(), adapter, Task{ID: "expected-task", ProjectPath: t.TempDir(), Intent: "work"}, "prompt")
	if err == nil || !strings.Contains(err.Error(), "identity mismatch") {
		t.Fatalf("expected task identity rejection, got %v", err)
	}
}
