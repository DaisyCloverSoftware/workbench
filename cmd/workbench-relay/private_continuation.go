package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func sealRelayContinuation(authFile, relayID, project, intent string) (string, error) {
	auth, err := os.ReadFile(authFile)
	if err != nil {
		return "", fmt.Errorf("read local MCP auth for private relay continuation: %w", err)
	}
	return core.SealPrivateRelayContinuationIntent(strings.TrimSpace(string(auth)), relayID, project, intent)
}
