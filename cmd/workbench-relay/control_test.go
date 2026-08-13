package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestControlCallMapsKnowledgeActions(t *testing.T) {
	cases := []struct {
		action       string
		scope        string
		tool         string
		needsProject bool
	}{
		{"checkpoint", "", "save_checkpoint", true},
		{"remember", "project", "remember", true},
		{"remember", "global", "remember", false},
		{"routine", "project", "save_routine", true},
		{"routine", "global", "save_routine", false},
		{"context", "", "get_context_pack", true},
		{"recall", "", "recall_memory", false},
		{"routines", "", "find_routines", false},
	}
	for _, tc := range cases {
		t.Run(tc.action+"-"+tc.scope, func(t *testing.T) {
			tool, _, needsProject, err := controlCall(controlEnvelope{Action: tc.action, Scope: tc.scope})
			if err != nil {
				t.Fatal(err)
			}
			if tool != tc.tool || needsProject != tc.needsProject {
				t.Fatalf("got tool=%q needsProject=%v", tool, needsProject)
			}
		})
	}
	if _, _, _, err := controlCall(controlEnvelope{Action: "launch-missiles"}); err == nil {
		t.Fatal("expected unknown control action to be rejected")
	}
}

func TestPrivateControlRoundTripPublishesResult(t *testing.T) {
	if _, err := os.Stat("."); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	bare := filepath.Join(root, "remote.git")
	seed := filepath.Join(root, "seed")
	relay := filepath.Join(root, "relay")
	runnerRoot := filepath.Join(root, "projects")
	project := filepath.Join(runnerRoot, "widget")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WORKBENCH_RUNNER_ROOT", runnerRoot)
	t.Setenv("WORKBENCH_RELAY_STATE_PATH", filepath.Join(root, "relay-state.json"))

	runGit(t, "init", "--bare", bare)
	runGit(t, "init", "-b", "main", seed)
	controlID := "control-12345678"
	controlPath := filepath.Join(seed, "relay", "control", controlID+".json")
	if err := os.MkdirAll(filepath.Dir(controlPath), 0o755); err != nil {
		t.Fatal(err)
	}
	payload := map[string]any{
		"version":      1,
		"id":           controlID,
		"action":       "checkpoint",
		"project":      "widget",
		"summary":      "Compact state ready for a fresh conversation.",
		"decisions":    []string{"Use durable checkpoints."},
		"open_loops":   []string{"Add semantic retrieval later."},
		"next_actions": []string{"Resume from the context pack."},
	}
	b, _ := json.Marshal(payload)
	if err := os.WriteFile(controlPath, append(b, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, "-C", seed, "add", ".")
	runGit(t, "-C", seed, "-c", "user.name=Workbench Test", "-c", "user.email=workbench-test@example.invalid", "commit", "-m", "control")
	runGit(t, "-C", seed, "remote", "add", "origin", bare)
	runGit(t, "-C", seed, "push", "origin", "main")
	runGit(t, "--git-dir", bare, "symbolic-ref", "HEAD", "refs/heads/main")
	runGit(t, "clone", "--quiet", bare, relay)

	authFile := filepath.Join(root, "auth")
	if err := os.WriteFile(authFile, []byte("Bearer test-token\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("authorization header = %q", r.Header.Get("Authorization"))
		}
		var req struct {
			Params struct {
				Name      string         `json:"name"`
				Arguments map[string]any `json:"arguments"`
			} `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.Params.Name != "save_checkpoint" {
			t.Fatalf("tool = %q", req.Params.Name)
		}
		if req.Params.Arguments["project_path"] != project {
			t.Fatalf("project_path = %#v want %q", req.Params.Arguments["project_path"], project)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"structuredContent":{"ok":true,"checkpoint":{"id":"checkpoint-test"}},"isError":false}}`))
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := processControlPath(ctx, relay, "origin/main", "relay/control/"+controlID+".json", server.URL, authFile); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("MCP calls = %d want 1", calls)
	}
	if err := processControlPath(ctx, relay, "origin/main", "relay/control/"+controlID+".json", server.URL, authFile); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("idempotent control called MCP again: %d", calls)
	}
	rec, ok, err := core.LoadRelayControlRecord(controlID)
	if err != nil || !ok {
		t.Fatalf("load control record: ok=%v err=%v", ok, err)
	}
	if rec.Action != "checkpoint" || !strings.Contains(string(rec.Response), "checkpoint-test") {
		t.Fatalf("unexpected control record: %#v", rec)
	}
	if err := syncControlOutbox(ctx, relay, "origin", "main"); err != nil {
		t.Fatal(err)
	}
	runGit(t, "-C", relay, "fetch", "--quiet", "origin", "main")
	out := runGit(t, "-C", relay, "show", "origin/main:relay/control-outbox/"+controlID+".json")
	if !strings.Contains(out, `"status": "completed"`) || !strings.Contains(out, "checkpoint-test") {
		t.Fatalf("unexpected control outbox: %s", out)
	}
}
