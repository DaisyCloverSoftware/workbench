package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withRunnerHarnessConfigPath(t *testing.T) string {
	t.Helper()
	old := runnerHarnessConfigPathOverride
	path := filepath.Join(t.TempDir(), "config", "runner-harness.json")
	runnerHarnessConfigPathOverride = path
	t.Cleanup(func() { runnerHarnessConfigPathOverride = old })
	return path
}

func makeRunnerHarnessAdapter(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := testHarnessAdapterPath(dir)
	if err := os.WriteFile(path, []byte("structured adapter fixture"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunnerHarnessConfigRoundTripStatusAndDelete(t *testing.T) {
	configPath := withRunnerHarnessConfigPath(t)
	adapter := makeRunnerHarnessAdapter(t)

	saved, err := SaveRunnerHarnessAdapter(adapter)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(adapter)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Version != 1 || saved.AdapterPath != resolved || saved.UpdatedAt.IsZero() {
		t.Fatalf("unexpected saved runner harness config: %#v", saved)
	}

	loaded, configured, err := LoadRunnerHarnessConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !configured || loaded.AdapterPath != resolved {
		t.Fatalf("runner harness config did not round-trip: %#v configured=%t", loaded, configured)
	}
	status := RunnerHarnessConfigurationStatus()
	if !status.Configured || !status.Available || status.AdapterName != filepath.Base(resolved) {
		t.Fatalf("unexpected runner harness status: %#v", status)
	}
	if strings.Contains(status.Status, resolved) || strings.Contains(status.AdapterName, filepath.Dir(resolved)) {
		t.Fatalf("safe runner harness status leaked full host path: %#v", status)
	}

	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() {
		t.Fatalf("runner harness config is not a regular file: %v", info.Mode())
	}

	if err := DeleteRunnerHarnessAdapter(); err != nil {
		t.Fatal(err)
	}
	if status := RunnerHarnessConfigurationStatus(); status.Configured || status.Available || status.Status != "not configured" {
		t.Fatalf("runner harness config survived delete: %#v", status)
	}
}

func TestRunnerHarnessConfigRejectsInvalidAndCorruptState(t *testing.T) {
	configPath := withRunnerHarnessConfigPath(t)
	if _, err := SaveRunnerHarnessAdapter(filepath.Join(t.TempDir(), "missing-adapter")); err == nil {
		t.Fatal("missing adapter was saved")
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatal(err)
	}
	cases := []string{
		`{"version":1,"adapter_path":"/tmp/a","unknown":true}`,
		`{"version":1,"adapter_path":"/tmp/a"}{"version":1,"adapter_path":"/tmp/b"}`,
		`{"version":2,"adapter_path":"/tmp/a"}`,
		`{"version":1,"adapter_path":""}`,
	}
	for _, body := range cases {
		if err := os.WriteFile(configPath, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, _, err := LoadRunnerHarnessConfig(); err == nil {
			t.Fatalf("corrupt runner harness config was accepted: %s", body)
		}
	}

	if err := os.WriteFile(configPath, []byte(strings.Repeat("x", maxRunnerHarnessConfigBytes+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadRunnerHarnessConfig(); err == nil {
		t.Fatal("oversized runner harness config was accepted")
	}
}

func TestRunnerHarnessConfigIsVisibleOnlyInRunnerInventoryMode(t *testing.T) {
	withRunnerHarnessConfigPath(t)
	adapter := makeRunnerHarnessAdapter(t)
	if _, err := SaveRunnerHarnessAdapter(adapter); err != nil {
		t.Fatal(err)
	}
	prefs := Preferences{AvoidWorkUsage: true}
	base := []Provider{{ID: "codex", Name: "Codex", Command: "codex", Installed: true, Authenticated: true, Cost: CostScarce, Priority: 80, CanWrite: true}}

	desktop := providerInventoryWithConfiguredHarnessMode(base, prefs, false)
	if len(desktop) != 1 || desktop[0].ID != "codex" {
		t.Fatalf("runner-private adapter leaked into desktop inventory: %#v", desktop)
	}

	runner := providerInventoryWithConfiguredHarnessMode(base, prefs, true)
	if len(runner) != 2 {
		t.Fatalf("runner inventory=%#v want adapter + Codex", runner)
	}
	if runner[1].ID != StructuredHarnessProviderID && runner[0].ID != StructuredHarnessProviderID {
		t.Fatalf("runner inventory did not contain structured adapter: %#v", runner)
	}
	// routeCandidates sees an already-materialised structured provider and must
	// still preserve the normal included-before-scarce ordering. Production
	// workbench-runner materialises it through the same mode helper above.
	candidates := routeCandidates(runner, prefs, Task{})
	if len(candidates) != 2 || candidates[0].ID != StructuredHarnessProviderID || candidates[1].ID != "codex" {
		t.Fatalf("runner-private adapter was not routed before scarce Work: %#v", candidates)
	}
}

func TestCorruptRunnerHarnessConfigDoesNotPoisonBaseRouting(t *testing.T) {
	configPath := withRunnerHarnessConfigPath(t)
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("not-json"), 0o600); err != nil {
		t.Fatal(err)
	}
	base := []Provider{{ID: "claude", Name: "Claude", Command: "claude", Installed: true, Authenticated: true, Cost: CostIncluded, Priority: 40, CanWrite: true}}
	providers := providerInventoryWithConfiguredHarnessMode(base, Preferences{}, true)
	if len(providers) != 1 || providers[0].ID != "claude" {
		t.Fatalf("corrupt optional runner harness config poisoned base providers: %#v", providers)
	}
}

func TestRunnerRequestNeverCarriesHarnessConfiguration(t *testing.T) {
	req := RunnerRequest{
		Task:            Task{ID: "task-private", ProjectPath: "/desktop/project", Intent: "work"},
		AvoidWorkUsage:  true,
		AllowMeteredAPI: false,
	}
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ToLower(string(body))
	for _, forbidden := range []string{"harness_adapter_path", "openclaw_command", "runner-harness", "adapter_path", "publication_target", "remote_url"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("runner request exposed private operator configuration %q: %s", forbidden, body)
		}
	}
}
