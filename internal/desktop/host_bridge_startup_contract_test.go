package desktop

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestOwnedWindowsStartupRunsOutboundHostBridge(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test source path")
	}
	path := filepath.Join(filepath.Dir(here), "startup_owned_windows.go")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, required := range []string{
		"context.WithCancel(context.Background())",
		"defer stopHostBridge()",
		"core.NewSavedHostBridgeTarget(st.Preferences.OpenClawSSHHost)",
		"core.RunWindowsHostBridgeAgentWithTargetSource(hostBridgeCtx, hostBridgeTarget.Current)",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("owned desktop startup is missing host bridge lifecycle contract %q", required)
		}
	}
	if strings.Contains(text, "if host := strings.TrimSpace(st.Preferences.OpenClawSSHHost)") {
		t.Fatal("bridge must not be gated on a stale startup preference snapshot")
	}
	if strings.Contains(text, "ListenAndServe") || strings.Contains(text, "net.Listen") {
		t.Fatal("Windows host bridge startup unexpectedly opens an inbound listener")
	}
}
