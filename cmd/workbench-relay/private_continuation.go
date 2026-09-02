package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func prepareRelayTaskIntent(authFile, relayID, project, intent string) (string, string, error) {
	intent = strings.TrimSpace(intent)
	switch {
	case strings.HasPrefix(intent, core.OpenClawExplicitAuthorizationPrefix):
		body := strings.TrimSpace(strings.TrimPrefix(intent, core.OpenClawExplicitAuthorizationPrefix))
		if !strings.HasPrefix(body, core.RelayOperationsIntentPrefix) {
			return "", "", errors.New("explicit OpenClaw authorization must be followed by [workbench:operations]")
		}
		body = strings.TrimSpace(strings.TrimPrefix(body, core.RelayOperationsIntentPrefix))
		if body == "" {
			return "", "", errors.New("OpenClaw operations relay intent is empty")
		}
		return core.OpenClawExplicitAuthorizationPrefix + " [relay:" + relayID + "] " + core.RelayOperationsIntentPrefix + " " + body, "openclaw-operations", nil

	case strings.HasPrefix(intent, core.RelayOperationsIntentPrefix):
		return "", "", errors.New("[workbench:operations] is routing metadata, not owner authorization; relay/inbox OpenClaw operations require explicit owner authorization naming OpenClaw")

	case strings.HasPrefix(intent, core.RelayContinuationIntentPrefix):
		body := strings.TrimSpace(strings.TrimPrefix(intent, core.RelayContinuationIntentPrefix))
		if body == "" {
			return "", "", errors.New("continuation relay intent is empty")
		}
		sealed, err := sealRelayContinuation(authFile, relayID, project, body)
		if err != nil {
			return "", "", err
		}
		return sealed, "continuation", nil

	default:
		return "", "", errors.New("relay/inbox accepts an authenticated development continuation or an explicitly owner-authorized OpenClaw operation")
	}
}

func sealRelayContinuation(authFile, relayID, project, intent string) (string, error) {
	auth, err := os.ReadFile(authFile)
	if err != nil {
		return "", fmt.Errorf("read local MCP auth for private relay continuation: %w", err)
	}
	return core.SealPrivateRelayContinuationIntent(strings.TrimSpace(string(auth)), relayID, project, intent)
}
