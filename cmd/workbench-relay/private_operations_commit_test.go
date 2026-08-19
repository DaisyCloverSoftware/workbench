package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestPrivateOperationsScriptCommitIsStructuredAndValidated(t *testing.T) {
	_, _ = privateSafeHandsFixture(t)

	_, err := executePrivateControl(context.Background(), privateControlEnvelope{
		Version: 1,
		ID:      "ops-commit-12345678",
		Action:  "run_operations_script",
		Project: "sample",
		Args:    json.RawMessage(`{"path":"scripts/ops/verify.sh","commit":"deadbeef"}`),
	}, "http://127.0.0.1:1", "unused")
	if err == nil || !strings.Contains(err.Error(), "40-character Git SHA") {
		t.Fatalf("structured commit was not passed to core validation: %v", err)
	}

	_, err = executePrivateControl(context.Background(), privateControlEnvelope{
		Version: 1,
		ID:      "ops-origin-12345678",
		Action:  "run_operations_script",
		Project: "sample",
		Args:    json.RawMessage(`{"path":"scripts/ops/verify.sh","source_url":"https://example.invalid/repo.git"}`),
	}, "http://127.0.0.1:1", "unused")
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("relay must reject caller-supplied remote URLs: %v", err)
	}
}
