//go:build windows

package core

import (
	"errors"
	"os/exec"
	"strings"
)

func runRunnerSSHConsole(host string, remoteArgs []string) error {
	return launchRunnerSSHConsole(host, remoteArgs)
}

func startRunnerSSHConsole(host string, remoteArgs []string) error {
	return launchRunnerSSHConsole(host, remoteArgs)
}

func launchRunnerSSHConsole(host string, remoteArgs []string) error {
	cmd, err := runnerSSHConsoleLauncher(host, remoteArgs)
	if err != nil {
		return err
	}
	// The hidden wrapper exits after Windows START has created the real console.
	// The inner cmd.exe uses /K so SSH gets genuine console input/output and the
	// window remains visible after SSH exits, allowing the operator to read the
	// result before closing it and pressing Rescan in Workbench.
	return cmd.Run()
}

func runnerSSHConsoleLauncher(host string, remoteArgs []string) (*exec.Cmd, error) {
	parts := []string{
		"ssh",
		"-t",
		"-o", "ConnectTimeout=10",
		"-o", "StrictHostKeyChecking=accept-new",
		host,
	}
	parts = append(parts, remoteArgs...)
	for _, part := range parts {
		if !runnerSSHConsoleTokenSafe(part) {
			return nil, errors.New("runner SSH console command contains an unsafe token")
		}
	}
	sshLine := strings.Join(parts, " ")
	launcher := `start "Workbench Runner SSH" cmd.exe /D /S /K "` + sshLine + `"`
	cmd := exec.Command("cmd.exe", "/D", "/S", "/C", launcher)
	configureChildProcess(cmd, false)
	return cmd, nil
}

func runnerSSHConsoleTokenSafe(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			continue
		}
		switch r {
		case '@', '.', '_', '-', ':', '/', '[', ']', '$', '=':
			continue
		default:
			return false
		}
	}
	return true
}
