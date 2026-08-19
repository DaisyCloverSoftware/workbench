package core

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// discoverOpenClawOperationsCloudCatalog mirrors the privacy-minimal dynamic
// OpenClaw model catalogue used elsewhere, but launches the CLI with the same
// repaired user-executable environment as real machine operations. This matters
// for systemd user services where OpenClaw is an npm/NVM script whose sibling
// Node runtime is not present on the service's original PATH.
func discoverOpenClawOperationsCloudCatalog(ctx context.Context, command string) (OpenClawCloudCatalog, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return OpenClawCloudCatalog{}, errors.New("OpenClaw command is empty")
	}

	defaultModel := ""
	if status, err := runOpenClawOperationsModelJSON(ctx, command, 10*time.Second, "models", "status", "--json"); err == nil {
		defaultModel = parseOpenClawDefaultModel(status)
	}
	body, err := runOpenClawOperationsModelJSON(ctx, command, 20*time.Second, "models", "list", "--json")
	if err != nil {
		return OpenClawCloudCatalog{}, err
	}
	models, err := parseOpenClawModelList(body)
	if err != nil {
		return OpenClawCloudCatalog{}, err
	}

	seenDefault := false
	for i := range models {
		if defaultModel != "" && strings.EqualFold(models[i].Key, defaultModel) {
			models[i].Default = true
			seenDefault = true
		}
	}
	if defaultModel != "" && !seenDefault && openClawCloudModelKeyAllowed(defaultModel) {
		provider, name := splitOpenClawModelKey(defaultModel)
		models = append(models, OpenClawCloudModel{Key: defaultModel, Provider: provider, Name: name, Available: true, Default: true})
	}
	if len(models) == 0 {
		return OpenClawCloudCatalog{}, errors.New("OpenClaw exposed no available OpenAI or Anthropic cloud models")
	}
	annotateOpenClawModelHealth(models, time.Now().UTC())
	sort.SliceStable(models, func(i, j int) bool {
		return strings.ToLower(models[i].Key) < strings.ToLower(models[j].Key)
	})
	return OpenClawCloudCatalog{DefaultModel: defaultModel, Models: models}, nil
}

func runOpenClawOperationsModelJSON(ctx context.Context, command string, timeout time.Duration, args ...string) ([]byte, error) {
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(callCtx, command, args...)
	cmd.Env = environmentForUserExecutable(command)
	configureChildProcess(cmd, false)
	stdout := newBoundedWorkerCapture(maxOpenClawDiscoveryBytes)
	stderr := newBoundedWorkerCapture(256 << 10)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		if callCtx.Err() != nil {
			return nil, callCtx.Err()
		}
		return nil, fmt.Errorf("OpenClaw operations model discovery command failed: %w", err)
	}
	if stdout.Truncated() {
		return nil, errors.New("OpenClaw operations model discovery response exceeded Workbench's bounded limit")
	}
	body := []byte(stdout.String())
	if len(strings.TrimSpace(string(body))) == 0 {
		return nil, errors.New("OpenClaw operations model discovery returned no JSON")
	}
	return body, nil
}
