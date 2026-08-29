package core

import (
	"context"
	"errors"
	"strings"
)

const ownerGatedOperationsSourceMessage = "owner-gated RUM operations scripts must execute the exact current main commit; branch-head overrides are blocked"

const (
	rateAnythingPreviewCandidateCommit = "60cab55d5bd868913da833e60c15cbe938afa494"
	rateAnythingPreviewOperationPath   = "scripts/ops/deploy-rate-anything-preview.sh"
	rateAnythingPreviewAPIImage        = "ghcr.io/daisycloversoftware/rum-api@sha256:b3ee83307651abae8e9b93e63920336659f29932474788e7c0ac9022bc7bee44"
	rateAnythingPreviewFrontendImage   = "ghcr.io/daisycloversoftware/rum-rate-anything@sha256:7b3576d66dd288ec8309b6c62a67d81592ae896176420c0ec8f4cbae58d7e74e"
)

var ownerGatedOperationsMainHead = resolveOwnerGatedOperationsMainHead

type ownerGatedBranchOperationException struct {
	commit string
	path   string
	args   []string
}

// Branch-head RUM operations remain fail-closed by default. Exceptions must pin
// the entire reviewed operation tuple: immutable repository commit, exact
// operations-script path and exact argv. Never add a branch-name, namespace or
// image wildcard here. This candidate's wrapper hard-codes the dedicated
// isolated Rate Anything preview namespace and host, while its exact argv pins
// the already-CI-verified images that may be deployed there.
var ownerGatedRUMBranchOperationExceptions = []ownerGatedBranchOperationException{
	{
		commit: rateAnythingPreviewCandidateCommit,
		path:   rateAnythingPreviewOperationPath,
		args: []string{
			rateAnythingPreviewAPIImage,
			rateAnythingPreviewFrontendImage,
		},
	},
}

func validateOwnerGatedOperationsSource(ctx context.Context, root, commit, scriptPath string, args []string) error {
	origin, err := operationsGitOutput(ctx, root, "remote", "get-url", "origin")
	if err != nil || !ownerGatedRUMOrigin(origin) {
		return nil
	}

	if ownerGatedRUMBranchOperationAllowed(commit, scriptPath, args) {
		return nil
	}

	mainCommit, err := ownerGatedOperationsMainHead(ctx, root)
	if err != nil {
		return errors.New(ownerGatedOperationsSourceMessage)
	}
	if !strings.EqualFold(strings.TrimSpace(mainCommit), strings.TrimSpace(commit)) {
		return errors.New(ownerGatedOperationsSourceMessage)
	}
	return nil
}

func ownerGatedRUMBranchOperationAllowed(commit, scriptPath string, args []string) bool {
	commit = strings.ToLower(strings.TrimSpace(commit))
	scriptPath = strings.TrimSpace(scriptPath)
	for _, exception := range ownerGatedRUMBranchOperationExceptions {
		if commit != strings.ToLower(exception.commit) || scriptPath != exception.path || len(args) != len(exception.args) {
			continue
		}
		matched := true
		for i := range args {
			if args[i] != exception.args[i] {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

func resolveOwnerGatedOperationsMainHead(ctx context.Context, root string) (string, error) {
	out, err := operationsGitNetworkOutput(ctx, root, "ls-remote", "--heads", "origin", "refs/heads/main")
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) == 2 && fields[1] == "refs/heads/main" {
			return normalizeOperationsCommit(fields[0])
		}
	}
	return "", errors.New("origin main head was not found")
}

func ownerGatedRUMOrigin(raw string) bool {
	remote := strings.TrimSpace(raw)
	remote = strings.TrimSuffix(remote, ".git")
	for _, prefix := range []string{
		"https://github.com/",
		"ssh://git@github.com/",
		"git@github.com:",
	} {
		if strings.HasPrefix(remote, prefix) {
			remote = strings.TrimPrefix(remote, prefix)
			break
		}
	}
	return strings.EqualFold(strings.Trim(remote, "/"), "DaisyCloverSoftware/rum")
}
