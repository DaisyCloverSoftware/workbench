package core

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

// RunHostBridgeRPCSSH sends one typed host-bridge request through the same
// fixed Workbench Runner SSH transport used by the desktop's existing runner
// operations. The caller cannot select a remote executable or remote command.
func RunHostBridgeRPCSSH(ctx context.Context, host string, req HostBridgeRPCRequest) (HostBridgeRPCResponse, error) {
	validated, err := validateSSHHostTarget(host)
	if err != nil {
		return HostBridgeRPCResponse{}, err
	}
	body, err := json.Marshal(req)
	if err != nil {
		return HostBridgeRPCResponse{}, err
	}
	stdout, stderr, truncated, runErr := runRunnerSSHCommand(ctx, validated, body, "host-json")
	if truncated {
		return HostBridgeRPCResponse{}, errors.New("host bridge response exceeded Workbench limits")
	}

	var response HostBridgeRPCResponse
	decodeErr := json.Unmarshal(stdout, &response)
	if decodeErr == nil {
		if !response.OK || strings.TrimSpace(response.Error) != "" {
			message := strings.TrimSpace(response.Error)
			if message == "" {
				message = "host bridge operation failed"
			}
			return response, errors.New(message)
		}
		if runErr != nil {
			return response, classifyRunnerToolSSHFailure(combineRunnerSSHOutput(stdout, stderr), runErr, ctx.Err())
		}
		return response, nil
	}
	if runErr != nil {
		return HostBridgeRPCResponse{}, classifyRunnerToolSSHFailure(combineRunnerSSHOutput(stdout, stderr), runErr, ctx.Err())
	}
	return HostBridgeRPCResponse{}, errors.New("host bridge transport returned invalid JSON")
}
