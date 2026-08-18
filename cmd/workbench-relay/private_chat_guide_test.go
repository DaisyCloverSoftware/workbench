package main

import (
	"strings"
	"testing"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestPrivateChatGuideBootstrapsFreshChatWithoutSecrets(t *testing.T) {
	guide := string(privateChatGuide)
	for _, want := range []string{
		"ChatGPT owns the software-development loop",
		"ChatGPT determines and writes the code",
		"ChatGPT owns Git and GitHub",
		"CI runs and GitHub Actions",
		"relay/control/<id>.json",
		"relay/control-outbox/<id>.json",
		"list_projects",
		"ensure_github_project",
		"read_file",
		"apply_patch",
		"run_safe_command",
		"relay/inbox/<id>.json",
		"machine-operations bridge",
		"[workbench:operations]",
		"supervised OpenClaw operations lane",
		"human is never the transport layer",
		"watch OpenClaw and type “continue”",
		"OpenClaw is an **operator only**",
		"workbench-relay",
		"repository search",
		"WORKBENCH_CAPABILITIES.json",
		"WORKBENCH_CHATGPT.md",
	} {
		if !strings.Contains(guide, want) {
			t.Fatalf("private ChatGPT guide missing %q", want)
		}
	}
	for _, forbidden := range []string{
		"ordinary autonomous coding",
		"ordinary development delegation",
		"coding worker according to its configured routing policy",
	} {
		if strings.Contains(strings.ToLower(guide), strings.ToLower(forbidden)) {
			t.Fatalf("private ChatGPT guide still advertises delegated development: %q", forbidden)
		}
	}
	if core.LooksSecret(guide) {
		t.Fatal("private ChatGPT guide must never contain secret-like material")
	}
	if len(privateChatGuide) == 0 || len(privateChatGuide) > 128<<10 {
		t.Fatalf("private ChatGPT guide size=%d outside bounded publish limit", len(privateChatGuide))
	}
}
