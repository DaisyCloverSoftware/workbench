package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

func rankOpenClawCloudModelsWithOverride(catalog OpenClawCloudCatalog, intent, requested string) []OpenClawCloudModel {
	base := RankOpenClawCloudModels(catalog, intent)
	requested = strings.TrimSpace(requested)
	if requested == "" {
		return base
	}
	validated, err := normalizeOpenClawCloudModelRef(requested)
	if err != nil {
		return base
	}

	var selected *OpenClawCloudModel
	for i := range catalog.Models {
		model := catalog.Models[i]
		if strings.EqualFold(model.Key, validated) && model.Available && !model.Cooling {
			copy := model
			selected = &copy
			break
		}
	}
	// OAuth/model-policy refreshes can legitimately remove a previously saved
	// model. A stale override therefore falls back to the live dynamic ladder
	// instead of making an otherwise runnable Workbench task fail.
	if selected == nil {
		return base
	}

	out := make([]OpenClawCloudModel, 0, maxOpenClawCloudCandidates)
	out = append(out, *selected)
	for _, model := range base {
		if strings.EqualFold(model.Key, selected.Key) {
			continue
		}
		out = append(out, model)
		if len(out) == maxOpenClawCloudCandidates {
			break
		}
	}
	return out
}

// RunOpenClawCloudAgentCLIWithTaskOverride is the same bounded runner-host
// OpenClaw cloud stage as RunOpenClawCloudAgentCLI, with one additional control:
// an explicit per-task model may be supplied by Workbench through a process
// environment variable. The override is never read from the worker prompt and
// never changes the outer provider route. If the selected model disappeared or
// is cooling, the live automatic ladder remains authoritative.
func RunOpenClawCloudAgentCLIWithTaskOverride(ctx context.Context, agentArgs []string, stdout, stderr io.Writer) int {
	prompt, err := parseOpenClawAgentArgs(agentArgs)
	if err != nil {
		if stderr != nil {
			_, _ = fmt.Fprintln(stderr, err)
		}
		return 2
	}
	command, err := resolveOpenClawCommand("")
	if err != nil {
		if stderr != nil {
			_, _ = fmt.Fprintln(stderr, err)
		}
		return 1
	}
	watched, cancel := contextWatchingParent(ctx)
	defer cancel()

	catalog, discoveryErr := DiscoverOpenClawCloudModels(watched, command)
	if discoveryErr != nil {
		attempt := runOpenClawAgentAttempt(watched, command, agentArgs, "")
		writeOpenClawAttempt(attempt, stdout, stderr)
		return attempt.code
	}
	intent := workbenchIntentFromWorkerPrompt(prompt)
	candidates := rankOpenClawCloudModelsWithOverride(catalog, intent, os.Getenv("WORKBENCH_OPENCLAW_TASK_MODEL"))
	if len(candidates) == 0 {
		if stdout != nil {
			_, _ = fmt.Fprintln(stdout, "WORKER_UNAVAILABLE: all discovered OpenClaw cloud models are temporarily cooling down after recent model-level failures")
		}
		return 1
	}

	var last openClawAttempt
	for _, model := range candidates {
		if watched.Err() != nil {
			return 1
		}
		attempt := runOpenClawAgentAttempt(watched, command, agentArgs, model.Key)
		last = attempt
		healthID := openClawModelHealthID(model.Key)
		if attempt.err == nil && strings.TrimSpace(attempt.result.WorkerUnavailable) == "" {
			_ = ClearProviderHealth(healthID)
			writeOpenClawAttempt(attempt, stdout, stderr)
			return 0
		}
		if strings.TrimSpace(attempt.result.Attention) != "" {
			_ = ClearProviderHealth(healthID)
			writeOpenClawAttempt(attempt, stdout, stderr)
			return 0
		}
		if attempt.result.Retryable {
			healthErr := attempt.err
			if healthErr == nil {
				healthErr = errors.New("OpenClaw model reported itself unavailable")
			}
			_, _ = RecordProviderRunOutcome(healthID, attempt.result, healthErr)
		}
	}
	writeOpenClawAttempt(last, stdout, stderr)
	if last.code == 0 {
		return 1
	}
	return last.code
}
