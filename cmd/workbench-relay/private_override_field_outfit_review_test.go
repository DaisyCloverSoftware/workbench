package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestPrivateRelayReviewCaptureAcceptsOnlyHost(t *testing.T) {
	if !isPrivateSafeHandsAction("run_override_rin_field_outfit_review_capture") {
		t.Fatal("review capture must be an explicit private safe-hands action")
	}
	args, err := json.Marshal(map[string]any{"host_id": "windows-test"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = executePrivateSafeHands(context.Background(), privateControlEnvelope{
		Version: 1, ID: "review-capture-project-rejection", Action: "run_override_rin_field_outfit_review_capture",
		Project: "runner://override", Args: args,
	}, "", "")
	if err == nil || !strings.Contains(err.Error(), "does not accept a project") {
		t.Fatalf("expected project rejection before host submission, got %v", err)
	}
	extra, err := json.Marshal(map[string]any{
		"host_id": "windows-test", "path": `C:\unsafe\candidate.blend`, "hash": "caller-supplied",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = executePrivateSafeHands(context.Background(), privateControlEnvelope{
		Version: 1, ID: "review-capture-extra-args", Action: "run_override_rin_field_outfit_review_capture", Args: extra,
	}, "", "")
	if err == nil {
		t.Fatal("caller-supplied path/hash fields were accepted")
	}
}
