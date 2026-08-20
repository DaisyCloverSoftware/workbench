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
	case strings.HasPrefix(intent, core.RelayOperationsIntentPrefix):
		body := strings.TrimSpace(strings.TrimPrefix(intent, core.RelayOperationsIntentPrefix))
		if body == "" {
			return "", "", errors.New("operations relay intent is empty")
		}
		return "[relay:" + relayID + "] " + core.RelayOperationsIntentPrefix + " " + body, "operations", nil

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
		return "", "", errors.New("relay/inbox requires an explicit [workbench:operations] or [workbench:continuation] handoff")
	}
}

func sealRelayContinuation(authFile, relayID, project, intent string) (string, error) {
	auth, err := os.ReadFile(authFile)
	if err != nil {
		return "", fmt.Errorf("read local MCP auth for private relay continuation: %w", err)
	}
	return core.SealPrivateRelayContinuationIntent(strings.TrimSpace(string(auth)), relayID, project, intent)
}
