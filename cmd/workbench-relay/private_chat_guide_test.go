package main

import (
	"strings"
	"testing"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestPrivateChatGuideBootstrapsFreshChatWithoutSecrets(t *testing.T) {
	guide := string(privateChatGuide)
	for _, want := range []string{
		"ChatGPT owns the software-development and routine machine-operations loop",
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
		"inspect_machine",
		"run_machine_command",
		"run_operations_script",
		"save_context",
		"scripts/ops/workbench-health.sh",
		"scripts/ops/cluster-health.sh",
		"scripts/ops/namespace-health.sh",
		"DaisyCloverSoftware/workbench",
		"full 40-character head SHA",
		"built-in zero-credit read-only operations",
		"no shell or AI worker",
		"relay/inbox/<id>.json",
		"Durable Development continuation lane",
		"[workbench:continuation]",
		"WORKBENCH_WAIT_GITHUB_ACTIONS:",
		"waiting_dependency",
		"authenticated continuation seal",
		"Duplicate active waits",
		"optional autonomous machine-operations bridge",
		"[workbench:operations]",
		"Optional supervised OpenClaw operations lane",
		"Except for the authenticated `[workbench:continuation]` lane above",
		"human is never the transport layer",
		"optional operator capacity",
		"workbench-relay",
		"repository search",
		"WORKBENCH_CAPABILITIES.json",
		"WORKBENCH_CHATGPT.md",
		"resume it rather than requiring a new `continue` prompt",
	} {
		if !strings.Contains(strings.ToLower(guide), strings.ToLower(want)) {
			t.Fatalf("private ChatGPT guide missing %q", want)
		}
	}
	for _, forbidden := range []string{
		"ordinary autonomous coding",
		"ordinary development delegation",
		"coding worker according to its configured routing policy",
		"OpenClaw is an **operator only**",
	} {
		if strings.Contains(strings.ToLower(guide), strings.ToLower(forbidden)) {
			t.Fatalf("private ChatGPT guide still advertises obsolete delegated-development/ownership wording: %q", forbidden)
		}
	}
	if core.LooksSecret(guide) {
		t.Fatal("private ChatGPT guide must never contain secret-like material")
	}
	if len(privateChatGuide) == 0 || len(privateChatGuide) > 128<<10 {
		t.Fatalf("private ChatGPT guide size=%d outside bounded publish limit", len(privateChatGuide))
	}
}
