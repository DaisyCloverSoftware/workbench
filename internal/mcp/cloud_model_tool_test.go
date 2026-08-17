package mcp

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDelegateTaskExposesOnlyInnerCloudModelOverride(t *testing.T) {
	var delegate map[string]any
	for _, candidate := range toolsList() {
		if candidate["name"] == "delegate_task" {
			delegate = candidate
			break
		}
	}
	if delegate == nil {
		t.Fatal("delegate_task tool is missing")
	}
	body, err := json.Marshal(delegate)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, required := range []string{"cloud_model", "OpenClaw", "never selects OpenClaw", "provider hierarchy"} {
		if !strings.Contains(text, required) {
			t.Fatalf("delegate_task cloud override contract missing %q in %s", required, text)
		}
	}
}
