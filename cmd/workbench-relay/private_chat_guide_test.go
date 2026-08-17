package main

import (
	"strings"
	"testing"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestPrivateChatGuideBootstrapsFreshChatWithoutSecrets(t *testing.T) {
	guide := string(privateChatGuide)
	for _, want := range []string{
		"ChatGPT is the primary reasoning/coding brain",
		"relay/control/<id>.json",
		"relay/control-outbox/<id>.json",
		"list_projects",
		"read_file",
		"apply_patch",
		"run_safe_command",
		"relay/inbox/<id>.json",
		"Codex/Work remains a last resort",
		"WORKBENCH_CHATGPT.md",
	} {
		if !strings.Contains(guide, want) {
			t.Fatalf("private ChatGPT guide missing %q", want)
		}
	}
	if core.LooksSecret(guide) {
		t.Fatal("private ChatGPT guide must never contain secret-like material")
	}
	if len(privateChatGuide) == 0 || len(privateChatGuide) > 128<<10 {
		t.Fatalf("private ChatGPT guide size=%d outside bounded publish limit", len(privateChatGuide))
	}
}
