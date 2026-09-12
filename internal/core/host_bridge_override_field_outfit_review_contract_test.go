package core

import (
	"os"
	"strings"
	"testing"
)

func TestOverrideRinFieldOutfitReviewPinsAreExactAndDistinct(t *testing.T) {
	if overrideRinFieldOutfitReviewExpectedHead != "3aed03a7d30b987bb76af5fd8230c941a2758e49" {
		t.Fatal("unexpected review protected HEAD pin")
	}
	if overrideRinFieldOutfitReviewStageID != "rin-field-outfit-review-v1-eec3287f2544d70f" {
		t.Fatalf("unexpected review stage identity: %q", overrideRinFieldOutfitReviewStageID)
	}
	if overrideRinFieldOutfitReviewStageID == overrideRinFieldOutfitStageID {
		t.Fatal("review candidate silently reused the released inspection stage")
	}
	if overrideRinFieldOutfitReviewManifestSHA256 != "eec3287f2544d70fd6e132f4c2179790e5f9e5cff90ffea6fc74308c14de5927" {
		t.Fatal("unexpected review manifest digest")
	}
	if len(overrideRinFieldOutfitReviewPinnedFiles) != 5 {
		t.Fatalf("expected five review pins, got %d", len(overrideRinFieldOutfitReviewPinnedFiles))
	}
	want := map[string]string{
		"integrated_outfit":   "7ec4d0db94d3f7ad71df357b1e0a801df6a5f873484abddca0803c84c2295469",
		"integrated_export":   "115515610f608694791ac377f6e7ad8d3e4fbda1b93700443da6c0fbfb7e7f21",
		"combat_boots_fit":    "9fa87f121fdddcf2cee530a46e59cb73fb02bc20e15d0505f9516a35898bb9f6",
		"cargo_fit":           "fdcf2290e86d70c6f80742de4e86e7441693571cbf885a3d7e2939e99619ccbe",
		"cargo_authored_base": "306960e889a5e674777f6077d012dc8bc31e181fdd085665487beb52b4f67d10",
	}
	for _, pin := range overrideRinFieldOutfitReviewPinnedFiles {
		if want[pin.Role] != pin.SHA256 || pin.Bytes <= 0 || len(pin.SHA256) != 64 || !strings.HasPrefix(pin.Path, overrideRinFieldOutfitSubtree+"/") {
			t.Fatalf("unexpected review pin: %+v", pin)
		}
	}
}

func TestOverrideRinFieldOutfitReviewWindowsSourceKeepsIsolatedRenderBoundary(t *testing.T) {
	b, err := os.ReadFile("host_bridge_override_field_outfit_review_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(b)
	for _, token := range []string{
		`"--disable-autoexec"`,
		`"--background"`,
		"overrideRinFieldOutfitReviewPinnedFiles",
		"ProtectedWorktreeUnchanged:    true",
		`VisualAcceptance:              "not_assessed"`,
		"bpy.ops.render.render(write_still=True)",
		"ImageBase64:",
		"external_images_suppressed",
	} {
		if !strings.Contains(source, token) {
			t.Fatalf("review implementation missing %q", token)
		}
	}
	for _, forbidden := range []string{
		"bpy.ops.wm.save",
		"bpy.ops.wm.save_as_mainfile",
		"cmd.exe",
		"powershell",
		"--python-expr",
	} {
		if strings.Contains(strings.ToLower(source), strings.ToLower(forbidden)) {
			t.Fatalf("review implementation widened boundary with %q", forbidden)
		}
	}
}

func TestOverrideRinFieldOutfitReviewResultKeepsTypedImageField(t *testing.T) {
	b, err := os.ReadFile("host_bridge_override_field_outfit_review.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "`json:\"image_base64\"`") {
		t.Fatal("review result lost the typed image_base64 field")
	}
}

func TestReleasedFieldOutfitInspectionStillDisablesRendering(t *testing.T) {
	b, err := os.ReadFile("host_bridge_override_field_outfit_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "bpy.ops.render") {
		t.Fatal("released structural inspection unexpectedly gained rendering")
	}
}
