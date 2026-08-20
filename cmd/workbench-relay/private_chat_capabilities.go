package main

import (
	"bytes"
	"encoding/json"
)

const privateChatCapabilitiesPath = "WORKBENCH_CAPABILITIES.json"

type privateBuiltinOperation struct {
	Name    string   `json:"name"`
	Project string   `json:"project"`
	Path    string   `json:"path"`
	Args    []string `json:"args,omitempty"`
	Purpose string   `json:"purpose"`
}

type privateChatCapabilities struct {
	Protocol                    int                       `json:"protocol"`
	WorkbenchVersion            string                    `json:"workbench_version"`
	Transport                   string                    `json:"transport"`
	PrimaryBrain                string                    `json:"primary_brain"`
	PreferredWriteTransport     string                    `json:"preferred_write_transport"`
	MCPRole                     string                    `json:"mcp_role"`
	NoModelCreditRequired       bool                      `json:"no_model_credit_required"`
	FreshChatBootstrap          string                    `json:"fresh_chat_bootstrap"`
	MachineReadBatchPolicy      string                    `json:"machine_read_batch_policy"`
	OperationsScriptPolicy      string                    `json:"operations_script_policy"`
	WindowsHostBridgePolicy     string                    `json:"windows_host_bridge_policy"`
	BuiltinOperationsCommitRule string                    `json:"builtin_operations_commit_rule"`
	BuiltinReadonlyOperations   []privateBuiltinOperation `json:"builtin_readonly_operations"`
	ChatGPTOwns                 []string                  `json:"chatgpt_owns"`
	OpenClawOwns                []string                  `json:"openclaw_owns"`
	ControlRequest              string                    `json:"control_request"`
	ControlResult               string                    `json:"control_result"`
	ControlActions              []string                  `json:"control_actions"`
	AutonomousRequest           string                    `json:"autonomous_request"`
	AutonomousResult            string                    `json:"autonomous_result"`
	AutonomousPurpose           string                    `json:"autonomous_purpose"`
	AutonomousOperationsPrefix  string                    `json:"autonomous_operations_prefix,omitempty"`
	AttentionAnswer             string                    `json:"attention_answer"`
	ProjectReference            string                    `json:"project_reference"`
}

func privateChatCapabilitiesJSON() ([]byte, error) {
	manifest := privateChatCapabilities{
		Protocol:                1,
		WorkbenchVersion:        relayVersion,
		Transport:               "private-git-relay",
		PrimaryBrain:            "chatgpt",
		PreferredWriteTransport: "private-git-relay",
		MCPRole:                 "optional_direct_read_fetch_on_personal_plans; full_tools_when_the_chatgpt_plan_supports_full_mcp",
		NoModelCreditRequired:   true,
		FreshChatBootstrap:      "Use connected GitHub to locate the user's private repository whose name contains workbench-relay, then read WORKBENCH_CAPABILITIES.json and WORKBENCH_CHATGPT.md. For common read-only health questions, prefer the built-in operations advertised in this manifest before issuing many individual machine reads.",
		MachineReadBatchPolicy:  "The private relay action inspect_machine_batch accepts no project and args.commands containing 1-8 objects with program, optional literal args, and optional timeout_seconds. Items execute sequentially through the exact inspect_machine read-only policy; one failed or rejected item does not stop later reads. There is deliberately no run_machine_command_batch; mutations remain one-at-a-time.",
		OperationsScriptPolicy:  "run_operations_script requires a project and a Git-tracked regular .sh beneath scripts/ops/. Without commit, Workbench executes exact local HEAD. With an optional full 40-character commit currently advertised by a credential-free github.com origin branch head, Workbench fetches into a disposable repository, creates a detached worktree at that exact commit, and executes without moving or modifying the registered checkout. Bash receives literal argv, never bash -c. Results include the commit and script SHA-256.",
		WindowsHostBridgePolicy: "Windows local-tool access is outbound-only through the existing Workbench Runner SSH target; no inbound Windows listener is opened. Use list_windows_hosts with empty args, run_windows_blender_version with args.host_id, then get_windows_host_job with args.job_id. This tranche can execute only exact local argv blender.exe --version after a second Windows-side allowlist check. There is no generic Windows command action and rendering is not enabled yet.",
		BuiltinOperationsCommitRule: "Resolve the current full 40-character head SHA of DaisyCloverSoftware/workbench main through connected GitHub and pass it as run_operations_script.args.commit. The registered runner://workbench checkout does not need to be updated.",
		BuiltinReadonlyOperations: []privateBuiltinOperation{
			{
				Name:    "workbench_health",
				Project: "runner://workbench",
				Path:    "scripts/ops/workbench-health.sh",
				Purpose: "Check Workbench binaries, MCP and relay services, loopback MCP health, and relay checkout cleanliness without reading credentials or restarting anything.",
			},
			{
				Name:    "cluster_health",
				Project: "runner://workbench",
				Path:    "scripts/ops/cluster-health.sh",
				Purpose: "Return a compact read-only cluster snapshot covering nodes, current abnormal pods, recent warnings, ARC assignments, Longhorn node readiness/schedulability, and attached unhealthy volumes.",
			},
			{
				Name:    "namespace_health",
				Project: "runner://workbench",
				Path:    "scripts/ops/namespace-health.sh",
				Args:    []string{"<namespace>"},
				Purpose: "Return a compact read-only namespace snapshot covering deployments, statefulsets, pods, jobs, PVCs, and recent Warning events.",
			},
		},
		ChatGPTOwns: []string{
			"reasoning",
			"source_code",
			"git",
			"github",
			"pull_requests",
			"ci",
			"github_actions",
			"release_orchestration",
			"bounded_machine_inspection",
			"bounded_machine_mutation",
			"committed_operations_script_execution",
			"bounded_windows_local_tool_execution",
		},
		OpenClawOwns: []string{
			"optional_autonomous_machine_investigation",
			"machine_operations_outside_the_direct_allowlist",
			"long_running_operator_reasoning_when_explicitly_available",
		},
		ControlRequest: "relay/control/<id>.json",
		ControlResult:  "relay/control-outbox/<id>.json",
		ControlActions: []string{
			"list_projects",
			"ensure_github_project",
			"get_task",
			"list_tasks",
			"list_files",
			"search_text",
			"read_file",
			"apply_patch",
			"run_safe_command",
			"inspect_machine",
			"inspect_machine_batch",
			"run_machine_command",
			"run_operations_script",
			"list_windows_hosts",
			"run_windows_blender_version",
			"get_windows_host_job",
			"save_note",
			"search_memory",
			"save_memory",
			"get_context",
			"save_context",
			"update_status",
			"update_workbench",
		},
		AutonomousRequest:          "relay/inbox/<id>.json",
		AutonomousResult:           "relay/outbox/<id>.json",
		AutonomousPurpose:          "optional_machine_side_autonomy_fallback",
		AutonomousOperationsPrefix: "[workbench:operations]",
		AttentionAnswer:            "relay/answers/<id>.json",
		ProjectReference:           "Use the exact opaque ref returned by list_projects; do not infer a runner filesystem path.",
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(manifest); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
