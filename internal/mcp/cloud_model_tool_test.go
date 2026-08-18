package mcp

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestChatGPTToolSurfaceExposesOperationsNotCodingDelegation(t *testing.T) {
	var operation map[string]any
	for _, candidate := range toolsList() {
		switch candidate["name"] {
		case "delegate_task":
			t.Fatal("delegate_task must not be advertised to ChatGPT; ChatGPT owns the development loop")
		case "delegate_operation":
			operation = candidate
		}
	}
	if operation == nil {
		t.Fatal("delegate_operation tool is missing")
	}
	body, err := json.Marshal(operation)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, required := range []string{"machine-side", "OpenClaw", "never the coder", "source code", "Git/GitHub", "CI", "GitHub Actions"} {
		if !strings.Contains(text, required) {
			t.Fatalf("delegate_operation ownership contract missing %q in %s", required, text)
		}
	}
	if strings.Contains(text, "cloud_model") {
		t.Fatalf("machine operations tool must not expose coding-model routing: %s", text)
	}
	ann, ok := operation["annotations"].(map[string]any)
	if !ok {
		t.Fatalf("delegate_operation annotations missing: %#v", operation)
	}
	if ann["readOnlyHint"] != false || ann["destructiveHint"] != true || ann["openWorldHint"] != true {
		t.Fatalf("machine operations must be advertised as permission-worthy write/open-world actions: %#v", ann)
	}
}
