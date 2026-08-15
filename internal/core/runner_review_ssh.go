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

func RunRunnerReviewSSH(ctx context.Context, host string, req RunnerReviewRequest) (RunnerReviewResponse, error) {
	host, err := validateSSHHostTarget(host)
	if err != nil {
		return RunnerReviewResponse{}, err
	}
	body, err := json.Marshal(req)
	if err != nil {
		return RunnerReviewResponse{}, err
	}
	cmd := exec.CommandContext(ctx,
		"ssh",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=10",
		"-o", "StrictHostKeyChecking=accept-new",
		host,
		"$HOME/.local/bin/workbench-runner", "review-json",
	)
	configureChildProcess(cmd, false)
	cmd.Stdin = bytes.NewReader(body)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()

	var response RunnerReviewResponse
	if decodeErr := json.Unmarshal(stdout.Bytes(), &response); decodeErr == nil {
		if !response.OK {
			message := strings.TrimSpace(response.Error)
			if message == "" {
				message = "runner rejected review delivery request"
			}
			return response, errors.New(message)
		}
		if runErr != nil {
			return response, fmt.Errorf("runner review SSH failed: %w", runErr)
		}
		return response, nil
	}
	if runErr != nil {
		return RunnerReviewResponse{}, errors.New("runner review delivery is unreachable")
	}
	return RunnerReviewResponse{}, errors.New("runner review SSH returned invalid JSON")
}
