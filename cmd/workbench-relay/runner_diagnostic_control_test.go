package main

import (
	"context"
	"encoding/json"
	"testing"
)

func TestRunnerDiagnosticControlRejectsExtraAuthority(t *testing.T) {
	if !isPrivateSafeHandsAction("inspect_windows_actions_runner") {
		t.Fatal("missing diagnostic action")
	}
	for _, raw := range []string{`{"host_id":"windows_fixture","command":"anything"}`, `{"host_id":"windows_fixture","path":"anything"}`, `{"host_id":"windows_fixture","service":"anything"}`, `{"host_id":"windows_fixture","start":true}`} {
		env := privateControlEnvelope{Version: 1, ID: "diagnostic_fixture", Action: "inspect_windows_actions_runner", Args: json.RawMessage(raw)}
		if _, e := executePrivateSafeHands(context.Background(), env, "", ""); e == nil {
			t.Fatal("extra authority accepted")
		}
	}
	env := privateControlEnvelope{Version: 1, ID: "diagnostic_fixture", Action: "inspect_windows_actions_runner", Project: "runner://fixture", Args: json.RawMessage(`{}`)}
	if _, e := executePrivateSafeHands(context.Background(), env, "", ""); e == nil {
		t.Fatal("project accepted")
	}
}
