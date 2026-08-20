package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestDelegateRelayTaskMCPUsesPrivateCompatibilityRoute(t *testing.T) {
	authFile := filepath.Join(t.TempDir(), "auth")
	if err := os.WriteFile(authFile, []byte("Bearer relay-test"), 0o600); err != nil {
		t.Fatal(err)
	}

	intent := "[relay:durable_001] " + core.RelayOperationsIntentPrefix + " continue until the bounded operation is complete"
	project := "/tmp/workbench"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer relay-test" {
			t.Fatalf("Authorization=%q", got)
		}
		var request struct {
			Method string `json:"method"`
			Params struct {
				Name      string         `json:"name"`
				Arguments map[string]any `json:"arguments"`
			} `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.Method != "tools/call" {
			t.Fatalf("method=%q", request.Method)
		}
		if request.Params.Name != "delegate_task" {
			t.Fatalf("tool=%q, want hidden private-relay compatibility tool delegate_task", request.Params.Name)
		}
		if request.Params.Arguments["intent"] != intent {
			t.Fatalf("intent=%q", request.Params.Arguments["intent"])
		}
		if request.Params.Arguments["project_path"] != project {
			t.Fatalf("project_path=%q", request.Params.Arguments["project_path"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"structuredContent":{"task_id":"task-durable-001"},"isError":false}}`))
	}))
	defer server.Close()

	taskID, err := delegateRelayTaskMCP(context.Background(), server.URL, authFile, intent, project)
	if err != nil {
		t.Fatal(err)
	}
	if taskID != "task-durable-001" {
		t.Fatalf("taskID=%q", taskID)
	}
}

func TestRelayTaggedIntentRemainsOperationsOnly(t *testing.T) {
	intent := "[relay:durable_001] " + core.RelayOperationsIntentPrefix + " continue safely"
	task := core.Task{Intent: intent}
	if !core.IsOperationsTask(task) || !core.TaskUsesOperationsLane(task) {
		t.Fatalf("private relay intent escaped operations lane: %q", intent)
	}
}
