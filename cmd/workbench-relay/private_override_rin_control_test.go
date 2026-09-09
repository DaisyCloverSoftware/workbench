package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestPrivateRelayAcceptsOnlyHostForCanonicalRinExport(t *testing.T) {
	if !isPrivateSafeHandsAction("run_override_rin_canonical_export") {
		t.Fatal("run_override_rin_canonical_export must be an explicit private safe-hands action")
	}
	args, err := json.Marshal(map[string]any{"host_id": "windows-test"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = executePrivateSafeHands(context.Background(), privateControlEnvelope{
		Version: 1, ID: "rin-export-project-rejection", Action: "run_override_rin_canonical_export",
		Project: "runner://override", Args: args,
	}, "", "")
	if err == nil || !strings.Contains(err.Error(), "does not accept a project") {
		t.Fatalf("expected project rejection before host submission, got %v", err)
	}

	extra, err := json.Marshal(map[string]any{
		"host_id": "windows-test", "project": "C:/unsafe", "script": "arbitrary.py",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = executePrivateSafeHands(context.Background(), privateControlEnvelope{
		Version: 1, ID: "rin-export-extra-args", Action: "run_override_rin_canonical_export", Args: extra,
	}, "", "")
	if err == nil {
		t.Fatal("caller-supplied project/script fields were accepted")
	}
}
