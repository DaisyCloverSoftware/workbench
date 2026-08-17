package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPrivateUpdateSourceDirUsesConfiguredRoot(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WORKBENCH_SOURCE_DIR", "")
	t.Setenv("WORKBENCH_RUNNER_ROOT", root)
	got, err := privateUpdateSourceDir()
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.Abs(filepath.Join(root, "workbench"))
	if got != want {
		t.Fatalf("source dir=%q want %q", got, want)
	}
}

func TestDecodePrivateControlAllowsOnlyArgumentFreeUpdateAction(t *testing.T) {
	raw := []byte(`{"version":1,"id":"update-12345678","action":"update_workbench","args":{}}`)
	env, err := decodePrivateControl(raw, "update-12345678")
	if err != nil {
		t.Fatal(err)
	}
	if env.Action != "update_workbench" {
		t.Fatalf("action=%q", env.Action)
	}
	if err := decodePrivateControlArgs([]byte(`{"command":"rm -rf /"}`), &struct{}{}); err == nil {
		t.Fatal("update control must reject arbitrary arguments")
	}
}

func TestPrivateUpdateStatusRoundTripIsCategorical(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	t.Setenv("WORKBENCH_PRIVATE_UPDATE_STATUS_FILE", path)
	if err := writePrivateUpdateStatus(privateUpdateRunning); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("status file permissions are too broad: %o", info.Mode().Perm())
	}
	got, err := readPrivateUpdateStatus()
	if err != nil {
		t.Fatal(err)
	}
	if got["state"] != privateUpdateRunning || strings.TrimSpace(got["updated_at"].(string)) == "" {
		t.Fatalf("unexpected status: %#v", got)
	}
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	for _, forbidden := range []string{"command", "remote", "repo", "path", "token", "secret", "journal"} {
		if strings.Contains(strings.ToLower(text), forbidden) {
			t.Fatalf("categorical status leaked %q: %s", forbidden, text)
		}
	}
}

func TestPrivateUpdateStatusReportsNeverRunWithoutStateFile(t *testing.T) {
	t.Setenv("WORKBENCH_PRIVATE_UPDATE_STATUS_FILE", filepath.Join(t.TempDir(), "missing.json"))
	got, err := readPrivateUpdateStatus()
	if err != nil {
		t.Fatal(err)
	}
	if got["state"] != privateUpdateNeverRun {
		t.Fatalf("status=%#v", got)
	}
}

func TestPrivateControlUpdateStatusNeedsNoWorkerOrMCP(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	t.Setenv("WORKBENCH_PRIVATE_UPDATE_STATUS_FILE", path)
	if err := writePrivateUpdateStatus(privateUpdateSucceeded); err != nil {
		t.Fatal(err)
	}
	result, err := executePrivateControl(context.Background(), privateControlEnvelope{
		Version: 1,
		ID:      "status-12345678",
		Action:  "update_status",
		Args:    json.RawMessage(`{}`),
	}, "http://127.0.0.1:1", "unused")
	if err != nil {
		t.Fatal(err)
	}
	if result["state"] != privateUpdateSucceeded {
		t.Fatalf("result=%#v", result)
	}
}

func TestPrivateControlUpdateStatusRejectsProjectAndArguments(t *testing.T) {
	for _, env := range []privateControlEnvelope{
		{Version: 1, ID: "status-12345678", Action: "update_status", Project: "workbench", Args: json.RawMessage(`{}`)},
		{Version: 1, ID: "status-12345678", Action: "update_status", Args: json.RawMessage(`{"command":"whoami"}`)},
	} {
		if _, err := executePrivateControl(context.Background(), env, "http://127.0.0.1:1", "unused"); err == nil {
			t.Fatalf("unsafe update-status request was accepted: %#v", env)
		}
	}
}

func TestPrivateBootstrapKeepsDeveloperCheckoutUntouched(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(here), "..", ".."))
	b, err := os.ReadFile(filepath.Join(root, "scripts", "bootstrap-private-relay.sh"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	for _, want := range []string{"project_source_dir", "update_source_dir", "Never reset, clean, switch or", "install-cluster-mcp.sh\" \"$project_source_dir"} {
		if !strings.Contains(text, want) {
			t.Fatalf("private bootstrap missing non-destructive update contract %q", want)
		}
	}
	for _, forbidden := range []string{"git -C \"$project_source_dir\" reset", "git -C \"$project_source_dir\" clean", "git -C \"$project_source_dir\" switch", "git -C \"$project_source_dir\" merge"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("private bootstrap can mutate developer checkout: %q", forbidden)
		}
	}

	helper, err := os.ReadFile(filepath.Join(root, "scripts", "run-private-update.sh"))
	if err != nil {
		t.Fatal(err)
	}
	helperText := string(helper)
	for _, want := range []string{"write_status running", "write_status succeeded", "write_status failed", "bootstrap-private-relay.sh"} {
		if !strings.Contains(helperText, want) {
			t.Fatalf("private update helper missing status contract %q", want)
		}
	}
}
