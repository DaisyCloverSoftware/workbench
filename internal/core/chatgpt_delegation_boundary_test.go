package core

import (
	"context"
	"strings"
	"testing"
)

func TestChatGPTDevelopmentTaskCannotBeFarmedOut(t *testing.T) {
	provider := Provider{ID: "claude", Name: "Claude", Installed: true, Authenticated: true, CanWrite: true, CanRunTools: true, Command: "claude"}
	task := Task{Origin: "chatgpt-mcp", Intent: "Implement the next feature", ProjectPath: t.TempDir()}
	_, err := RunProviderIsolated(context.Background(), provider, task, Preferences{})
	if err == nil || !strings.Contains(err.Error(), "ChatGPT owns coding, Git/GitHub, pull requests, CI and GitHub Actions") {
		t.Fatalf("ChatGPT development delegation was not refused: %v", err)
	}
}

func TestChatGPTOperationsTaskMayReachOperatorLane(t *testing.T) {
	provider := Provider{ID: "claude", Name: "Claude", Installed: true, Authenticated: true, CanWrite: true, CanRunTools: true, Command: "claude"}
	task := Task{Origin: "chatgpt-mcp", Mode: TaskModeOperations, Intent: "Restart a service", ProjectPath: t.TempDir()}
	_, err := RunProviderIsolated(context.Background(), provider, task, Preferences{})
	if err == nil || !strings.Contains(err.Error(), "operations are reserved for the cluster runner/OpenClaw operator lane") {
		t.Fatalf("operations task did not reach the operator-only boundary: %v", err)
	}
}
