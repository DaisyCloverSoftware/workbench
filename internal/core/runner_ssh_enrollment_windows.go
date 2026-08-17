//go:build windows

package core

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// startRunnerSSHEnrollment turns the unavoidable first trust step into one
// operator action: Workbench prepares a local key, copies a complete OpenClaw
// enrollment prompt containing only the public key, and shows a persistent
// console telling the operator what happened. The private key never leaves the
// Windows profile.
func startRunnerSSHEnrollment(host string) error {
	prompt, err := PrepareRunnerSSHEnrollment(host)
	if err != nil {
		return err
	}
	clip := exec.Command("clip.exe")
	configureChildProcess(clip, false)
	clip.Stdin = strings.NewReader(prompt)
	if out, err := clip.CombinedOutput(); err != nil {
		message := strings.TrimSpace(string(out))
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("copy runner SSH enrollment prompt: %s", message)
	}

	spec := runnerSSHConsoleSpec{
		File: "cmd.exe",
		Parameters: "/D /K echo Workbench prepared an unattended SSH key and copied ONE runner enrolment prompt to your clipboard. " +
			"Send that clipboard text to OpenClaw once. When OpenClaw reports completion, close this window and click Rescan in Workbench.",
	}
	go func() {
		time.Sleep(450 * time.Millisecond)
		_ = shellExecuteRunnerSSHConsole(spec)
	}()
	return nil
}
