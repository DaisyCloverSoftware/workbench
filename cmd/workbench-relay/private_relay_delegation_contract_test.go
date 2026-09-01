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

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestDelegateRelayTaskMCPUsesPrivateCompatibilityRoute(t *testing.T) {
	authFile := filepath.Join(t.TempDir(), "auth")
	if err := os.WriteFile(authFile, []byte("Bearer relay-test"), 0o600); err != nil {
		t.Fatal(err)
	}
	intent := core.OpenClawExplicitAuthorizationPrefix + " [relay:durable_001] " + core.RelayOperationsIntentPrefix + " continue until complete"
	project := "/tmp/workbench"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		if request.Params.Name != "delegate_task" {
			t.Fatalf("tool=%q", request.Params.Name)
		}
		if request.Params.Arguments["intent"] != intent {
			t.Fatalf("intent=%q", request.Params.Arguments["intent"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"structuredContent":{"task_id":"task-durable-001"},"isError":false}}`))
	}))
	defer server.Close()

	if taskID, err := delegateRelayTaskMCP(context.Background(), server.URL, authFile, intent, project); err != nil || taskID != "task-durable-001" {
		t.Fatalf("taskID=%q err=%v", taskID, err)
	}
}

func TestPrivateRelayDelegationRejectsOperationsMarkerWithoutOwnerAuthorization(t *testing.T) {
	_, _, err := prepareRelayTaskIntent("", "durable_001", "/tmp/workbench", core.RelayOperationsIntentPrefix+" kubectl apply")
	if err == nil || !strings.Contains(err.Error(), "not owner authorization") {
		t.Fatalf("ordinary operations marker authorized OpenClaw: %v", err)
	}
}

func TestPrepareRelayTaskIntentAcceptsExplicitOwnerAuthorizedOpenClaw(t *testing.T) {
	got, kind, err := prepareRelayTaskIntent("", "durable_001", "/tmp/workbench", core.OpenClawExplicitAuthorizationPrefix+" "+core.RelayOperationsIntentPrefix+" investigate runtime")
	if err != nil {
		t.Fatal(err)
	}
	if kind != "openclaw-operations" || !strings.HasPrefix(got, core.OpenClawExplicitAuthorizationPrefix) {
		t.Fatalf("unexpected authorized intent: %q %q", kind, got)
	}
}

func TestRelayTaggedIntentRemainsOperationsOnly(t *testing.T) {
	intent := "[relay:durable_001] " + core.RelayOperationsIntentPrefix + " continue safely"
	task := core.Task{Intent: intent}
	if !core.IsOperationsTask(task) || !core.TaskUsesOperationsLane(task) {
		t.Fatalf("private relay intent escaped operations lane: %q", intent)
	}
}
