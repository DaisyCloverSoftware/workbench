package core

import (
	"context"
	"errors"
	"strings"
)

const ownerGatedOperationsSourceMessage = "owner-gated RUM operations scripts must execute the exact current main commit; branch-head overrides are blocked"

var ownerGatedOperationsMainHead = resolveOwnerGatedOperationsMainHead

func validateOwnerGatedOperationsSource(ctx context.Context, root, commit string) error {
	origin, err := operationsGitOutput(ctx, root, "remote", "get-url", "origin")
	if err != nil || !ownerGatedRUMOrigin(origin) {
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
