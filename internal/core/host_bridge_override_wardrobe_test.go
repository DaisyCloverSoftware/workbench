package core

import (
	"strings"
	"testing"
)

func TestOverrideRinWardrobeCandidateScoreIsNarrow(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"SourceAssets/RinVale/Clash_Shirt.blend", true},
		{"Content/Wardrobe/Cargo_Trousers.uasset", true},
		{"Content/Rin/Combat_Boots.uasset", true},
		{"Textures/Clash_Shirt_Graphic.png", true},
		{"Content/Environment/FactoryWall.uasset", false},
		{"Source/Runtime/StringHelpers.cpp", false},
		{"Saved/RinVale/readme.txt", false},
	}
	for _, tt := range tests {
		t.Run(strings.ReplaceAll(tt.path, "/", "_"), func(t *testing.T) {
			got := overrideRinWardrobeCandidateScore(tt.path) > 0
			if got != tt.want {
				t.Fatalf("candidate score match for %q = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestSubmitOverrideRinWardrobeInventoryJobIsSealed(t *testing.T) {
	t.Setenv("WORKBENCH_HOST_BRIDGE_STATE_DIR", t.TempDir())
	hostID := "windows_wardrobe_test"
	_, err := RecordHostBridgeHeartbeat(HostBridgeHeartbeat{
		HostID: hostID, Label: "Wardrobe Test Host", Platform: HostBridgePlatformWindows, Arch: "amd64",
	})
	if err != nil {
		t.Fatal(err)
	}
	job, err := SubmitOverrideRinWardrobeInventoryJob(hostID)
	if err != nil {
		t.Fatal(err)
	}
	if job.HostID != hostID || job.Status != "queued" {
		t.Fatalf("unexpected host job: %+v", job)
	}
	if job.Spec.Tool != HostBridgeToolWorkbench || job.Spec.Operation != HostBridgeOperationOverrideRinWardrobeInventory {
		t.Fatalf("unexpected sealed inventory spec: %+v", job.Spec)
	}
	if len(job.Spec.Tool)+len(job.Spec.Operation) == 0 {
		t.Fatal("sealed inventory job lost its fixed operation identity")
	}
}

func TestOverrideRinWardrobeFixedBoundary(t *testing.T) {
	if overrideRinWardrobeReposRoot != `F:\Repos` {
		t.Fatalf("unexpected repository root %q", overrideRinWardrobeReposRoot)
	}
	if overrideRinWardrobeBranch != "agent/hardline-premium-slice-20260822" {
		t.Fatalf("unexpected protected branch %q", overrideRinWardrobeBranch)
	}
}
