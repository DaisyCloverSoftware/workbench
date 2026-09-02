package main

import (
	"strings"
	"testing"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestPrivateChatGuideBootstrapsFreshChatWithoutSecrets(t *testing.T) {
	guide := string(privateChatGuide)
	for _, want := range []string{
		"ChatGPT is the primary brain and owns engineering",
		"private Workbench control relay is the normal server/cluster/host/runtime execution transport",
		"relay/control/<id>.json",
		"relay/control-outbox/<id>.json",
		"Direct machine operations do not require OpenClaw",
		"OpenClaw is an owner-selected execution mode",
		"Only an explicit owner instruction naming OpenClaw authorizes its use",
		"effective OpenClaw authorization state is **DENIED**",
		"inspect_machine",
		"inspect_machine_batch",
		"run_machine_command",
		"run_safe_command",
		"run_operations_script",
		"scripts/ops/*.sh",
		"list_projects",
		"ensure_github_project",
		"read_file",
		"apply_patch",
		"relay/inbox/<id>.json",
		"[workbench:operations]",
		"only routing metadata and is not proof of owner consent",
		"current capabilities override historical assumptions",
		"Direct-capability failure never authorizes OpenClaw",
		"WORKBENCH_CAPABILITIES.json",
	} {
		if !strings.Contains(strings.ToLower(guide), strings.ToLower(want)) {
			t.Fatalf("private ChatGPT guide missing %q", want)
		}
	}
	for _, forbidden := range []string{
		"fallback to OpenClaw",
		"optional fallback capacity",
		"optional autonomous operator fallback",
		"use OpenClaw only as optional autonomous operator fallback",
		"use the optional autonomous operations lane only when",
	} {
		if strings.Contains(strings.ToLower(guide), strings.ToLower(forbidden)) {
			t.Fatalf("private ChatGPT guide still advertises automatic OpenClaw fallback wording: %q", forbidden)
		}
	}
	if core.LooksSecret(guide) {
		t.Fatal("private ChatGPT guide must never contain secret-like material")
	}
	if len(privateChatGuide) == 0 || len(privateChatGuide) > 128<<10 {
		t.Fatalf("private ChatGPT guide size=%d outside bounded publish limit", len(privateChatGuide))
	}
}
