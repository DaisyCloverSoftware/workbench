//go:build windows

package desktop

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

const idClusterProjectsReady = 3199

type clusterProjectDiscoveryResult struct {
	shell         *Shell
	host          string
	response      core.RunnerToolResponse
	err           error
	runnerVersion string
	upgradeNeeded bool
	updateAttempt bool
}

var clusterProjectDiscovery = struct {
	sync.Mutex
	pending *clusterProjectDiscoveryResult
}{}

func (s *Shell) startClusterProjectDiscovery(host string) {
	host = strings.TrimSpace(host)
	if host == "" {
		return
	}
	procEnableWindow.Call(s.controls[idAddProject], 0)
	go func() {
		result := &clusterProjectDiscoveryResult{shell: s, host: host}
		result.response, result.err = discoverClusterProjects(host)
		if result.err != nil {
			version, probeErr := core.TestWorkbenchRunnerSSH(host)
			if probeErr == nil {
				result.runnerVersion = strings.TrimSpace(version)
				result.upgradeNeeded = result.runnerVersion != "" && result.runnerVersion != core.Version
			}
		}
		postClusterProjectDiscoveryResult(result)
	}()
}

func (s *Shell) continueClusterProjectDiscovery() {
	clusterProjectDiscovery.Lock()
	result := clusterProjectDiscovery.pending
	clusterProjectDiscovery.pending = nil
	clusterProjectDiscovery.Unlock()
	if result == nil || result.shell != s {
		procEnableWindow.Call(s.controls[idAddProject], 1)
		return
	}

	if result.upgradeNeeded && !result.updateAttempt {
		procEnableWindow.Call(s.controls[idAddProject], 1)
		question := "The cluster runner reports Workbench " + result.runnerVersion + ", while this app is Workbench " + core.Version + ".\r\n\r\nCluster project discovery requires the matching runner protocol. Install the latest verified Workbench cluster update on that runner now?"
		if messageBox(s.hwnd, "Update cluster runner", question, mbYesNo|mbIconInformation) == idYes {
			s.updateRunnerAndRediscover(result.host)
		}
		return
	}

	procEnableWindow.Call(s.controls[idAddProject], 1)
	if result.err != nil {
		messageBox(s.hwnd, "Cluster projects unavailable", "Workbench could not read the configured runner project list. Open Settings and click Rescan to verify/update the runner connection, then try again.", mbOK|mbIconWarning)
		return
	}
	if len(result.response.Projects) == 0 {
		messageBox(s.hwnd, "No cluster projects found", "The configured runner responded, but no Git repositories were found directly under its authorised project roots.", mbOK|mbIconInformation)
		return
	}
	selection, ok := chooseRunnerProjects(s.hwnd, result.response.Projects)
	if !ok {
		return
	}
	added, err := s.eng.RegisterRunnerProjects(selection)
	if err != nil {
		messageBox(s.hwnd, "Cannot add cluster project", err.Error(), mbOK|mbIconWarning)
		return
	}
	message := fmt.Sprintf("Workbench selected %d cluster Git repository/project(s).", len(selection))
	if added > 0 {
		message += fmt.Sprintf(" %d new project(s) were added to the Work list.", added)
	} else {
		message += " They were already registered."
	}
	message += "\r\n\r\nCluster projects stay on the runner; Workbench does not copy them onto this PC."
	messageBox(s.hwnd, "Cluster project ready", message, mbOK|mbIconInformation)
}

func (s *Shell) updateRunnerAndRediscover(host string) {
	procEnableWindow.Call(s.controls[idAddProject], 0)
	go func() {
		result := &clusterProjectDiscoveryResult{shell: s, host: host, updateAttempt: true}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		_, updateErr := core.UpdateWorkbenchRunnerSSH(ctx, host)
		cancel()
		if updateErr != nil {
			result.err = updateErr
			postClusterProjectDiscoveryResult(result)
			return
		}
		result.response, result.err = discoverClusterProjects(host)
		postClusterProjectDiscoveryResult(result)
	}()
}

func discoverClusterProjects(host string) (core.RunnerToolResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return core.RunRunnerToolSSH(ctx, host, core.RunnerToolRequest{Action: "list_projects"})
}

func postClusterProjectDiscoveryResult(result *clusterProjectDiscoveryResult) {
	if result == nil || result.shell == nil || result.shell.hwnd == 0 {
		return
	}
	clusterProjectDiscovery.Lock()
	clusterProjectDiscovery.pending = result
	clusterProjectDiscovery.Unlock()
	procPostMessageW.Call(result.shell.hwnd, wmCommand, uintptr(idClusterProjectsReady), 0)
}
