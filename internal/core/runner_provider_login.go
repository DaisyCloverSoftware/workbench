package core

import (
	"errors"
	"os"
	"os/exec"
	"strings"
)

const RunnerConnectionProviderID = "workbench-runner-connection"

// RunProviderLoginInteractive is an operator-only helper used by the cluster
// runner. The provider ID must resolve through Workbench's fixed LoginCommand
// allowlist; no executable, argument or shell text is accepted from the caller.
func RunProviderLoginInteractive(providerID string) error {
	providerID = strings.TrimSpace(providerID)
	name, args, ok := LoginCommand(providerID)
	if !ok {
		return errors.New("provider does not expose a supported login flow")
	}
	cmd := exec.Command(name, args...)
	configureChildProcess(cmd, true)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

// RunRunnerSSHVerificationInteractive launches one human-visible SSH session
// and invokes only Workbench Runner's fixed version operation. Normal Workbench
// jobs still require the non-interactive SSH path to succeed afterwards; this
// helper never turns password-only SSH into a false unattended-ready state.
func RunRunnerSSHVerificationInteractive(host string) error {
	host, err := validateSSHHostTarget(host)
	if err != nil {
		return err
	}
	return runRunnerSSHConsole(host, []string{"$HOME/.local/bin/workbench-runner", "version"})
}

// StartRunnerProviderLogin opens a human-visible SSH session to one configured
// runner and invokes only the runner's allowlisted provider-login operation.
// It never accepts arbitrary remote command text and is not exposed through MCP.
func StartRunnerProviderLogin(host, providerID string) error {
	host, err := validateSSHHostTarget(host)
	if err != nil {
		return err
	}
	providerID = strings.TrimSpace(providerID)
	if providerID == RunnerConnectionProviderID {
		return startRunnerSSHConsole(host, []string{"$HOME/.local/bin/workbench-runner", "version"})
	}
	if _, _, ok := LoginCommand(providerID); !ok {
		return errors.New("provider does not expose a supported login flow")
	}
	return startRunnerSSHConsole(host, []string{"$HOME/.local/bin/workbench-runner", "provider-login", providerID})
}
