package core

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestOverrideRinFieldOutfitPinsAreExact(t *testing.T) {
	if overrideRinFieldOutfitExpectedHead != "3aed03a7d30b987bb76af5fd8230c941a2758e49" {
		t.Fatal("unexpected protected HEAD pin")
	}
	if overrideRinFieldOutfitSubtree != "SourceAssets/RinWardrobe" {
		t.Fatal("unexpected subtree pin")
	}
	if len(overrideRinFieldOutfitPinnedFiles) != 5 {
		t.Fatalf("expected five pinned field-outfit files, got %d", len(overrideRinFieldOutfitPinnedFiles))
	}
	for _, pin := range overrideRinFieldOutfitPinnedFiles {
		if pin.Role == "" || !strings.HasPrefix(pin.Path, overrideRinFieldOutfitSubtree+"/") || pin.Bytes <= 0 || len(pin.SHA256) != 64 {
			t.Fatalf("invalid field-outfit pin: %+v", pin)
		}
	}
	production, ok := overrideRinFieldOutfitPinnedByPath("SourceAssets/RinWardrobe/AST_RL_CHR_RIN_FIELD_OUTFIT.production.blend")
	if !ok || production.SHA256 != "7ec41284367e3e8638ba2fec33eeb64608a166b7473c5c42bf71b34b1e22a1db" {
		t.Fatalf("production blend pin changed: %+v", production)
	}
}

func TestOverrideRinFieldOutfitResultDoesNotSerializeLocalRoots(t *testing.T) {
	encoded, err := json.Marshal(overrideRinFieldOutfitInspectResult{SchemaVersion: 1, StageID: "safe-stage", StagedFiles: []overrideRinFieldOutfitStagedFile{{Path: "SourceAssets/RinWardrobe/test.blend"}}})
	if err != nil {
		t.Fatal(err)
	}
	lower := strings.ToLower(string(encoded))
	for _, forbidden := range []string{"repos_root", "worktree_root", "stage_root", "cache_root", `f:\\repos`} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("serialized field-outfit result leaked %q: %s", forbidden, encoded)
		}
	}
}

func TestOverrideRinFieldOutfitWindowsSourceKeepsSealedInspectionBoundary(t *testing.T) {
	b, err := os.ReadFile("host_bridge_override_field_outfit_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(b)
	required := []string{
		"FileSHA256(",
		`"--disable-autoexec"`,
		`"--background"`,
		"bpy.data.libraries.load(blend_path, link=False)",
		"ProtectedWorktreeUnchanged: true",
		"MutationPerformedSource: false",
	}
	for _, token := range required {
		if !strings.Contains(source, token) {
			t.Fatalf("Windows field-outfit implementation missing %q", token)
		}
	}
	forbidden := []string{
		"bpy.data.filepath",
		"bpy.ops.render",
		"--python-expr",
		"cmd.exe",
		"powershell",
		`"--disable-autoexec", blendPath`,
	}
	for _, token := range forbidden {
		if strings.Contains(strings.ToLower(source), strings.ToLower(token)) {
			t.Fatalf("Windows field-outfit implementation widened boundary with %q", token)
		}
	}
}
