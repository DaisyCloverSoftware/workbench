package desktop

import (
	"fmt"
	"strings"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

const primaryChatProviderID = "chatgpt"

func primaryChatRoutingRow(bridgeReady bool) (target, line string) {
	mark := "○"
	status := "MCP bridge unavailable"
	if bridgeReady {
		mark = "●"
		status = "ready via local MCP bridge"
	}
	return "brain:" + primaryChatProviderID, fmt.Sprintf("%s PRIMARY · This PC · ChatGPT Chat  ·  %s  ·  included", mark, status)
}

func autonomousProviderRoutingLine(location string, provider core.Provider, ready bool) string {
	mark := "○"
	if ready {
		mark = "●"
	}
	role := "AUTONOMOUS"
	if provider.ID == "codex" || provider.Cost == core.CostScarce {
		role = "LAST RESORT"
	}
	return fmt.Sprintf("%s %s · %s · %s  ·  %s  ·  %s", mark, role, strings.TrimSpace(location), provider.Name, provider.Status, provider.Cost)
}

func runnerAutonomousRoutingLine(provider core.RunnerProviderInfo) string {
	mark := "○"
	if provider.Ready {
		mark = "●"
	}
	role := "AUTONOMOUS"
	if provider.ID == "codex" || provider.Cost == core.CostScarce {
		role = "LAST RESORT"
	}
	return fmt.Sprintf("%s %s · Runner · %s  ·  %s  ·  %s", mark, role, provider.Name, provider.Status, provider.Cost)
}
