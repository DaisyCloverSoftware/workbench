package core

import (
	"io/fs"
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
		"cmd/workbench-relay/private_chat_capabilities.go": {
			"OpenClawPolicy",
			"explicit_owner_request_only",
			"owner_selected_openclaw_execution_only_no_automatic_routing",
		},
		"cmd/workbench-relay/private_continuation.go": {
			"routing metadata, not owner authorization",
			"OpenClawExplicitAuthorizationPrefix",
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
			"routing metadata, not owner consent",
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
			"is not part of normal routing",
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
		"internal/core/engine_operations.go": {
			"OpenClawOwnerAuthorized: true",
			"explicit owner authorization naming OpenClaw is required",
		},
		"internal/core/engine.go": {
			"IsOperationsTask(t) && !t.OpenClawOwnerAuthorized",
			"OpenClaw is not an automatic coding fallback",
		},
		"internal/core/task_execution.go": {
			"Operations task lacks durable explicit owner authorization naming OpenClaw",
			"cannot be used as an automatic development or fallback provider",
		},
		"internal/core/openclaw_operations.go": {
			"owner explicitly assigned to OpenClaw by name",
			"OpenClaw authorization denied before process invocation",
		},
		"internal/mcp/server.go": {
			"OpenClaw is owner-opt-in only",
			"never call delegate_operation",
			"explicit_owner_request_only",
			"operations marker alone never authorizes OpenClaw",
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

func TestRepositoryContainsNoRetiredAutomaticOpenClawFallbackInstructions(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	forbidden := []string{
		"optional_machine_side_autonomy_fallback",
		"machine_operations_outside_the_direct_allowlist",
		"external autonomous operators are optional fallback capacity",
		"use OpenClaw only as optional autonomous operator fallback",
		"optional autonomous operator fallback",
		"delegate_operation remains optional autonomous operator capacity when an outcome cannot reasonably be decomposed",
		"use delegate_operation only as an optional autonomous fallback",
		"optional fallback only: delegate a machine-side outcome to OpenClaw",
		"optional autonomous fallback started",
		"only operations-marked tasks may run",
		"use the autonomous inbox only when ordinary ChatGPT cannot sensibly complete",
		"use it only when a host/server/cluster/runtime outcome genuinely cannot be expressed",
		"Workbench uses OpenClaw only for delegated machine-side/autonomous operations that ChatGPT cannot reasonably complete",
		"your job is only the machine-side operational work ChatGPT cannot execute itself",
		"falling back to OpenClaw",
	}

	allowedExtensions := map[string]bool{
		".go": true, ".md": true, ".json": true, ".txt": true,
		".yaml": true, ".yml": true, ".toml": true, ".sh": true, ".ps1": true,
	}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		// Tests must be able to quote retired wording in negative regression
		// assertions. The repository-wide instruction scan applies to active
		// implementation/configuration/documentation surfaces, not test literals.
		if strings.HasSuffix(strings.ToLower(d.Name()), "_test.go") {
			return nil
		}
		if !allowedExtensions[strings.ToLower(filepath.Ext(path))] {
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		low := strings.ToLower(string(b))
		rel, _ := filepath.Rel(root, path)
		for _, phrase := range forbidden {
			if strings.Contains(low, strings.ToLower(phrase)) {
				t.Errorf("%s still contains retired automatic OpenClaw fallback instruction %q", filepath.ToSlash(rel), phrase)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
