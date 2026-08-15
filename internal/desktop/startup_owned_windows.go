//go:build windows

package desktop

import (
	"fmt"
	"strings"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
	"github.com/DaisyCloverSoftware/workbench/internal/mcp"
)

// RunOwned is the production desktop startup path. processOwnershipConfirmed is
// supplied by cmd/workbench after its per-user named mutex init has run. MCP
// listener ownership is an independent fallback if mutex creation was not
// available on an unusual Windows host. Durable interrupted work is recovered
// only after at least one ownership proof exists.
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

	if CanRecoverInterruptedTasks(processOwnershipConfirmed, srv != nil) {
		_ = eng.ResumeInterruptedTasks()
	} else {
		warning := "Interrupted tasks were not resumed because Workbench could not confirm exclusive control-plane ownership."
		if strings.TrimSpace(mcpErr) == "" {
			mcpErr = warning
		} else {
			mcpErr = strings.TrimSpace(mcpErr) + " " + warning
		}
	}

	shell := &Shell{
		eng:      eng,
		mcp:      srv,
		mcpURL:   mcpURL,
		mcpErr:   mcpErr,
		version:  strings.TrimSpace(version),
		controls: map[int]uintptr{},
		page:     pageWork,
	}
	runningShell = shell
	defer func() { runningShell = nil }()
	return shell.run()
}
