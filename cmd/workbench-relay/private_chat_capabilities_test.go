package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestPrivateChatCapabilitiesAreMachineReadableAndSecretFree(t *testing.T) {
	b, err := privateChatCapabilitiesJSON()
	if err != nil {
		t.Fatal(err)
	}
	if core.LooksSecret(string(b)) {
		t.Fatal("private relay capabilities must not contain secret-like material")
	}
	for _, literal := range []string{
		"relay/control/<id>.json",
		"relay/control-outbox/<id>.json",
		"relay/inbox/<id>.json",
		"relay/outbox/<id>.json",
		"relay/answers/<id>.json",
		"[workbench:operations]",
		"[workbench:continuation]",
		"WORKBENCH_WAIT_GITHUB_ACTIONS:",
		"authenticated_development_continuation",
		"github_actions",
		"inspect_machine",
		"inspect_machine_batch",
		"run_machine_command",
		"run_operations_script",
		"preferred_write_transport",
		"no_model_credit_required",
		"machine_read_batch_policy",
		"operations_script_policy",
		"builtin_operations_commit_rule",
		"builtin_readonly_operations",
		"workbench-health.sh",
		"cluster-health.sh",
		"namespace-health.sh",
		"workbench-relay",
	} {
		if !strings.Contains(string(b), literal) {
			t.Fatalf("capability manifest should keep protocol ownership/path markers human-readable: missing %q in %s", literal, b)
		}
	}
	var manifest privateChatCapabilities
	if err := json.Unmarshal(b, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Protocol != 1 || manifest.WorkbenchVersion != relayVersion || manifest.Transport != "private-git-relay" || manifest.PrimaryBrain != "chatgpt" {
		t.Fatalf("unexpected private relay manifest identity: %+v", manifest)
	}
	if manifest.PreferredWriteTransport != "private-git-relay" || !manifest.NoModelCreditRequired {
		t.Fatalf("personal-plan write transport/model-credit contract missing: %+v", manifest)
	}
	if !strings.Contains(manifest.MCPRole, "read_fetch_on_personal_plans") || !strings.Contains(manifest.MCPRole, "full_mcp") {
		t.Fatalf("MCP plan-role contract missing: %q", manifest.MCPRole)
	}
	if !strings.Contains(manifest.FreshChatBootstrap, "connected GitHub") || !strings.Contains(manifest.FreshChatBootstrap, "WORKBENCH_CAPABILITIES.json") || !strings.Contains(manifest.FreshChatBootstrap, "built-in operations") || !strings.Contains(manifest.FreshChatBootstrap, "continuation/dependency") {
		t.Fatalf("fresh-chat relay bootstrap missing: %q", manifest.FreshChatBootstrap)
	}
	for _, want := range []string{"1-8", "sequentially", "exact inspect_machine read-only policy", "does not stop later reads", "no run_machine_command_batch", "one-at-a-time"} {
		if !strings.Contains(manifest.MachineReadBatchPolicy, want) {
			t.Fatalf("machine-read batch policy missing %q: %q", want, manifest.MachineReadBatchPolicy)
		}
	}
	for _, want := range []string{"scripts/ops/", "Git-tracked", "detached worktree", "literal argv", "SHA-256"} {
		if !strings.Contains(manifest.OperationsScriptPolicy, want) {
			t.Fatalf("operations-script safety contract missing %q: %q", want, manifest.OperationsScriptPolicy)
		}
	}
	if !strings.Contains(manifest.BuiltinOperationsCommitRule, "DaisyCloverSoftware/workbench") || !strings.Contains(manifest.BuiltinOperationsCommitRule, "40-character") || !strings.Contains(manifest.BuiltinOperationsCommitRule, "runner://workbench") {
		t.Fatalf("built-in operation commit rule is incomplete: %q", manifest.BuiltinOperationsCommitRule)
	}
	if len(manifest.BuiltinReadonlyOperations) != 3 {
		t.Fatalf("expected three built-in read-only operations, got %+v", manifest.BuiltinReadonlyOperations)
	}
	wantOps := map[string]struct {
		path string
		args []string
	}{
		"workbench_health": {path: "scripts/ops/workbench-health.sh"},
		"cluster_health":   {path: "scripts/ops/cluster-health.sh"},
		"namespace_health": {path: "scripts/ops/namespace-health.sh", args: []string{"<namespace>"}},
	}
	for _, operation := range manifest.BuiltinReadonlyOperations {
		want, ok := wantOps[operation.Name]
		if !ok {
			t.Fatalf("unexpected built-in operation: %+v", operation)
		}
		if operation.Project != "runner://workbench" || operation.Path != want.path || strings.TrimSpace(operation.Purpose) == "" {
			t.Fatalf("invalid built-in operation contract: %+v", operation)
		}
		if len(operation.Args) != len(want.args) {
			t.Fatalf("unexpected args for %s: %+v", operation.Name, operation.Args)
		}
		for i := range want.args {
			if operation.Args[i] != want.args[i] {
				t.Fatalf("unexpected args for %s: %+v", operation.Name, operation.Args)
			}
		}
	}
	for _, want := range []string{
		"source_code", "git", "github", "pull_requests", "ci", "github_actions",
		"bounded_machine_inspection", "bounded_machine_mutation", "committed_operations_script_execution",
	} {
		if !containsString(manifest.ChatGPTOwns, want) {
			t.Fatalf("capability manifest must keep %q with ChatGPT: %+v", want, manifest.ChatGPTOwns)
		}
	}
	for _, want := range []string{
		"optional_autonomous_machine_investigation",
		"machine_operations_outside_the_direct_allowlist",
		"long_running_operator_reasoning_when_explicitly_available",
	} {
		if !containsString(manifest.OpenClawOwns, want) {
			t.Fatalf("capability manifest missing optional OpenClaw responsibility %q: %+v", want, manifest.OpenClawOwns)
		}
	}
	for _, want := range []string{
		"list_projects", "ensure_github_project", "get_task", "list_tasks", "read_file", "apply_patch",
		"run_safe_command", "inspect_machine", "inspect_machine_batch", "run_machine_command", "run_operations_script", "search_memory", "save_context", "update_workbench",
	} {
		if !containsString(manifest.ControlActions, want) {
			t.Fatalf("capability manifest missing control action %q", want)
		}
	}
	if containsString(manifest.ControlActions, "run_machine_command_batch") {
		t.Fatal("capability manifest must not advertise a mutation batch action")
	}
	if manifest.ControlRequest != "relay/control/<id>.json" ||
		manifest.ControlResult != "relay/control-outbox/<id>.json" ||
		manifest.AutonomousRequest != "relay/inbox/<id>.json" ||
		manifest.AutonomousResult != "relay/outbox/<id>.json" ||
		manifest.AutonomousPurpose != "optional_machine_side_autonomy_fallback_or_authenticated_development_continuation" ||
		manifest.AutonomousOperationsPrefix != core.RelayOperationsIntentPrefix ||
		manifest.AutonomousContinuationPrefix != core.RelayContinuationIntentPrefix ||
		manifest.GitHubActionsWaitPrefix != "WORKBENCH_WAIT_GITHUB_ACTIONS:" ||
		!strings.Contains(manifest.AutonomousContinuationPurpose, "already_authorised_development") ||
		manifest.AttentionAnswer != "relay/answers/<id>.json" {
		t.Fatalf("unexpected private relay protocol paths/purpose: %+v", manifest)
	}
	if !strings.Contains(manifest.ProjectReference, "exact opaque ref returned by list_projects") {
		t.Fatalf("capability manifest missing opaque project-ref rule: %q", manifest.ProjectReference)
	}
}

func TestPrivateChatCapabilitiesMatchImplementedControlActions(t *testing.T) {
	b, err := privateChatCapabilitiesJSON()
	if err != nil {
		t.Fatal(err)
	}
	var manifest privateChatCapabilities
	if err := json.Unmarshal(b, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, action := range manifest.ControlActions {
		if isPrivateSafeHandsAction(action) {
			continue
		}
		switch action {
		case "save_memory", "search_memory", "save_context", "get_context", "update_workbench":
			continue
		default:
			t.Fatalf("capability manifest advertises unimplemented control action %q", action)
		}
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
