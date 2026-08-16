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
	if err := cmd.Start(); err != nil {
		return err
	}
	// The hidden wrapper exits after Windows START has created the real console.
	// Reap it asynchronously so the Win32 button handler never waits on process
	// startup, SSH authentication, Tailscale approval, or console lifetime.
	go func() { _ = cmd.Wait() }()
	return nil
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
	// START's first quoted argument is a window title. Use an explicit empty
	// title, then keep the child cmd.exe open so any SSH/Tailscale prompt or
	// failure remains readable. Backslash does not escape quotes in cmd.exe.
	launcher := "start \"\" cmd.exe /D /K " + sshLine
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
