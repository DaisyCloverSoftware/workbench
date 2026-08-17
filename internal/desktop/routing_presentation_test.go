package desktop

import (
	"strings"
	"testing"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestPrimaryChatRoutingRowIsExplicitlyFirstClassBrain(t *testing.T) {
	resetRunnerProviderInventory()
	t.Cleanup(resetRunnerProviderInventory)
	target, line := primaryChatRoutingRow(true)
	if target != "brain:chatgpt" {
		t.Fatalf("target=%q want brain:chatgpt", target)
	}
	for _, want := range []string{"PRIMARY", "This PC", "ChatGPT Chat", "local MCP endpoint listening", "chat transport not yet verified", "included"} {
		if !strings.Contains(line, want) {
			t.Fatalf("primary Chat row %q missing %q", line, want)
		}
	}
	if strings.HasPrefix(line, "●") {
		t.Fatalf("a loopback listener alone must not claim that ChatGPT transport is verified: %q", line)
	}
}

func TestPrimaryChatRoutingRowReportsUnverifiedTransportTruthfully(t *testing.T) {
	resetRunnerProviderInventory()
	t.Cleanup(resetRunnerProviderInventory)
	_, line := primaryChatRoutingRow(false)
	if !strings.Contains(line, "Workbench transport not verified") || strings.HasPrefix(line, "●") {
		t.Fatalf("unverified Chat row is misleading: %q", line)
	}
}

func TestPrimaryChatRoutingRowUsesVerifiedPrivateRelayHealth(t *testing.T) {
	resetRunnerProviderInventory()
	t.Cleanup(resetRunnerProviderInventory)
	runnerProviderCache.Lock()
	runnerProviderCache.chatBridge = &core.RunnerChatBridgeInfo{Ready: true, Transport: "private-git-relay", Status: "private ChatGPT relay active · bidirectional control ready"}
	runnerProviderCache.Unlock()
	_, line := primaryChatRoutingRow(true)
	if !strings.HasPrefix(line, "●") || !strings.Contains(line, "bidirectional control ready") {
		t.Fatalf("verified private relay was not presented as ready: %q", line)
	}
	if strings.Contains(line, "ready via local MCP bridge") {
		t.Fatalf("verified relay row fell back to obsolete local-MCP claim: %q", line)
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
