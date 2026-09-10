package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestPrivateRelayAcceptsOnlyHostForFieldOutfitInspect(t *testing.T) {
	if !isPrivateSafeHandsAction("run_override_rin_field_outfit_inspect") {
		t.Fatal("run_override_rin_field_outfit_inspect must be an explicit private safe-hands action")
	}
	args, err := json.Marshal(map[string]any{"host_id": "windows-test"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = executePrivateSafeHands(context.Background(), privateControlEnvelope{
		Version: 1, ID: "field-outfit-project-rejection", Action: "run_override_rin_field_outfit_inspect",
		Project: "runner://override", Args: args,
	}, "", "")
	if err == nil || !strings.Contains(err.Error(), "does not accept a project") {
		t.Fatalf("expected project rejection before host submission, got %v", err)
	}

	extra, err := json.Marshal(map[string]any{
		"host_id": "windows-test", "root": `C:\unsafe`, "path": `C:\unsafe\candidate.blend`, "executable": "blender.exe",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = executePrivateSafeHands(context.Background(), privateControlEnvelope{
		Version: 1, ID: "field-outfit-extra-args", Action: "run_override_rin_field_outfit_inspect", Args: extra,
	}, "", "")
	if err == nil {
		t.Fatal("caller-supplied root/path/executable fields were accepted")
	}
}
