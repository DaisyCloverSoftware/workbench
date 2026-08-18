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
		"ChatGPT writes and edits the code",
		"handles Git and GitHub",
		"runs and diagnoses CI",
		"operates GitHub Actions",
		"OpenClaw is the operator, never the coder",
		"shell, systemd, Docker, Kubernetes, Helm",
		"Never ask me to copy/paste prompts",
		"type \"continue\"",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("ChatGPT bootstrap instruction missing %q", want)
		}
	}
	for _, forbiddenWorkflow := range []string{
		"safe hands before autonomous delegation",
		"Codex/Work as a last resort",
		"metered API fallback opt-in",
	} {
		if strings.Contains(text, forbiddenWorkflow) {
			t.Fatalf("ChatGPT bootstrap still advertises delegated development policy %q", forbiddenWorkflow)
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
