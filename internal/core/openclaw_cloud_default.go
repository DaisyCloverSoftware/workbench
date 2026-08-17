package core

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const RunnerCloudModelProviderIDPrefix = "openclaw-model:"

// normalizeOpenClawCloudModelRef accepts only bounded canonical provider/model
// references from the OpenAI/Anthropic cloud families that Workbench exposes in
// its OpenClaw cloud stage. The value is still passed as one argv element (never
// through a shell); validation additionally keeps control/state identifiers
// predictable and prevents an arbitrary provider from being smuggled through a
// synthetic runner-provider row.
func normalizeOpenClawCloudModelRef(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 240 {
		return "", errors.New("OpenClaw cloud model reference is empty or too long")
	}
	provider, model := splitOpenClawModelKey(value)
	if provider == "" || model == "" || !openClawCloudModelKeyAllowed(value) {
		return "", errors.New("OpenClaw cloud model must be an available OpenAI or Anthropic provider/model reference")
	}
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			continue
		}
		switch r {
		case '/', '.', '_', '-', ':', '@':
			continue
		default:
			return "", errors.New("OpenClaw cloud model reference contains unsupported characters")
		}
	}
	return value, nil
}

func RunnerCloudModelProviderID(model string) (string, error) {
	model, err := normalizeOpenClawCloudModelRef(model)
	if err != nil {
		return "", err
	}
	return RunnerCloudModelProviderIDPrefix + model, nil
}

func RunnerCloudModelRefFromProviderID(providerID string) (string, bool) {
	providerID = strings.TrimSpace(providerID)
	if !strings.HasPrefix(providerID, RunnerCloudModelProviderIDPrefix) {
		return "", false
	}
	model, err := normalizeOpenClawCloudModelRef(strings.TrimPrefix(providerID, RunnerCloudModelProviderIDPrefix))
	return model, err == nil
}

// SetOpenClawCloudDefault changes OpenClaw's own global default model. Workbench
// deliberately does not invent a second competing global-default store: the
// cloud-stage router already follows OpenClaw's resolved default for routine
// work. The requested ref must first be present in OpenClaw's effective allowed
// catalogue, and the post-write resolved default is verified before success.
func SetOpenClawCloudDefault(ctx context.Context, command, requested string) (OpenClawCloudCatalog, error) {
	requested, err := normalizeOpenClawCloudModelRef(requested)
	if err != nil {
		return OpenClawCloudCatalog{}, err
	}
	resolved, err := resolveOpenClawCommand(command)
	if err != nil {
		return OpenClawCloudCatalog{}, err
	}

	before, err := DiscoverOpenClawCloudModels(ctx, resolved)
	if err != nil {
		return OpenClawCloudCatalog{}, err
	}
	allowed := false
	for _, model := range before.Models {
		if model.Available && strings.EqualFold(model.Key, requested) {
			requested = model.Key
			allowed = true
			break
		}
	}
	if !allowed {
		return OpenClawCloudCatalog{}, errors.New("requested OpenClaw cloud model is not in the current allowed/available catalogue")
	}

	setCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(setCtx, resolved, "models", "set", requested)
	configureChildProcess(cmd, false)
	stdout := newBoundedWorkerCapture(256 << 10)
	stderr := newBoundedWorkerCapture(256 << 10)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		if setCtx.Err() != nil {
			return OpenClawCloudCatalog{}, setCtx.Err()
		}
		return OpenClawCloudCatalog{}, fmt.Errorf("OpenClaw could not set the selected default model: %w", err)
	}

	after, err := DiscoverOpenClawCloudModels(ctx, resolved)
	if err != nil {
		return OpenClawCloudCatalog{}, fmt.Errorf("OpenClaw changed its default but Workbench could not verify the new model: %w", err)
	}
	if !strings.EqualFold(after.DefaultModel, requested) {
		return after, fmt.Errorf("OpenClaw reported default %q after Workbench requested %q", after.DefaultModel, requested)
	}
	return after, nil
}
