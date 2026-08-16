package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
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
	cmd := exec.CommandContext(ctx,
		"ssh",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=10",
		"-o", "StrictHostKeyChecking=accept-new",
		host,
		"$HOME/.local/bin/workbench-runner", "tool-json",
	)
	configureChildProcess(cmd, false)
	cmd.Stdin = bytes.NewReader(body)
	stdout := &limitedCapture{limit: 4 << 20}
	stderr := &limitedCapture{limit: 64 << 10}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	runErr := cmd.Run()
	if stdout.exceeded || stderr.exceeded {
		return RunnerToolResponse{}, errors.New("runner tool response exceeded Workbench limits")
	}

	var response RunnerToolResponse
	if decodeErr := json.Unmarshal([]byte(stdout.String()), &response); decodeErr != nil {
		return RunnerToolResponse{}, errors.New("runner tool transport returned invalid JSON")
	}
	if !response.OK || strings.TrimSpace(response.Error) != "" {
		message := strings.TrimSpace(response.Error)
		if message == "" {
			message = "runner tool operation failed"
		}
		return response, errors.New(message)
	}
	if runErr != nil {
		return response, errors.New("runner tool transport is unavailable")
	}
	return response, nil
}
