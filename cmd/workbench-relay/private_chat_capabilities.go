package main

import (
	"bytes"
	"encoding/json"
)

const privateChatCapabilitiesPath = "WORKBENCH_CAPABILITIES.json"

type privateChatCapabilities struct {
	Protocol                   int      `json:"protocol"`
	WorkbenchVersion           string   `json:"workbench_version"`
	Transport                  string   `json:"transport"`
	PrimaryBrain               string   `json:"primary_brain"`
	PreferredWriteTransport    string   `json:"preferred_write_transport"`
	MCPRole                    string   `json:"mcp_role"`
	NoModelCreditRequired      bool     `json:"no_model_credit_required"`
	FreshChatBootstrap         string   `json:"fresh_chat_bootstrap"`
	ChatGPTOwns                []string `json:"chatgpt_owns"`
	OpenClawOwns               []string `json:"openclaw_owns"`
	ControlRequest             string   `json:"control_request"`
	ControlResult              string   `json:"control_result"`
	ControlActions             []string `json:"control_actions"`
	AutonomousRequest          string   `json:"autonomous_request"`
	AutonomousResult           string   `json:"autonomous_result"`
	AutonomousPurpose          string   `json:"autonomous_purpose"`
	AutonomousOperationsPrefix string   `json:"autonomous_operations_prefix,omitempty"`
	AttentionAnswer            string   `json:"attention_answer"`
	ProjectReference           string   `json:"project_reference"`
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
		FreshChatBootstrap:      "Use connected GitHub to locate the user's private repository whose name contains workbench-relay, then read WORKBENCH_CAPABILITIES.json and WORKBENCH_CHATGPT.md.",
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
			"run_machine_command",
			"run_operations_script",
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
