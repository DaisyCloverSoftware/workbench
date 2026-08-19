package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestOversizedPrivateControlResultFailsOnlyThatRequest(t *testing.T) {
	b, err := marshalPrivateControlOutbox(privateControlOutbox{
		Version:   1,
		ID:        "oversize-12345678",
		Action:    "inspect_machine",
		Status:    "completed",
		Result:    map[string]any{"output": strings.Repeat("safe-output-", 30000)},
		UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		t.Fatalf("oversized result must become a bounded failed outbox, not wedge the relay: %v", err)
	}
	if len(b) > maxPrivateControlResult {
		t.Fatalf("bounded failure still exceeds relay limit: %d", len(b))
	}
	var got privateControlOutbox
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Status != "failed" || got.Result != nil || !strings.Contains(got.Error, "narrower bounded result") {
		t.Fatalf("unexpected oversized-result isolation envelope: %#v", got)
	}
}

func TestPrivateControlErrorIsBounded(t *testing.T) {
	b, err := marshalPrivateControlOutbox(privateControlOutbox{
		Version:   1,
		ID:        "long-error-12345678",
		Action:    "inspect_machine",
		Status:    "failed",
		Error:     strings.Repeat("diagnostic ", 10000),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(b) > maxPrivateControlResult {
		t.Fatalf("bounded error exceeds relay limit: %d", len(b))
	}
	var got privateControlOutbox
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.Error, "error truncated by Workbench") {
		t.Fatalf("long error was not bounded: %d bytes", len(got.Error))
	}
}

func TestPrivateControlPriorityPublishesMaintenanceFirst(t *testing.T) {
	if privateControlPriority("update_workbench") >= privateControlPriority("update_status") {
		t.Fatal("update_workbench must outrank update_status so its acknowledgement is published before restart")
	}
	if privateControlPriority("update_status") >= privateControlPriority("inspect_machine") {
		t.Fatal("maintenance status must outrank ordinary backlog diagnostics")
	}
	if maxPrivateControlsPerPoll <= 0 || maxPrivateControlsPerPoll > 32 {
		t.Fatalf("private control poll bound is unreasonable: %d", maxPrivateControlsPerPoll)
	}
}

func TestPrivateUpdateRestartGraceAllowsAcknowledgementPush(t *testing.T) {
	if privateUpdateDelay < 30*time.Second {
		t.Fatalf("private update restart grace is too short for contended Git acknowledgement: %s", privateUpdateDelay)
	}
}
