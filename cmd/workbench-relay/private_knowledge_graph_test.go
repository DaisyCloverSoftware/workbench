package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestPrivateKnowledgeDiscoveryUsesSafeHandsBoundary(t *testing.T) {
	for _, action := range []string{"search_decisions", "get_knowledge_graph"} {
		if !isPrivateSafeHandsAction(action) {
			t.Fatalf("knowledge discovery %q must be a private read/safe-hands action", action)
		}
	}
}

func TestPrivateKnowledgeDiscoveryForwardsResolvedProjectAndBoundedArgs(t *testing.T) {
	_, project := privateSafeHandsFixture(t)
	canonicalProject, err := filepath.EvalSymlinks(project)
	if err != nil {
		t.Fatal(err)
	}
	var calls []struct {
		Tool string
		Args map[string]any
	}
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
		calls = append(calls, struct {
			Tool string
			Args map[string]any
		}{req.Params.Name, req.Params.Arguments})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"structuredContent":{"ok":true},"isError":false}}`))
	}))
	defer ts.Close()
	authFile := filepath.Join(t.TempDir(), "auth")
	if err := os.WriteFile(authFile, []byte("Bearer test-only\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, env := range []privateControlEnvelope{
		{Version: 1, ID: "decision-12345678", Action: "search_decisions", Project: "sample", Args: json.RawMessage(`{"query":"release","limit":999}`)},
		{Version: 1, ID: "graph-12345678", Action: "get_knowledge_graph", Project: "sample", Args: json.RawMessage(`{"query":"routing","limit":0}`)},
	} {
		if _, err := executePrivateControl(context.Background(), env, ts.URL, authFile); err != nil {
			t.Fatal(err)
		}
	}
	if len(calls) != 2 || calls[0].Tool != "search_decisions" || calls[1].Tool != "get_knowledge_graph" {
		t.Fatalf("unexpected private knowledge MCP calls: %+v", calls)
	}
	if calls[0].Args["project_path"] != canonicalProject || calls[0].Args["query"] != "release" || calls[0].Args["limit"] != float64(20) {
		// JSON-RPC marshalling converts integral interface values to JSON numbers;
		// the httptest decoder therefore observes float64 limits.
		t.Fatalf("unexpected decision args: %+v", calls[0].Args)
	}
	if calls[1].Args["project_path"] != canonicalProject || calls[1].Args["query"] != "routing" || calls[1].Args["limit"] != float64(40) {
		t.Fatalf("unexpected graph args: %+v", calls[1].Args)
	}
}
