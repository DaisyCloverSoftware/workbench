package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func freePort(t *testing.T) int {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	p := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	return p
}

func TestMCPRequiresTokenAndListsEyesHandsAndOperationsTools(t *testing.T) {
	store, err := core.NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	eng, err := core.NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	s := New(eng, freePort(t), "test-token")
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Close(context.Background())
	time.Sleep(10 * time.Millisecond)
	payload, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list"})
	resp, err := http.Post(s.URL(), "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status=%d", resp.StatusCode)
	}
	req, _ := http.NewRequest(http.MethodPost, s.URL(), bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var got map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(got)
	text := string(b)
	for _, name := range []string{
		"get_workspace", "list_files", "search_text", "read_file",
		"apply_patch", "run_safe_command", "delegate_operation", "await_operation", "get_task",
	} {
		if !bytes.Contains(b, []byte(name)) {
			t.Fatalf("missing tool %s in %s", name, text)
		}
	}
	if bytes.Contains(b, []byte(`"name":"delegate_task"`)) {
		t.Fatalf("coding delegation must not be advertised to ChatGPT: %s", text)
	}
}
