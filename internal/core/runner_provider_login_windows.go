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
	// The interactive child owns its own Windows console and cmd.exe /K keeps
	// that console alive after SSH exits. Reap the child asynchronously so the
	// Win32 button handler never waits on authentication or console lifetime.
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
	// Do not nest cmd.exe behind START. START's quoting and process handoff can
	// outlive or discard the shell that /K was intended to keep open. Launch the
	// persistent shell directly and give it a real new console instead.
	cmd := exec.Command("cmd.exe", "/D", "/K", sshLine)
	configureChildProcess(cmd, true)
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
