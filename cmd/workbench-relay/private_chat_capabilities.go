package main

import "encoding/json"

const privateChatCapabilitiesPath = "WORKBENCH_CAPABILITIES.json"

type privateChatCapabilities struct {
	Protocol         int      `json:"protocol"`
	WorkbenchVersion string   `json:"workbench_version"`
	Transport        string   `json:"transport"`
	PrimaryBrain     string   `json:"primary_brain"`
	ControlRequest   string   `json:"control_request"`
	ControlResult    string   `json:"control_result"`
	ControlActions   []string `json:"control_actions"`
	AutonomousRequest string  `json:"autonomous_request"`
	AutonomousResult string   `json:"autonomous_result"`
	AttentionAnswer  string   `json:"attention_answer"`
	ProjectReference string   `json:"project_reference"`
}

func privateChatCapabilitiesJSON() ([]byte, error) {
	manifest := privateChatCapabilities{
		Protocol:         1,
		WorkbenchVersion: relayVersion,
		Transport:        "private-git-relay",
		PrimaryBrain:     "chatgpt",
		ControlRequest:   "relay/control/<id>.json",
		ControlResult:    "relay/control-outbox/<id>.json",
		ControlActions: []string{
			"list_projects",
			"list_files",
			"search_text",
			"read_file",
			"apply_patch",
			"run_safe_command",
			"save_note",
			"search_memory",
			"save_memory",
			"get_context",
			"save_context",
			"update_status",
			"update_workbench",
		},
		AutonomousRequest: "relay/inbox/<id>.json",
		AutonomousResult:  "relay/outbox/<id>.json",
		AttentionAnswer:   "relay/answers/<id>.json",
		ProjectReference:  "Use the exact opaque ref returned by list_projects; do not infer a runner filesystem path.",
	}
	b, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}
