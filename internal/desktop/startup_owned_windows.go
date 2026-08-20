//go:build windows

package desktop

import (
	"context"
	"fmt"
	"strings"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
	"github.com/DaisyCloverSoftware/workbench/internal/mcp"
)

// RunOwned is the production desktop startup path. processOwnershipConfirmed is
// supplied by cmd/workbench after its per-user named mutex init has run. The MCP
// bridge may move to another free port, so listener acquisition is deliberately
// not treated as an exclusivity lock. Durable interrupted work is recovered only
// when the named-mutex ownership proof exists.
func RunOwned(version string, processOwnershipConfirmed bool) error {
	store, err := core.NewStore()
	if err != nil {
		return fmt.Errorf("open Workbench state: %w", err)
	}
	eng, err := core.NewEngine(store)
	if err != nil {
		return fmt.Errorf("initialise Workbench: %w", err)
	}
	st := eng.State()
	var srv *mcp.Server
	var mcpURL, mcpErr string
	for port := st.Preferences.MCPPort; port < st.Preferences.MCPPort+20; port++ {
		candidate := mcp.New(eng, port, st.Preferences.MCPToken)
		if startErr := candidate.Start(); startErr == nil {
			srv = candidate
			mcpURL = candidate.URL()
			if port != st.Preferences.MCPPort {
				prefs := st.Preferences
				prefs.MCPPort = port
				_ = eng.SavePreferences(prefs)
			}
			break
		} else {
			mcpErr = startErr.Error()
		}
	}
	setDashboardMCPRuntime(srv != nil && strings.TrimSpace(mcpURL) != "")

	if CanRecoverInterruptedTasks(processOwnershipConfirmed) {
		_ = eng.ResumeInterruptedTasks()
	} else {
		warning := "Interrupted tasks were not resumed because Workbench could not acquire its per-user desktop ownership mutex."
		if strings.TrimSpace(mcpErr) == "" {
			mcpErr = warning
		} else {
			mcpErr = strings.TrimSpace(mcpErr) + " " + warning
		}
	}

	// The Windows host bridge is deliberately subordinate to the owned desktop
	// lifecycle. It opens no inbound listener and uses the same configured
	// Workbench Runner SSH target as the rest of the desktop. Bridge failures are
	// non-fatal to the UI and are retried by the agent while Workbench is open.
	hostBridgeCtx, stopHostBridge := context.WithCancel(context.Background())
	defer stopHostBridge()
	if host := strings.TrimSpace(st.Preferences.OpenClawSSHHost); host != "" {
		go func() {
			_ = core.RunWindowsHostBridgeAgent(hostBridgeCtx, host)
		}()
	}

	shell := &Shell{
		eng:      eng,
		mcp:      srv,
		mcpURL:   mcpURL,
		mcpErr:   mcpErr,
		version:  strings.TrimSpace(version),
		controls: map[int]uintptr{},
		page:     pageDashboard,
	}
	runningShell = shell
	defer func() { runningShell = nil }()
	return runProductionShell(shell)
}
