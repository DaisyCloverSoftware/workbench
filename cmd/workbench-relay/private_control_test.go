package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodePrivateControlIsStrict(t *testing.T) {
	good := []byte(`{"version":1,"id":"control-12345678","action":"search_memory","project":"workbench","args":{"query":"retry"}}`)
	env, err := decodePrivateControl(good, "control-12345678")
	if err != nil {
		t.Fatal(err)
	}
	if env.Action != "search_memory" || env.Project != "workbench" {
		t.Fatalf("unexpected envelope: %#v", env)
	}

	for _, raw := range [][]byte{
		[]byte(`{"version":1,"id":"control-12345678","action":"search_memory","surprise":true}`),
		[]byte(`{"version":1,"id":"different-12345678","action":"search_memory"}`),
		[]byte(`{"version":1,"id":"control-12345678","action":"run_command"}`),
	} {
		if _, err := decodePrivateControl(raw, "control-12345678"); err == nil {
			t.Fatalf("expected private control envelope to be rejected: %s", raw)
		}
	}
}

func TestPrivateControlWithholdsSecretLikeResult(t *testing.T) {
	secretish := "token: " + strings.Repeat("x", 12)
	b, err := marshalPrivateControlOutbox(privateControlOutbox{
		Version: 1,
		ID:      "control-12345678",
		Action:  "search_memory",
		Status:  "completed",
		Result:  map[string]any{"content": secretish},
	})
	if err != nil {
		t.Fatal(err)
	}
	var got privateControlOutbox
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Status != "failed" || got.Result != nil || !strings.Contains(got.Error, "withheld") {
		t.Fatalf("secret-like result was not withheld: %#v", got)
	}
}

func TestExecutePrivateControlCallsLocalMCP(t *testing.T) {
	var gotTool string
	var gotArgs map[string]any
	var gotAuth string
	var decodeErr error
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Params struct {
				Name      string         `json:"name"`
				Arguments map[string]any `json:"arguments"`
			} `json:"params"`
		}
		decodeErr = json.Unmarshal(body, &req)
		gotTool = req.Params.Name
		gotArgs = req.Params.Arguments
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"structuredContent":{"ok":true},"isError":false}}`))
	}))
	defer ts.Close()

	authFile := filepath.Join(t.TempDir(), "auth")
	if err := os.WriteFile(authFile, []byte("Bearer test-only\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	env := privateControlEnvelope{
		Version: 1,
		ID:      "control-12345678",
		Action:  "save_memory",
		Args:    json.RawMessage(`{"scope":"global","kind":"routine","title":"Go verification","content":"Run the repository test suite before reporting completion.","tags":["go","test"]}`),
	}
	result, err := executePrivateControl(context.Background(), env, ts.URL, authFile)
	if err != nil {
		t.Fatal(err)
	}
	if decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if gotAuth != "Bearer test-only" {
		t.Fatalf("unexpected authorization header %q", gotAuth)
	}
	if result["ok"] != true || gotTool != "save_memory" {
		t.Fatalf("unexpected MCP result/tool: result=%#v tool=%q", result, gotTool)
	}
	if gotArgs["scope"] != "global" || gotArgs["kind"] != "routine" {
		t.Fatalf("unexpected save_memory args: %#v", gotArgs)
	}
}

func TestPrivateControlProjectStaysUnderRunnerRoot(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "sample")
	if err := os.Mkdir(project, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WORKBENCH_RUNNER_ROOT", root)

	var gotProject string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Params struct {
				Arguments map[string]any `json:"arguments"`
			} `json:"params"`
		}
		_ = json.Unmarshal(body, &req)
		gotProject, _ = req.Params.Arguments["project_path"].(string)
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"structuredContent":{"found":false},"isError":false}}`))
	}))
	defer ts.Close()
	authFile := filepath.Join(t.TempDir(), "auth")
	if err := os.WriteFile(authFile, []byte("Bearer test-only"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := executePrivateControl(context.Background(), privateControlEnvelope{Version: 1, ID: "control-12345678", Action: "get_context", Project: "sample", Args: json.RawMessage(`{}`)}, ts.URL, authFile)
	if err != nil {
		t.Fatal(err)
	}
	if gotProject != project {
		t.Fatalf("project_path=%q want %q", gotProject, project)
	}
}
