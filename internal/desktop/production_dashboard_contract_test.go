package desktop

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestProductionDesktopStartsInRealDashboardShell(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve desktop source directory")
	}
	dir := filepath.Dir(here)
	startup, err := os.ReadFile(filepath.Join(dir, "startup_owned_windows.go"))
	if err != nil {
		t.Fatal(err)
	}
	startupText := string(startup)
	for _, want := range []string{"page:     pageDashboard", "return runProductionShell(shell)"} {
		if !strings.Contains(startupText, want) {
			t.Fatalf("production startup is not dashboard-bound: missing %q", want)
		}
	}

	dashboard, err := os.ReadFile(filepath.Join(dir, "dashboard_windows.go"))
	if err != nil {
		t.Fatal(err)
	}
	dashboardText := string(dashboard)
	for _, want := range []string{"Recent activity", "Active tasks", "System status", "ChatGPT brain", "Autonomous worker health", "BuildDashboardSnapshot(s.eng)", "+ Delegate Task", "Review & Publish"} {
		if !strings.Contains(dashboardText, want) {
			t.Fatalf("production dashboard contract missing %q", want)
		}
	}
	for _, forbidden := range []string{"96.3%", "98 / 100", "fake worker", "synthetic health"} {
		if strings.Contains(strings.ToLower(dashboardText), strings.ToLower(forbidden)) {
			t.Fatalf("dashboard embedded invented operational telemetry %q", forbidden)
		}
	}
}
