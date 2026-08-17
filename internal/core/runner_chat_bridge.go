package core

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

const runnerGitRelayService = "workbench-github-relay.service"

// RunnerChatBridgeInfo is deliberately categorical and privacy-minimal. It
// lets a desktop explain whether the configured runner currently has a
// supported ChatGPT transport without returning relay repository names,
// filesystem paths, Git remotes, credentials, task content, or service output.
type RunnerChatBridgeInfo struct {
	Ready     bool   `json:"ready"`
	Transport string `json:"transport,omitempty"`
	Status    string `json:"status"`
}

func DetectRunnerChatBridge(ctx context.Context) *RunnerChatBridgeInfo {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return nil
	}

	probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	active := exec.CommandContext(probeCtx, "systemctl", "--user", "is-active", "--quiet", runnerGitRelayService).Run() == nil
	if !active {
		return classifyRunnerChatBridge(false, "")
	}

	showCtx, showCancel := context.WithTimeout(ctx, 2*time.Second)
	defer showCancel()
	out, err := exec.CommandContext(showCtx, "systemctl", "--user", "show", "--property=ExecStart", "--value", runnerGitRelayService).Output()
	if err != nil {
		return classifyRunnerChatBridge(true, "")
	}
	return classifyRunnerChatBridge(true, string(out))
}

func classifyRunnerChatBridge(active bool, execStart string) *RunnerChatBridgeInfo {
	if !active {
		return &RunnerChatBridgeInfo{
			Ready:     false,
			Transport: "git-relay",
			Status:    "ChatGPT Git relay not active on runner",
		}
	}

	// The relay installer records privacy mode in ExecStart. Inspect it locally
	// and return only a categorical result; never return the command itself.
	if strings.Contains(strings.ToLower(execStart), "--public-transport=false") {
		return &RunnerChatBridgeInfo{
			Ready:     true,
			Transport: "private-git-relay",
			Status:    "private ChatGPT relay active · bidirectional control ready",
		}
	}
	if strings.Contains(strings.ToLower(execStart), "--public-transport=true") {
		return &RunnerChatBridgeInfo{
			Ready:     false,
			Transport: "public-git-relay",
			Status:    "Git relay active · public/status-only mode",
		}
	}
	return &RunnerChatBridgeInfo{
		Ready:     false,
		Transport: "git-relay",
		Status:    "Git relay active · privacy mode could not be verified",
	}
}
