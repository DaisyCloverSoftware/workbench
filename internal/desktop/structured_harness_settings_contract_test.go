package desktop

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDesktopSettingsUseStructuredHarnessAdapterNotLegacyCommandTemplate(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate desktop source directory")
	}
	body, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "settings_windows.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, want := range []string{
		"Structured harness adapter executable (optional)",
		"prefs.HarnessAdapterPath",
		`prefs.OpenClawCommand = ""`,
		`case "workbench-runner":`,
		"core.TestWorkbenchRunnerSSH(host)",
		"core.ValidateHarnessAdapterPath(adapter)",
		`case "legacy-harness-command":`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("desktop structured-harness settings contract missing %q", want)
		}
	}
	for _, forbidden := range []string{
		"setWindowText(s.controls[idHarnessCommand], prefs.OpenClawCommand)",
		"prefs.OpenClawCommand = strings.TrimSpace(windowText(s.controls[idHarnessCommand]))",
		"core.TestOpenClawSSH(",
		"advanced adapter: command {project} {prompt}",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("desktop restored legacy harness/runner behavior %q", forbidden)
		}
	}
}
