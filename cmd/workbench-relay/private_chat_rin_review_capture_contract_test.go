package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPrivateChatCapabilitiesAdvertiseRinFieldOutfitReviewCapture(t *testing.T) {
	b, err := privateChatCapabilitiesJSON()
	if err != nil {
		t.Fatal(err)
	}
	var manifest privateChatCapabilities
	if err := json.Unmarshal(b, &manifest); err != nil {
		t.Fatal(err)
	}

	const action = "run_override_rin_field_outfit_review_capture"
	if !containsString(manifest.ControlActions, action) {
		t.Fatalf("private manifest does not advertise %q", action)
	}
	if !isPrivateSafeHandsAction(action) {
		t.Fatalf("advertised action %q is not implemented as private safe hands", action)
	}
	for _, required := range []string{
		action,
		"sole bounded rendering exception",
		"exact current five-file review manifest",
		"protected worktree and all pinned source bytes remain unchanged",
		"visual and animation acceptance not_assessed",
		"AAA acceptance false",
	} {
		if !strings.Contains(manifest.WindowsHostBridgePolicy, required) {
			t.Fatalf("Windows host bridge policy is missing %q: %s", required, manifest.WindowsHostBridgePolicy)
		}
	}
	if strings.Contains(manifest.WindowsHostBridgePolicy, "rendering is not enabled by this manifest") {
		t.Fatal("manifest still claims all rendering is disabled after advertising the bounded review capture")
	}
}
