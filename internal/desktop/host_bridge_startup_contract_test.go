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
		"st.Preferences.OpenClawSSHHost",
		"core.RunWindowsHostBridgeAgent(hostBridgeCtx, host)",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("owned desktop startup is missing host bridge lifecycle contract %q", required)
		}
	}
	if strings.Contains(text, "ListenAndServe") || strings.Contains(text, "net.Listen") {
		t.Fatal("Windows host bridge startup unexpectedly opens an inbound listener")
	}
}
