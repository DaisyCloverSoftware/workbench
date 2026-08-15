package core

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRouteCandidatesNeverExecutesLegacyHarnessTemplate(t *testing.T) {
	base := []Provider{
		{ID: "codex", Name: "Codex", Command: "codex", Installed: true, Authenticated: true, Cost: CostScarce, Priority: 80, CanWrite: true},
	}
	candidates := routeCandidates(base, Preferences{OpenClawCommand: "legacy {project} {prompt}", AvoidWorkUsage: true}, Task{})
	for _, provider := range candidates {
		if provider.ID == StructuredHarnessProviderID || provider.ID == "openclaw" || provider.ID == "legacy-harness-command" {
			t.Fatalf("legacy template became routable provider: %#v", provider)
		}
	}
}

func TestRouteCandidatesIncludesValidStructuredHarnessBeforeScarceWork(t *testing.T) {
	dir := t.TempDir()
	adapter := testHarnessAdapterPath(dir)
	if err := os.WriteFile(adapter, []byte("fixture"), 0o755); err != nil {
		t.Fatal(err)
	}
	base := []Provider{
		{ID: "codex", Name: "Codex", Command: "codex", Installed: true, Authenticated: true, Cost: CostScarce, Priority: 80, CanWrite: true},
	}
	candidates := routeCandidates(base, Preferences{HarnessAdapterPath: adapter, AvoidWorkUsage: true}, Task{})
	if len(candidates) != 2 {
		t.Fatalf("candidate count=%d want structured harness + Codex: %#v", len(candidates), candidates)
	}
	if candidates[0].ID != StructuredHarnessProviderID || candidates[0].Command == "" {
		t.Fatalf("structured harness was not preferred before scarce Work: %#v", candidates)
	}
	if candidates[1].ID != "codex" {
		t.Fatalf("unexpected fallback ordering: %#v", candidates)
	}
}

func TestCodingExecutorContainsNoLegacyHarnessShellTemplatePath(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate core source directory")
	}
	coreDir := filepath.Dir(thisFile)
	for _, name := range []string{"runner.go", "engine.go"} {
		body, err := os.ReadFile(filepath.Join(coreDir, name))
		if err != nil {
			t.Fatal(err)
		}
		text := string(body)
		if strings.Contains(text, "runCommandTemplate") {
			t.Fatalf("%s restored legacy shell-template coding execution", name)
		}
		if strings.Contains(text, "prefs.OpenClawCommand") {
			t.Fatalf("%s restored legacy OpenClawCommand coding route", name)
		}
	}
}
