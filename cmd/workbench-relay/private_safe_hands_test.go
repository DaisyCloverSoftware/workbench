package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func privateSafeHandsFixture(t *testing.T) (root, project string) {
	t.Helper()
	root = t.TempDir()
	project = filepath.Join(root, "sample")
	if err := os.Mkdir(project, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WORKBENCH_RUNNER_ROOT", root)
	return root, project
}

func TestPrivateSafeHandsActionBoundary(t *testing.T) {
	for _, action := range []string{"list_projects", "ensure_github_project", "list_files", "search_text", "read_file", "apply_patch", "run_safe_command", "save_note"} {
		if !isPrivateSafeHandsAction(action) {
			t.Fatalf("expected %q to be a private safe-hands action", action)
		}
	}
	for _, action := range []string{"delegate_task", "resolve_attention", "update_workbench", "run_command"} {
		if isPrivateSafeHandsAction(action) {
			t.Fatalf("%q must not cross the safe-hands boundary", action)
		}
	}
}

func TestPrivateSafeHandsListsOnlyRunnerRepositoryRoots(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WORKBENCH_RUNNER_ROOT", root)
	repo := filepath.Join(root, "sample")
	if err := os.Mkdir(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "init", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	if err := os.Mkdir(filepath.Join(root, "not-a-repo"), 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := executePrivateControl(context.Background(), privateControlEnvelope{
		Version: 1,
		ID:      "control-12345678",
		Action:  "list_projects",
		Args:    json.RawMessage(`{}`),
	}, "http://127.0.0.1:1", "unused")
	if err != nil {
		t.Fatal(err)
	}
	if result["count"] != 1 {
		t.Fatalf("unexpected project count/result: %#v", result)
	}
	b, err := json.Marshal(result["projects"])
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	if !strings.Contains(text, `"name":"sample"`) || strings.Contains(text, "not-a-repo") {
		t.Fatalf("private project discovery leaked non-repository entries: %s", text)
	}
}

func TestPrivateEnsureGitHubProjectFailsClosedBeforeNetworkForUnsafeSlug(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WORKBENCH_RUNNER_ROOT", root)
	_, err := executePrivateControl(context.Background(), privateControlEnvelope{
		Version: 1,
		ID:      "control-12345678",
		Action:  "ensure_github_project",
		Args:    json.RawMessage(`{"repository":"https://example.invalid/repo"}`),
	}, "http://127.0.0.1:1", "unused")
	if err == nil || !strings.Contains(err.Error(), "owner/name") {
		t.Fatalf("unsafe GitHub import did not fail closed: %v", err)
	}
}

func TestPrivateSafeHandsForwardsBoundedCommandToLocalMCP(t *testing.T) {
	_, project := privateSafeHandsFixture(t)
	canonicalProject, err := filepath.EvalSymlinks(project)
	if err != nil {
		t.Fatal(err)
	}

	var gotTool string
	var gotArgs map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Params struct {
				Name      string         `json:"name"`
				Arguments map[string]any `json:"arguments"`
			} `json:"params"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		gotTool = req.Params.Name
		gotArgs = req.Params.Arguments
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"structuredContent":{"ok":true,"output":"ok"},"isError":false}}`))
	}))
	defer ts.Close()

	authFile := filepath.Join(t.TempDir(), "auth")
	if err := os.WriteFile(authFile, []byte("Bearer test-only\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := executePrivateControl(context.Background(), privateControlEnvelope{
		Version: 1,
		ID:      "control-12345678",
		Action:  "run_safe_command",
		Project: "sample",
		Args:    json.RawMessage(`{"command":"go test ./..."}`),
	}, ts.URL, authFile)
	if err != nil {
		t.Fatal(err)
	}
	if result["ok"] != true || gotTool != "run_safe_command" {
		t.Fatalf("unexpected result/tool: result=%#v tool=%q", result, gotTool)
	}
	if gotArgs["project_path"] != canonicalProject || gotArgs["command"] != "go test ./..." {
		t.Fatalf("unexpected MCP args: %#v", gotArgs)
	}
}

func TestPrivateSafeHandsRejectsMissingProject(t *testing.T) {
	if _, err := executePrivateControl(context.Background(), privateControlEnvelope{
		Version: 1,
		ID:      "control-12345678",
		Action:  "run_safe_command",
		Args:    json.RawMessage(`{"command":"go test ./..."}`),
	}, "http://127.0.0.1:1", "unused"); err == nil {
		t.Fatal("missing safe-hands project must fail")
	}
}

func TestPrivateSafeHandsRejectsLooseArgs(t *testing.T) {
	privateSafeHandsFixture(t)
	_, err := executePrivateControl(context.Background(), privateControlEnvelope{
		Version: 1,
		ID:      "control-12345678",
		Action:  "search_text",
		Project: "sample",
		Args:    json.RawMessage(`{"query":"router","unexpected":true}`),
	}, "http://127.0.0.1:1", "unused")
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("loose safe-hands args were not rejected strictly: %v", err)
	}
}

func TestPrivateSafeHandsRejectsProjectSymlinkOutsideRunnerRoot(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unavailable on this test host: %v", err)
	}
	t.Setenv("WORKBENCH_RUNNER_ROOT", root)
	if _, err := resolvePrivateSafeHandsProject("escape"); err == nil {
		t.Fatal("safe hands accepted a project symlink escaping the runner root")
	}
}

func TestDecodePrivateControlAcceptsSafeHandsButStillRejectsArbitraryCommand(t *testing.T) {
	good := []byte(`{"version":1,"id":"control-12345678","action":"apply_patch","project":"workbench","args":{"patch":"diff --git a/a b/a"}}`)
	if _, err := decodePrivateControl(good, "control-12345678"); err != nil {
		t.Fatalf("safe-hands envelope rejected: %v", err)
	}
	bad := []byte(`{"version":1,"id":"control-12345678","action":"run_command","project":"workbench","args":{"command":"rm -rf /"}}`)
	if _, err := decodePrivateControl(bad, "control-12345678"); err == nil {
		t.Fatal("arbitrary command action must remain rejected")
	}
}
