package core

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

type RunnerUpdateResult struct {
	OK              bool   `json:"ok"`
	CurrentVersion  string `json:"current_version"`
	LatestVersion   string `json:"latest_version"`
	UpdateAvailable bool   `json:"update_available"`
	Applied         bool   `json:"applied"`
	Message         string `json:"message,omitempty"`
}

func UpdateWorkbenchRunnerSSH(ctx context.Context, host string) (RunnerUpdateResult, error) {
	validated, err := validateSSHHostTarget(host)
	if err != nil {
		return RunnerUpdateResult{}, err
	}
	stdout, stderr, truncated, runErr := runRunnerSSHCommand(ctx, validated, nil, "update", "apply")
	if truncated {
		return RunnerUpdateResult{}, errors.New("Workbench Runner update response exceeded the bounded SSH transport limit")
	}
	var result RunnerUpdateResult
	if err := json.Unmarshal(stdout, &result); err == nil && result.OK {
		if runErr != nil {
			return result, errors.New("Workbench Runner update command failed after returning a success payload")
		}
		return result, nil
	}
	combined := strings.TrimSpace(combineRunnerSSHOutput(stdout, stderr))
	if combined == "" {
		combined = "Workbench Runner update did not complete"
	}
	return RunnerUpdateResult{}, errors.New(boundWorkerControlText(combined))
}
