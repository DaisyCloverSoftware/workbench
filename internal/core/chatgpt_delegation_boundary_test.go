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

func TestUnauthorizedOperationsTaskCannotReachAnyOperatorLane(t *testing.T) {
	for _, provider := range []Provider{
		{ID: "openclaw", Name: "OpenClaw", Installed: true, Authenticated: true, CanWrite: true, CanRunTools: true, Command: "openclaw"},
		{ID: "workbench-runner", Name: "Workbench Runner", Installed: true, Authenticated: true, CanWrite: true, CanRunTools: true, Command: "workbench-runner"},
		{ID: "claude", Name: "Claude", Installed: true, Authenticated: true, CanWrite: true, CanRunTools: true, Command: "claude"},
	} {
		task := Task{Origin: "chatgpt-mcp", Mode: TaskModeOperations, Intent: "Restart a service", ProjectPath: t.TempDir()}
		_, err := RunProviderIsolated(context.Background(), provider, task, Preferences{})
		if err == nil || !strings.Contains(err.Error(), "lacks durable explicit owner authorization naming OpenClaw") {
			t.Fatalf("provider %s received unauthorized Operations task: %v", provider.ID, err)
		}
	}
}

func TestOpenClawCannotBeUsedAsDevelopmentProvider(t *testing.T) {
	provider := Provider{ID: "openclaw", Name: "OpenClaw", Installed: true, Authenticated: true, CanWrite: true, CanRunTools: true, Command: "openclaw"}
	task := Task{Origin: "workbench-ui", Mode: TaskModeDevelopment, Intent: "Implement a feature", ProjectPath: t.TempDir()}
	_, err := RunProviderIsolated(context.Background(), provider, task, Preferences{})
	if err == nil || !strings.Contains(err.Error(), "cannot be used as an automatic development or fallback provider") {
		t.Fatalf("OpenClaw development-provider boundary was not enforced: %v", err)
	}
}
