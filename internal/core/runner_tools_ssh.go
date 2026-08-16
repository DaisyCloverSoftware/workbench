package core

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
)

var (
	ErrRunnerSSHAuthentication       = errors.New("runner unattended SSH authentication is unavailable")
	ErrRunnerSSHClientUnavailable    = errors.New("Windows OpenSSH client is unavailable")
	ErrRunnerSSHConnectionTimeout    = errors.New("runner SSH connection timed out")
	ErrRunnerExecutableUnavailable   = errors.New("Workbench Runner executable is unavailable on the configured host")
	ErrRunnerSSHTransportUnavailable = errors.New("runner SSH transport is unavailable")
)

func RunRunnerToolSSH(ctx context.Context, host string, req RunnerToolRequest) (RunnerToolResponse, error) {
	host, err := validateSSHHostTarget(host)
	if err != nil {
		return RunnerToolResponse{}, err
	}
	body, err := json.Marshal(req)
	if err != nil {
		return RunnerToolResponse{}, err
	}
	stdout, stderr, truncated, runErr := runRunnerSSHCommand(ctx, host, body, "tool-json")
	if truncated {
		return RunnerToolResponse{}, errors.New("runner tool response exceeded Workbench limits")
	}

	var response RunnerToolResponse
	decodeErr := json.Unmarshal(stdout, &response)
	if decodeErr == nil {
		if !response.OK || strings.TrimSpace(response.Error) != "" {
			message := strings.TrimSpace(response.Error)
			if message == "" {
				message = "runner tool operation failed"
			}
			return response, errors.New(message)
		}
		if runErr != nil {
			return response, classifyRunnerToolSSHFailure(combineRunnerSSHOutput(stdout, stderr), runErr, ctx.Err())
		}
		return response, nil
	}
	if runErr != nil {
		return RunnerToolResponse{}, classifyRunnerToolSSHFailure(combineRunnerSSHOutput(stdout, stderr), runErr, ctx.Err())
	}
	return RunnerToolResponse{}, errors.New("runner tool transport returned invalid JSON")
}

func classifyRunnerToolSSHFailure(output string, runErr, contextErr error) error {
	if contextErr != nil {
		return ErrRunnerSSHConnectionTimeout
	}
	low := strings.ToLower(strings.TrimSpace(output))
	if runErr != nil {
		low += " " + strings.ToLower(runErr.Error())
	}
	if runnerSSHAuthenticationFailure(output, runErr) {
		return ErrRunnerSSHAuthentication
	}
	if errors.Is(runErr, exec.ErrNotFound) || strings.Contains(low, "executable file not found") || strings.Contains(low, "cannot find the file") {
		return ErrRunnerSSHClientUnavailable
	}
	if strings.Contains(low, ".local/bin/workbench-runner") && (strings.Contains(low, "no such file") || strings.Contains(low, "not found")) {
		return ErrRunnerExecutableUnavailable
	}
	if strings.Contains(low, "connection timed out") || strings.Contains(low, "connect timeout") || strings.Contains(low, "operation timed out") {
		return ErrRunnerSSHConnectionTimeout
	}
	return ErrRunnerSSHTransportUnavailable
}
