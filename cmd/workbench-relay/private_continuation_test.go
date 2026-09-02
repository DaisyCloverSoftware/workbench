package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestPrepareRelayTaskIntentRejectsOperationsMarkerWithoutOwnerAuthorization(t *testing.T) {
	_, _, err := prepareRelayTaskIntent("unused", "relay_ops_001", "/tmp/workbench", core.RelayOperationsIntentPrefix+" restart the bounded service")
	if err == nil || !strings.Contains(err.Error(), "routing metadata") || !strings.Contains(err.Error(), "owner authorization") {
		t.Fatalf("plain operations marker must not authorize OpenClaw: %v", err)
	}
}

func TestPrepareRelayTaskIntentPreservesExplicitlyAuthorizedOpenClawLane(t *testing.T) {
	raw := core.OpenClawExplicitAuthorizationPrefix + " " + core.RelayOperationsIntentPrefix + " restart the bounded service"
	intent, kind, err := prepareRelayTaskIntent("unused", "relay_ops_002", "/tmp/workbench", raw)
	if err != nil {
		t.Fatal(err)
	}
	if kind != "openclaw-operations" || !strings.HasPrefix(intent, core.OpenClawExplicitAuthorizationPrefix) || !strings.Contains(intent, core.RelayOperationsIntentPrefix) {
		t.Fatalf("kind=%q intent=%q", kind, intent)
	}
	if !core.IsOperationsTask(core.Task{Intent: intent}) {
		t.Fatal("explicitly authorized OpenClaw relay escaped operations lane")
	}
}

func TestPrepareRelayTaskIntentSealsContinuation(t *testing.T) {
	authFile := filepath.Join(t.TempDir(), "auth")
	if err := os.WriteFile(authFile, []byte("Bearer relay-secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	body := "Continue the development workflow autonomously through completion."
	intent, kind, err := prepareRelayTaskIntent(authFile, "relay_dev_003", project, core.RelayContinuationIntentPrefix+" "+body)
	if err != nil {
		t.Fatal(err)
	}
	if kind != "continuation" {
		t.Fatalf("kind=%q", kind)
	}
	clean, ok := core.ValidatePrivateRelayContinuationIntent(intent, project, "relay-secret")
	if !ok || clean != body {
		t.Fatalf("continuation seal invalid: ok=%t clean=%q", ok, clean)
	}
	if core.IsOperationsTask(core.Task{Intent: intent}) {
		t.Fatal("development continuation was incorrectly routed to operations")
	}
}

func TestPrepareRelayTaskIntentRejectsImplicitDevelopment(t *testing.T) {
	_, _, err := prepareRelayTaskIntent("unused", "relay_dev_004", "/tmp/workbench", "Implement the feature")
	if err == nil || !strings.Contains(err.Error(), "explicit") {
		t.Fatalf("implicit development handoff was not rejected: %v", err)
	}
}
