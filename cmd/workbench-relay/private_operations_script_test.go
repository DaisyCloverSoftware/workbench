package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPrivateRelayRunsCommittedOperationsScriptWithoutMCPOrAIWorker(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX committed operations script smoke")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash unavailable")
	}
	root := t.TempDir()
	repo := filepath.Join(root, "sample")
	if err := os.Mkdir(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WORKBENCH_RUNNER_ROOT", root)
	gitRelayTest(t, repo, "init")
	gitRelayTest(t, repo, "config", "user.email", "workbench-test@example.invalid")
	gitRelayTest(t, repo, "config", "user.name", "Workbench Test")
	path := filepath.Join(repo, "scripts", "ops", "verify.sh")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("#!/usr/bin/env bash\nset -euo pipefail\nprintf 'relay-op=%s\\n' \"${1:-none}\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	gitRelayTest(t, repo, "add", "scripts/ops/verify.sh")
	gitRelayTest(t, repo, "commit", "-m", "add operation")

	result, err := executePrivateControl(context.Background(), privateControlEnvelope{
		Version: 1,
		ID:      "operation-script-12345678",
		Action:  "run_operations_script",
		Project: "sample",
		Args:    json.RawMessage(`{"path":"scripts/ops/verify.sh","args":["literal;argv"],"timeout_seconds":30}`),
	}, "http://127.0.0.1:1", "unused")
	if err != nil {
		t.Fatal(err)
	}
	got, ok := result["operation_script"].(interface{})
	if !ok || got == nil {
		t.Fatalf("missing operation_script result: %#v", result)
	}
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	for _, want := range []string{`"path":"scripts/ops/verify.sh"`, `"transport":"git-worktree-bash"`, "relay-op=literal;argv"} {
		if !strings.Contains(text, want) {
			t.Fatalf("operations result missing %q: %s", want, text)
		}
	}
}

func TestPrivateRelayOperationsScriptRequiresProject(t *testing.T) {
	_, err := executePrivateControl(context.Background(), privateControlEnvelope{
		Version: 1,
		ID:      "operation-script-87654321",
		Action:  "run_operations_script",
		Args:    json.RawMessage(`{"path":"scripts/ops/verify.sh"}`),
	}, "http://127.0.0.1:1", "unused")
	if err == nil || !strings.Contains(err.Error(), "project") {
		t.Fatalf("projectless operations script should fail closed: %v", err)
	}
}

func gitRelayTest(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return string(out)
}
