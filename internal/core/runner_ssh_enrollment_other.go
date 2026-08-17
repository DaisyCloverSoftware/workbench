//go:build !windows

package core

func startRunnerSSHEnrollment(host string) error {
	return startRunnerSSHConsole(host, []string{"$HOME/.local/bin/workbench-runner", "version"})
}
