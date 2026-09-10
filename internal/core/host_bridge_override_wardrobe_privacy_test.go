package core

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOverrideRinWardrobeInventoryJSONWithholdsAbsoluteRoots(t *testing.T) {
	result := overrideRinWardrobeInventoryResult{
		SchemaVersion: 1,
		ArtifactID: "hostjob_privacy_test",
		ReposRoot: `F:\Repos`,
		WorktreeRoot: `F:\Repos\private-owner-folder\override-hardline`,
		Branch: overrideRinWardrobeBranch,
		HeadSHA: strings.Repeat("a", 40),
		ReadOnly: true,
		Candidates: []overrideRinWardrobeCandidate{{Path: "SourceAssets/RinVale/Clash_Shirt.blend", Extension: ".blend", Bytes: 42, HashStatus: "complete", SHA256: strings.Repeat("b", 64), Score: 15}},
		VisualAcceptance: "not_assessed",
		AnimationAcceptance: "not_assessed",
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, forbidden := range []string{"repos_root", "worktree_root", "private-owner-folder", `F:\\Repos`} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("wardrobe inventory JSON leaked absolute-root data %q: %s", forbidden, text)
		}
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["branch"] != overrideRinWardrobeBranch || decoded["read_only"] != true {
		t.Fatalf("privacy fix removed required non-sensitive evidence: %v", decoded)
	}
}
