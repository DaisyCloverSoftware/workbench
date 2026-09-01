package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestActiveChatGPTOpenClawGovernanceSurfacesRequireExplicitOwnerSelection(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	required := map[string][]string{
		"cmd/workbench-relay/WORKBENCH_CHATGPT.md": {
			"OpenClaw is an owner-selected execution mode",
			"Only an explicit owner instruction naming OpenClaw",
			"effective OpenClaw authorization state is **DENIED**",
		},
		"docs/CHATGPT_BOOTSTRAP.md": {
			"OpenClaw is owner-opt-in only",
			"Failure of a direct capability never implicitly authorizes OpenClaw",
		},
		"docs/CHATGPT_SHARED_INTEGRATION.md": {
			"OpenClaw is owner-opt-in only on every transport",
			"Do not convert a direct-capability miss into autonomous/OpenClaw delegation",
		},
		"docs/PERSONAL_PRO_RELAY.md": {
			"Hard OpenClaw authorization boundary",
			"Availability does not constitute authorization",
		},
		"skills/workbench/SKILL.md": {
			"Hard OpenClaw authorization boundary",
			"effective authorization state is **DENIED**",
		},
		"plugins/chatgpt/SKILL.md": {
			"OpenClaw is owner-opt-in only",
			"[workbench:operations] is routing metadata, not owner consent",
		},
		"docs/GOVERNANCE.md": {
			"OpenClaw owner-authorization invariant",
			"effective OpenClaw authorization state is **DENIED**",
		},
		"docs/DECISIONS.md": {
			"WB-DEC-018 — OpenClaw is explicit-owner-request-only — CURRENT",
			"Automatic OpenClaw routing/fallback — REJECTED",
		},
		"ARCHITECTURE.md": {
			"OpenClaw is not part of normal routing",
			"installed/healthy state does not make a task eligible for OpenClaw",
		},
		"README.md": {
			"Only an explicit owner instruction naming OpenClaw authorizes its use",
			"OpenClaw does not participate in automatic cost/capability fallback",
		},
		"VISION.md": {
			"OpenClaw performs a machine operation only when the owner explicitly requested OpenClaw by name",
			"is never authorization to select OpenClaw",
		},
		"docs/operations/openclaw-agent-cli-contract.md": {
			"owner has explicitly assigned to OpenClaw by name",
			"unavailable to automatic routing",
		},
	}

	for rel, wants := range required {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		text := string(b)
		for _, want := range wants {
			if !strings.Contains(text, want) {
				t.Errorf("%s missing explicit-owner OpenClaw contract %q", rel, want)
			}
		}
	}
}

func TestActiveGovernanceContainsNoAutomaticOpenClawFallbackInstructions(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	files := []string{
		"cmd/workbench-relay/WORKBENCH_CHATGPT.md",
		"docs/CHATGPT_BOOTSTRAP.md",
		"docs/CHATGPT_SHARED_INTEGRATION.md",
		"docs/PERSONAL_PRO_RELAY.md",
		"skills/workbench/SKILL.md",
		"plugins/chatgpt/SKILL.md",
		"docs/GOVERNANCE.md",
		"ARCHITECTURE.md",
		"README.md",
		"VISION.md",
		"docs/operations/openclaw-agent-cli-contract.md",
	}
	forbidden := []string{
		"optional_machine_side_autonomy_fallback",
		"external autonomous operators are optional fallback capacity",
		"use OpenClaw only as optional autonomous operator fallback",
		"optional autonomous operator fallback",
		"delegate_operation remains optional autonomous operator capacity when an outcome cannot reasonably be decomposed",
		"use the autonomous inbox only when ordinary ChatGPT cannot sensibly complete",
		"use it only when a host/server/cluster/runtime outcome genuinely cannot be expressed",
		"Workbench uses OpenClaw only for delegated machine-side/autonomous operations that ChatGPT cannot reasonably complete",
	}

	for _, rel := range files {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		low := strings.ToLower(string(b))
		for _, phrase := range forbidden {
			if strings.Contains(low, strings.ToLower(phrase)) {
				t.Errorf("%s still contains prohibited automatic OpenClaw fallback instruction %q", rel, phrase)
			}
		}
	}
}
