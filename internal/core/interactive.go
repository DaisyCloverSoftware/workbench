package core

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func StartProviderLogin(providerID string) error {
	name, args, ok := LoginCommand(providerID)
	if !ok {
		return fmt.Errorf("%s does not expose a supported subscription login flow", providerID)
	}
	cmd := exec.Command(name, args...)
	configureChildProcess(cmd, true)
	if !isWindows() {
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	}
	return cmd.Start()
}

func TestOpenClawSSH(host string) (string, error) {
	validatedHost, err := validateSSHHostTarget(host)
	if err != nil {
		return "", fmt.Errorf("OpenClaw SSH host is invalid: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=6", "-o", "StrictHostKeyChecking=accept-new", validatedHost, "openclaw", "--version")
	configureChildProcess(cmd, false)
	var buf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &buf
	err = cmd.Run()
	out := strings.TrimSpace(buf.String())
	if err != nil {
		if ctx.Err() != nil {
			return out, fmt.Errorf("OpenClaw SSH test timed out")
		}
		return out, fmt.Errorf("OpenClaw SSH test failed: %w", err)
	}
	return out, nil
}
