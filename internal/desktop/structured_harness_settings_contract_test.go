package desktop

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func desktopSource(t *testing.T, name string) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate desktop source directory")
	}
	body, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), name))
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestDesktopSettingsUseStructuredHarnessAdapterNotLegacyCommandTemplate(t *testing.T) {
	text := desktopSource(t, "settings_windows.go")
	for _, want := range []string{
		"Structured harness adapter executable (optional)",
		"prefs.HarnessAdapterPath",
		`prefs.OpenClawCommand = ""`,
		"core.IsCodingWorkerProvider(provider)",
		`"local:"+provider.ID`,
		`"runner:"+provider.ID`,
		"core.StartRunnerProviderLogin(host, id)",
		"core.ValidateHarnessAdapterPath(adapter)",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("desktop structured-harness/host-aware settings contract missing %q", want)
		}
	}
	for _, forbidden := range []string{
		"setWindowText(s.controls[idHarnessCommand], prefs.OpenClawCommand)",
		"prefs.OpenClawCommand = strings.TrimSpace(windowText(s.controls[idHarnessCommand]))",
		"core.TestOpenClawSSH(",
		"advanced adapter: command {project} {prompt}",
		`case "workbench-runner":`,
		`case "chatgpt":`,
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("desktop restored legacy/mixed-host provider behavior %q", forbidden)
		}
	}
}

func TestWindowsShellFallbackCopyAndRetryControlsMatchCurrentTaskModel(t *testing.T) {
	text := desktopSource(t, "shell_windows.go")
	for _, want := range []string{
		"Structured harness adapter executable (optional)",
		"absolute path to one adapter executable; no arguments or shell placeholders",
		"item.Status == core.TaskWaitingRetry",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("Windows shell contract missing %q", want)
		}
	}
	for _, forbidden := range []string{
		"Optional custom harness command",
		"{project}",
		"{prompt}",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("Windows shell restored obsolete harness copy %q", forbidden)
		}
	}
}
