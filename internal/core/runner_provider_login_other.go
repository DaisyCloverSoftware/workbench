//go:build !windows

package core

import (
	"os"
	"os/exec"
)

func runRunnerSSHConsole(host string, remoteArgs []string) error {
	cmd := runnerSSHConsoleCommand(host, remoteArgs)
	return cmd.Run()
}

func startRunnerSSHConsole(host string, remoteArgs []string) error {
	cmd := runnerSSHConsoleCommand(host, remoteArgs)
	return cmd.Start()
}

func runnerSSHConsoleCommand(host string, remoteArgs []string) *exec.Cmd {
	args := []string{
		"-t",
		"-o", "ConnectTimeout=10",
		"-o", "StrictHostKeyChecking=accept-new",
		host,
	}
	args = append(args, remoteArgs...)
	cmd := exec.Command("ssh", args...)
	configureChildProcess(cmd, true)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd
}
