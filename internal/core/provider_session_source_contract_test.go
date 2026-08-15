package core

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// This source-level contract keeps provider-native continuation private and
// optional. Session identity belongs to the execution host sidecar; it must not
// migrate into durable Task/RunnerRequest JSON where desktop/runner or MCP
// transport would expose host-local provider state.
func TestProviderSessionIdentityStaysOutOfTaskAndRunnerTransportSource(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate core source directory")
	}
	dir := filepath.Dir(thisFile)
	for _, name := range []string{"model.go", "cluster_runner.go"} {
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		text := strings.ToLower(string(body))
		for _, forbidden := range []string{"session_id", "provider_session", "provider_sessions"} {
			if strings.Contains(text, forbidden) {
				t.Fatalf("%s contains private provider-session transport field %q", name, forbidden)
			}
		}
	}
}
