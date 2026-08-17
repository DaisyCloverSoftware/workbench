package desktop

import (
	"strings"
	"testing"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestPrimaryChatRoutingRowIsExplicitlyFirstClassBrain(t *testing.T) {
	target, line := primaryChatRoutingRow(true)
	if target != "brain:chatgpt" {
		t.Fatalf("target=%q want brain:chatgpt", target)
	}
	for _, want := range []string{"PRIMARY", "This PC", "ChatGPT Chat", "ready via local MCP bridge", "included"} {
		if !strings.Contains(line, want) {
			t.Fatalf("primary Chat row %q missing %q", line, want)
		}
	}
}

func TestPrimaryChatRoutingRowReportsUnavailableBridgeTruthfully(t *testing.T) {
	_, line := primaryChatRoutingRow(false)
	if !strings.Contains(line, "MCP bridge unavailable") || strings.HasPrefix(line, "●") {
		t.Fatalf("unavailable Chat row is misleading: %q", line)
	}
}

func TestAutonomousRoutingMakesCodexLastResort(t *testing.T) {
	codex := core.Provider{ID: "codex", Name: "OpenAI Codex / Work", Status: "CLI detected", Cost: core.CostScarce}
	line := autonomousProviderRoutingLine("This PC", codex, true)
	if !strings.Contains(line, "LAST RESORT") {
		t.Fatalf("Codex row must be labelled LAST RESORT: %q", line)
	}
	ordinary := core.Provider{ID: "claude", Name: "Anthropic Claude Code", Status: "ready", Cost: core.CostIncluded}
	line = autonomousProviderRoutingLine("This PC", ordinary, true)
	if !strings.Contains(line, "AUTONOMOUS") || strings.Contains(line, "LAST RESORT") {
		t.Fatalf("ordinary worker role is misleading: %q", line)
	}
}

func TestChatRemainsNonExecutableDespitePrimaryPresentation(t *testing.T) {
	chat := core.Provider{ID: "chatgpt", CanWrite: false, CanRunTools: false}
	if core.IsCodingWorkerProvider(chat) {
		t.Fatal("ChatGPT primary brain must not become an executable coding worker")
	}
}
