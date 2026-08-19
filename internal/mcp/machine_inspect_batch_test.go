package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestMachineInspectBatchToolIsReadOnlyAndBounded(t *testing.T) {
	tool := machineInspectBatchTool(strProp("program"), stringArrayProp("args"), intProp("timeout"))
	if tool["name"] != "inspect_machine_batch" {
		t.Fatalf("unexpected tool name: %#v", tool["name"])
	}
	annotations, ok := tool["annotations"].(map[string]any)
	if !ok || annotations["readOnlyHint"] != true || annotations["destructiveHint"] != false || annotations["openWorldHint"] != true {
		t.Fatalf("batch annotations are not read-only/open-world: %#v", tool["annotations"])
	}
	schema := tool["inputSchema"].(map[string]any)
	props := schema["properties"].(map[string]any)
	commands := props["commands"].(map[string]any)
	if commands["minItems"] != 1 || commands["maxItems"] != 8 {
		t.Fatalf("batch command bounds missing: %#v", commands)
	}
	item := commands["items"].(map[string]any)
	if item["additionalProperties"] != false {
		t.Fatalf("nested command schema must reject unknown fields: %#v", item)
	}
}

func TestMachineInspectBatchRequestParsingIsStrict(t *testing.T) {
	for _, args := range []map[string]any{
		{},
		{"commands": []any{}},
		{"commands": []any{map[string]any{"program": "hostname", "unknown": true}}},
		{"commands": []any{map[string]any{"program": "hostname", "args": []any{"ok", 123.0}}}},
		{"commands": []any{map[string]any{"program": "hostname", "timeout_seconds": 1.5}}},
	} {
		if _, err := machineInspectBatchRequestsArg(args); err == nil {
			encoded, _ := json.Marshal(args)
			t.Fatalf("invalid batch args accepted: %s", encoded)
		}
	}
}

func TestCallMachineInspectBatchContinuesAfterReadOnlyPolicyFailure(t *testing.T) {
	result := callMachineInspectBatch(context.Background(), map[string]any{
		"commands": []any{
			map[string]any{"program": "hostname", "timeout_seconds": 10.0},
			map[string]any{"program": "kubectl", "args": []any{"delete", "pod", "definitely-not-real"}, "timeout_seconds": 10.0},
			map[string]any{"program": "hostname", "timeout_seconds": 10.0},
		},
	})
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	if !strings.Contains(text, `"ok_count":2`) || !strings.Contains(text, `"failed_count":1`) {
		t.Fatalf("unexpected batch counts: %s", text)
	}
	if !strings.Contains(text, `"index":2`) || !strings.Contains(text, `"status":"completed"`) {
		t.Fatalf("later safe read did not continue: %s", text)
	}
}
