package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

func validateSSHHostTarget(raw string) (string, error) {
	host := strings.TrimSpace(raw)
	if host == "" {
		return "", errors.New("SSH host is not configured")
	}
	if strings.HasPrefix(host, "-") || strings.ContainsAny(host, " \t\r\n\x00") {
		return "", errors.New("SSH host contains unsafe option or whitespace characters")
	}
	return host, nil
}

// RunRunnerPublicationPolicySSH applies private runner publication policy over
// a separate operator SSH channel. The target is sent as JSON on stdin rather
// than being embedded in task transport or remote shell arguments.
func RunRunnerPublicationPolicySSH(ctx context.Context, host string, req RunnerPolicyRequest) (RunnerPolicyResponse, error) {
	host, err := validateSSHHostTarget(host)
	if err != nil {
		return RunnerPolicyResponse{}, err
	}
	body, err := json.Marshal(req)
	if err != nil {
		return RunnerPolicyResponse{}, err
	}
	cmd := exec.CommandContext(ctx,
		"ssh",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=10",
		"-o", "StrictHostKeyChecking=accept-new",
		host,
		"$HOME/.local/bin/workbench-runner", "policy-json",
	)
	configureChildProcess(cmd, false)
	cmd.Stdin = bytes.NewReader(body)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()

	var response RunnerPolicyResponse
	if decodeErr := json.Unmarshal(stdout.Bytes(), &response); decodeErr == nil {
		if !response.OK || strings.TrimSpace(response.Error) != "" {
			message := strings.TrimSpace(response.Error)
			if message == "" {
				message = "runner rejected publication policy"
			}
			return response, errors.New(message)
		}
		if runErr != nil {
			return response, fmt.Errorf("runner policy SSH failed: %w", runErr)
		}
		return response, nil
	}

	message := strings.TrimSpace(stderr.String())
	if message == "" {
		message = strings.TrimSpace(stdout.String())
	}
	if runErr != nil {
		return RunnerPolicyResponse{}, fmt.Errorf("runner policy SSH failed: %s", message)
	}
	return RunnerPolicyResponse{}, errors.New("runner policy SSH returned invalid JSON")
}
