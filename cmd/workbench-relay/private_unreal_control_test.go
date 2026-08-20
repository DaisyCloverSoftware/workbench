package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestPrivateRelayAcceptsBoundedUnrealSmokeAction(t *testing.T) {
	if !isPrivateSafeHandsAction("run_windows_unreal_smoke") {
		t.Fatal("run_windows_unreal_smoke must be an explicit private safe-hands action")
	}

	args, err := json.Marshal(map[string]any{"host_id": "windows-test"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = executePrivateSafeHands(context.Background(), privateControlEnvelope{
		Version: 1,
		ID:      "unreal-smoke-project-rejection",
		Action:  "run_windows_unreal_smoke",
		Project: "runner://workbench",
		Args:    args,
	}, "", "")
	if err == nil || !strings.Contains(err.Error(), "does not accept a project") {
		t.Fatalf("expected project rejection before host execution, got %v", err)
	}
}
