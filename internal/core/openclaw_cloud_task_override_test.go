package core

import "testing"

func TestExplicitCloudModelOverrideWinsOnlyInsideOpenClawLadder(t *testing.T) {
	catalog := OpenClawCloudCatalog{
		DefaultModel: "openai/gpt-5.3-codex-spark",
		Models: []OpenClawCloudModel{
			{Key: "openai/gpt-5.3-codex-spark", Provider: "openai", Available: true, Default: true},
			{Key: "openai/gpt-5.6-sol", Provider: "openai", Available: true},
			{Key: "anthropic/claude-sonnet-4-6", Provider: "anthropic", Available: true},
		},
	}
	ranked := rankOpenClawCloudModelsWithOverride(catalog, "Investigate a security incident and data loss", "openai/gpt-5.3-codex-spark")
	if len(ranked) == 0 || ranked[0].Key != "openai/gpt-5.3-codex-spark" {
		t.Fatalf("explicit per-task override must lead the OpenClaw ladder: %#v", ranked)
	}

	providers := []Provider{
		{ID: "antigravity", Name: "Antigravity", Command: "agy", Installed: true, Authenticated: true, CanWrite: true, Cost: CostIncluded, Priority: 20},
		{ID: "openclaw", Name: "OpenClaw", Command: "openclaw", Installed: true, Authenticated: true, CanWrite: true, Cost: CostIncluded, Priority: 50},
		{ID: "codex", Name: "Codex", Command: "codex", Installed: true, Authenticated: true, CanWrite: true, Cost: CostScarce, Priority: 80},
	}
	outer := routeCandidates(providers, Preferences{AvoidWorkUsage: true}, Task{Mode: TaskModeDevelopment, CloudModelOverride: "openai/gpt-5.3-codex-spark"})
	if len(outer) != 2 || outer[0].ID != "antigravity" || outer[1].ID != "codex" {
		t.Fatalf("cloud model override must not make OpenClaw eligible for ordinary development routing: %#v", outer)
	}

	authorized := routeCandidates(providers, Preferences{AvoidWorkUsage: true}, Task{
		Mode:                    TaskModeOperations,
		OpenClawOwnerAuthorized: true,
		ProjectPath:             t.TempDir(),
		CloudModelOverride:      "openai/gpt-5.3-codex-spark",
	})
	if len(authorized) != 1 || authorized[0].ID != "openclaw" {
		t.Fatalf("explicitly owner-authorized OpenClaw operation should keep the OpenClaw route: %#v", authorized)
	}
}

func TestStaleOrCoolingCloudOverrideFallsBackToLiveLadder(t *testing.T) {
	catalog := OpenClawCloudCatalog{
		DefaultModel: "openai/gpt-5.3-codex-spark",
		Models: []OpenClawCloudModel{
			{Key: "openai/gpt-5.3-codex-spark", Provider: "openai", Available: true, Default: true},
			{Key: "openai/gpt-5.6-sol", Provider: "openai", Available: true, Cooling: true},
		},
	}
	for _, requested := range []string{"openai/gpt-5.7-removed", "openai/gpt-5.6-sol"} {
		ranked := rankOpenClawCloudModelsWithOverride(catalog, "routine engineering", requested)
		if len(ranked) == 0 || ranked[0].Key != "openai/gpt-5.3-codex-spark" {
			t.Fatalf("stale/cooling override %q should fall back to live default: %#v", requested, ranked)
		}
	}
}
