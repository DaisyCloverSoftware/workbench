package main

import (
	"strings"
	"testing"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestPrivateChatGuideBootstrapsFreshChatWithoutSecrets(t *testing.T) {
	guide := string(privateChatGuide)
	for _, want := range []string{
		"ChatGPT is the primary reasoning and coding brain",
		"relay/control/<id>.json",
		"relay/control-outbox/<id>.json",
		"list_projects",
		"ensure_github_project",
		"read_file",
		"apply_patch",
		"run_safe_command",
		"relay/inbox/<id>.json",
		"[workbench:operations]",
		"supervised operations lane",
		"never the transport layer",
		"do not ask the user to watch OpenClaw",
		"Codex/Work remains a last resort",
		"workbench-relay",
		"repository search",
		"WORKBENCH_CAPABILITIES.json",
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
