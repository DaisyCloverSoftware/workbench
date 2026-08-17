package core

import (
	"strings"
	"testing"
)

func TestChatGPTBootstrapInstructionIsSafeAndActionable(t *testing.T) {
	text := ChatGPTBootstrapInstruction()
	for _, want := range []string{
		"connected GitHub app",
		"workbench-relay",
		"WORKBENCH_CAPABILITIES.json",
		"WORKBENCH_CHATGPT.md",
		"ChatGPT is the primary brain/coder",
		"safe hands before autonomous delegation",
		"Codex/Work as a last resort",
		"metered API fallback opt-in",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("ChatGPT bootstrap instruction missing %q", want)
		}
	}
	if LooksSecret(text) {
		t.Fatal("ChatGPT bootstrap instruction must not contain secret-like material")
	}
	for _, forbidden := range []string{"Authorization:", "Bearer ", "127.0.0.1", "user@", "ssh ", "token="} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("ChatGPT bootstrap instruction contains transport-specific credential material %q", forbidden)
		}
	}
}
