package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfiguredHarnessProviderRequiresOneRealExecutableFile(t *testing.T) {
	dir := t.TempDir()
	adapter := filepath.Join(dir, "adapter")
	if err := os.WriteFile(adapter, []byte("adapter fixture"), 0o755); err != nil {
		t.Fatal(err)
	}
	provider, ok := configuredHarnessProvider(Preferences{HarnessAdapterPath: adapter})
	if !ok {
		t.Fatal("valid structured adapter was not configured")
	}
	if provider.ID != StructuredHarnessProviderID || provider.Command == "" || !provider.Installed || !provider.Authenticated || !provider.CanWrite {
		t.Fatalf("unexpected structured harness provider: %#v", provider)
	}
	if _, ok := configuredHarnessProvider(Preferences{HarnessAdapterPath: filepath.Join(dir, "missing")}); ok {
		t.Fatal("missing structured adapter was treated as routable")
	}
}

func TestProviderInventoryShowsLegacyCommandOnlyAsDisabledMigrationWarning(t *testing.T) {
	providers := providerInventoryWithConfiguredHarness(nil, Preferences{OpenClawCommand: "legacy {project} {prompt}"})
	if len(providers) != 1 {
		t.Fatalf("provider count=%d want 1 migration warning", len(providers))
	}
	legacy := providers[0]
	if legacy.ID != "legacy-harness-command" || legacy.Installed || legacy.Authenticated || legacy.CanWrite || legacy.Command != "" {
		t.Fatalf("legacy shell template became executable provider: %#v", legacy)
	}
}

func TestProviderInventoryKeepsStructuredAndLegacyEntriesDistinctDuringMigration(t *testing.T) {
	dir := t.TempDir()
	adapter := filepath.Join(dir, "adapter")
	if err := os.WriteFile(adapter, []byte("fixture"), 0o755); err != nil {
		t.Fatal(err)
	}
	providers := providerInventoryWithConfiguredHarness(nil, Preferences{
		HarnessAdapterPath: adapter,
		OpenClawCommand:    "legacy --project {project}",
	})
	if len(providers) != 2 {
		t.Fatalf("provider count=%d want structured provider plus migration warning", len(providers))
	}
	if providers[0].ID != StructuredHarnessProviderID || !providers[0].Installed {
		t.Fatalf("structured adapter missing: %#v", providers)
	}
	if providers[1].ID != "legacy-harness-command" || providers[1].Installed {
		t.Fatalf("legacy migration warning was not kept non-executable: %#v", providers)
	}
}
