package core

import (
	"context"
	"fmt"
	"strings"
)

type PublicationPolicySyncResult struct {
	Local  PublicationPolicy    `json:"local"`
	Runner *RunnerPolicyResponse `json:"runner,omitempty"`
}

// SavePublicationPolicyForExecutionHosts stores the explicit operator policy on
// the local Workbench host and, when a runner host is configured, mirrors the
// same policy through the separate operator SSH channel. Publication authority
// never enters task state, worker prompts or RunnerRequest transport.
func SavePublicationPolicyForExecutionHosts(ctx context.Context, project string, mode PublicationMode, remoteURL, runnerHost string) (PublicationPolicySyncResult, error) {
	policy := PublicationPolicy{Project: project, Mode: mode}
	if mode == PublicationPublish {
		policy.RemoteURL = strings.TrimSpace(remoteURL)
	}
	local, err := SavePublicationPolicy(policy)
	if err != nil {
		return PublicationPolicySyncResult{}, err
	}
	result := PublicationPolicySyncResult{Local: local}

	runnerHost = strings.TrimSpace(runnerHost)
	if runnerHost == "" {
		return result, nil
	}
	req := RunnerPolicyRequest{Action: string(mode), Project: project}
	if mode == PublicationPublish {
		req.RemoteURL = local.RemoteURL
	}
	runner, err := RunRunnerPublicationPolicySSH(ctx, runnerHost, req)
	if err != nil {
		return result, fmt.Errorf("local publication policy saved but runner sync failed: %w", err)
	}
	result.Runner = &runner
	return result, nil
}
