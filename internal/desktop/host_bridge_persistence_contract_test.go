package desktop

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestHostBridgeSavedTargetDesktopPersistenceContract(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate source")
	}
	read := func(name string) string {
		b, err := os.ReadFile(filepath.Join(filepath.Dir(here), name))
		if err != nil {
			t.Fatal(err)
		}
		return strings.Join(strings.Fields(string(b)), " ")
	}
	startup := read("startup_owned_windows.go")
	for _, want := range []string{
		"core.NewSavedHostBridgeTarget(st.Preferences.OpenClawSSHHost)",
		"core.RunWindowsHostBridgeAgentWithTargetSource(hostBridgeCtx, hostBridgeTarget.Current)",
		"hostBridgeTarget: hostBridgeTarget,",
	} {
		if !strings.Contains(startup, want) {
			t.Fatalf("missing committed target wiring: %s", want)
		}
	}
	if strings.Contains(startup, "return eng.State().Preferences.OpenClawSSHHost") {
		t.Fatal("bridge consumes unpersisted engine preferences")
	}
	settings := read("settings_windows.go")
	if !strings.Contains(settings, "if err := s.saveRoutingPreferences(prefs); err != nil {") {
		t.Fatal("routing save bypasses committed target")
	}
	helper := read("host_bridge_preferences_windows.go")
	if !strings.Contains(helper, "persist := func() error { return s.eng.SavePreferences(prefs) }") ||
		!strings.Contains(helper, "return s.hostBridgeTarget.Save(prefs.OpenClawSSHHost, persist)") {
		t.Fatal("connection target must depend on the actual persistence result")
	}
}
