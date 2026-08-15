package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// TestWorkbenchRunnerSSH verifies the configured execution host using the same
// fixed Workbench Runner transport as real durable jobs. It does not execute
// OpenClaw, an arbitrary remote command, or an operator-supplied command line.
func TestWorkbenchRunnerSSH(host string) (string, error) {
	validated, err := validateSSHHostTarget(host)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	stdout, stderr, truncated, runErr := runRunnerSSHCommand(ctx, validated, nil, "version")
	combined := combineRunnerSSHOutput(stdout, stderr)
	if truncated {
		return boundPersistedWorkerText(combined), errors.New("Workbench Runner probe response exceeded the bounded SSH transport limit")
	}
	if runErr != nil {
		if runnerSSHAuthenticationFailure(combined, runErr) {
			return combined, fmt.Errorf("Workbench Runner SSH authentication failed: %w", runErr)
		}
		return combined, fmt.Errorf("Workbench Runner probe failed: %w", runErr)
	}
	version := strings.TrimSpace(string(stdout))
	if version == "" {
		return combined, errors.New("Workbench Runner returned no version")
	}
	if strings.ContainsAny(version, "\r\n\t ") {
		return combined, errors.New("Workbench Runner returned an invalid version response")
	}
	return version, nil
}

// ValidateHarnessAdapterPath exposes the operator-side path check to native
// settings without exposing execution or model-facing authority.
func ValidateHarnessAdapterPath(path string) (string, error) {
	return validateHarnessAdapterPath(path)
}
