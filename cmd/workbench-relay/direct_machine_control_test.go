package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestPrivateRelayAdvertisesDirectMachineControls(t *testing.T) {
	for _, action := range []string{"inspect_machine", "run_machine_command"} {
		if !isPrivateSafeHandsAction(action) {
			t.Fatalf("direct machine action %q is not private safe-hands", action)
		}
	}
	b, err := privateChatCapabilitiesJSON()
	if err != nil {
		t.Fatal(err)
	}
	var got privateChatCapabilities
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	controls := strings.Join(got.ControlActions, " ")
	if !strings.Contains(controls, "inspect_machine") || !strings.Contains(controls, "run_machine_command") {
		t.Fatalf("direct machine controls missing from capabilities: %s", controls)
	}
	if got.OpenClawPolicy != "explicit_owner_request_only" {
		t.Fatalf("OpenClaw policy=%q, want explicit_owner_request_only", got.OpenClawPolicy)
	}
	openClaw := strings.Join(got.OpenClawOwns, " ")
	if strings.Contains(strings.ToLower(openClaw), "fallback") || !strings.Contains(openClaw, "owner_explicitly_authorized") {
		t.Fatalf("OpenClaw must be owner-authorized only and never an automatic fallback: %s", openClaw)
	}
	if got.AutonomousPurpose != "owner_selected_openclaw_execution_only_no_automatic_routing" {
		t.Fatalf("autonomous purpose permits ambiguous routing: %q", got.AutonomousPurpose)
	}
	chat := strings.Join(got.ChatGPTOwns, " ")
	if !strings.Contains(chat, "bounded_machine_inspection") || !strings.Contains(chat, "bounded_machine_mutation") {
		t.Fatalf("ChatGPT direct machine ownership missing: %s", chat)
	}
}

func TestPrivateRelayDirectMachineControlsDoNotAcceptProjectOrShell(t *testing.T) {
	shellArgs := json.RawMessage(`{"program":"bash","args":["-lc","kubectl get pods"]}`)
	_, err := executePrivateSafeHands(context.Background(), privateControlEnvelope{
		Version: 1,
		ID:      "machine_test_a",
		Action:  "inspect_machine",
		Args:    shellArgs,
	}, "", "")
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "allowlisted") {
		t.Fatalf("shell request should be rejected before execution, got %v", err)
	}

	_, err = executePrivateSafeHands(context.Background(), privateControlEnvelope{
		Version: 1,
		ID:      "machine_test_b",
		Action:  "inspect_machine",
		Project: "workbench",
		Args:    json.RawMessage(`{"program":"uptime"}`),
	}, "", "")
	if err == nil || !strings.Contains(err.Error(), "does not accept a project") {
		t.Fatalf("machine command should be host-scoped, got %v", err)
	}
}

func TestPrivateRelayMutationToolRejectsReadOnlyCommandClass(t *testing.T) {
	_, err := executePrivateSafeHands(context.Background(), privateControlEnvelope{
		Version: 1,
		ID:      "machine_test_c",
		Action:  "run_machine_command",
		Args:    json.RawMessage(`{"program":"uptime"}`),
	}, "", "")
	if err == nil || !strings.Contains(err.Error(), "read-only") {
		t.Fatalf("mutating machine tool must reject read-only command class, got %v", err)
	}
}

func TestUnsupportedDirectMachineOperationReturnsBoundaryWithoutOpenClawRouting(t *testing.T) {
	_, err := executePrivateControl(context.Background(), privateControlEnvelope{
		Version: 1,
		ID:      "unsupported_direct_001",
		Action:  "run_machine_command",
		Args:    json.RawMessage(`{"program":"kubectl","args":["delete","pod","example","-n","app-dev"]}`),
	}, "http://127.0.0.1:1", "unused")
	if err == nil {
		t.Fatal("unsupported direct operation should fail at the bounded Workbench policy")
	}
	low := strings.ToLower(err.Error())
	if !strings.Contains(low, "not allowlisted") {
		t.Fatalf("unsupported direct operation did not return a precise capability boundary: %v", err)
	}
	if strings.Contains(low, "openclaw") || strings.Contains(low, "autonomous") || strings.Contains(low, "fallback") {
		t.Fatalf("unsupported direct operation suggested or routed to an autonomous/OpenClaw path: %v", err)
	}
}
