package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

const (
	openClawOperationFallbackModelEnv = "WORKBENCH_OPENCLAW_OPERATION_FALLBACK_MODEL"
	openClawFallbackProbeTimeout      = 10 * time.Second
)

// runOpenClawOperationInvocationWithFallback keeps the configured OpenClaw
// model as first choice, then performs model-level failover inside the same
// Workbench job conversation. A provider usage ceiling therefore does not make
// the user babysit OpenClaw: Workbench first tries an explicitly configured
// model, otherwise a live model on the other cloud provider (Claude when the
// exhausted route is OpenAI/Codex), and finally a suitable local Ollama model.
func runOpenClawOperationInvocationWithFallback(ctx context.Context, p Provider, task Task, prefs Preferences, prompt, sessionID string) (RunResult, bool, error) {
	res, complete, err := runOpenClawOperationInvocation(ctx, p, task, prefs, prompt, sessionID)
	if err == nil || !operationModelCapacityFailure(res, err) || ctx.Err() != nil {
		return res, complete, err
	}

	tried := map[string]bool{}
	tryModel := func(model string) (RunResult, bool, error, bool) {
		model = strings.TrimSpace(model)
		if model == "" || tried[strings.ToLower(model)] {
			return RunResult{}, false, nil, false
		}
		tried[strings.ToLower(model)] = true
		fallbackRes, fallbackComplete, fallbackErr := runOpenClawOperationModelOverride(ctx, p, task, prefs, prompt, sessionID, model)
		return fallbackRes, fallbackComplete, fallbackErr, true
	}

	if configured := strings.TrimSpace(os.Getenv(openClawOperationFallbackModelEnv)); configured != "" {
		fallbackRes, fallbackComplete, fallbackErr, attempted := tryModel(configured)
		if attempted {
			if fallbackErr == nil {
				return fallbackRes, fallbackComplete, nil
			}
			if operationFallbackMustStop(fallbackRes) {
				return boundRunResultForPersistence(fallbackRes), false, fallbackErr
			}
		}
	}

	if cloudModel := detectOpenClawOperationsCloudFallback(ctx, p, prefs, res, err); cloudModel != "" {
		fallbackRes, fallbackComplete, fallbackErr, attempted := tryModel(cloudModel)
		if attempted {
			if fallbackErr == nil {
				return fallbackRes, fallbackComplete, nil
			}
			if operationFallbackMustStop(fallbackRes) {
				return boundRunResultForPersistence(fallbackRes), false, fallbackErr
			}
		}
	}

	if localModel := detectOpenClawOperationsLocalModel(ctx, prefs); localModel != "" {
		fallbackRes, fallbackComplete, fallbackErr, attempted := tryModel(localModel)
		if attempted {
			if fallbackErr == nil {
				return fallbackRes, fallbackComplete, nil
			}
			fallbackRes.Retryable = true
			if strings.TrimSpace(fallbackRes.Output) == "" {
				fallbackRes.Output = "OpenClaw's primary cloud model was unavailable and its cloud/local operations fallbacks could not complete the task."
			}
			return boundRunResultForPersistence(fallbackRes), false, fmt.Errorf("OpenClaw model capacity was exhausted and fallback model %s also failed: %w", localModel, fallbackErr)
		}
	}

	return res, complete, err
}

func operationFallbackMustStop(res RunResult) bool {
	return res.Authentication || strings.TrimSpace(res.Attention) != "" || strings.TrimSpace(res.WorkerUnavailable) != ""
}

func operationModelCapacityFailure(res RunResult, err error) bool {
	if err == nil || res.Authentication || strings.TrimSpace(res.Attention) != "" || strings.TrimSpace(res.WorkerUnavailable) != "" {
		return false
	}
	low := strings.ToLower(err.Error() + " " + res.Output)
	for _, marker := range []string{
		"usage limit",
		"quota",
		"rate limit",
		"rate_limit",
		"billing",
		"credit limit",
		"credits exhausted",
		"capacity exhausted",
		"too many requests",
		"subscription usage",
	} {
		if strings.Contains(low, marker) {
			return true
		}
	}
	return false
}

func operationFailedCloudProvider(res RunResult, err error, catalog OpenClawCloudCatalog) string {
	low := strings.ToLower(res.Output)
	if err != nil {
		low += " " + strings.ToLower(err.Error())
	}
	switch {
	case strings.Contains(low, "codex"), strings.Contains(low, "openai"), strings.Contains(low, "gpt-"):
		return "openai"
	case strings.Contains(low, "claude"), strings.Contains(low, "anthropic"):
		return "anthropic"
	}
	provider, _ := splitOpenClawModelKey(catalog.DefaultModel)
	return canonicalOpenClawProvider(provider)
}

func detectOpenClawOperationsCloudFallback(ctx context.Context, p Provider, prefs Preferences, primary RunResult, primaryErr error) string {
	// Runner-host operations execute this function locally on the runner. Direct
	// remote SSH operation mode does not yet expose a privacy-minimal remote model
	// catalogue, so leave that uncommon path to the local Ollama fallback.
	if strings.TrimSpace(prefs.OpenClawSSHHost) != "" || strings.TrimSpace(p.Command) == "" {
		return ""
	}
	probeCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	catalog, err := discoverOpenClawOperationsCloudCatalog(probeCtx, p.Command)
	if err != nil {
		return ""
	}
	failedProvider := operationFailedCloudProvider(primary, primaryErr, catalog)
	model, ok := preferredOpenClawOperationsCloudFallback(catalog, failedProvider)
	if !ok {
		return ""
	}
	return model.Key
}

func preferredOpenClawOperationsCloudFallback(catalog OpenClawCloudCatalog, failedProvider string) (OpenClawCloudModel, bool) {
	failedProvider = canonicalOpenClawProvider(failedProvider)
	candidates := make([]OpenClawCloudModel, 0, len(catalog.Models))
	for _, model := range catalog.Models {
		if !model.Available || model.Cooling {
			continue
		}
		if failedProvider != "" && canonicalOpenClawProvider(model.Provider) == failedProvider {
			continue
		}
		candidates = append(candidates, model)
	}
	if len(candidates) == 0 {
		return OpenClawCloudModel{}, false
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		a, b := operationCloudFallbackScore(candidates[i]), operationCloudFallbackScore(candidates[j])
		if a != b {
			return a < b
		}
		return strings.ToLower(candidates[i].Key) < strings.ToLower(candidates[j].Key)
	})
	return candidates[0], true
}

func operationCloudFallbackScore(model OpenClawCloudModel) int {
	provider := canonicalOpenClawProvider(model.Provider)
	low := strings.ToLower(model.Key + " " + model.Name)
	if provider == "anthropic" {
		switch {
		case strings.Contains(low, "sonnet"):
			return 0
		case strings.Contains(low, "opus"):
			return 5
		case strings.Contains(low, "haiku"):
			return 10
		default:
			return 15
		}
	}
	if provider == "openai" {
		switch {
		case strings.Contains(low, "terra"):
			return 20
		case strings.Contains(low, "gpt-5.5"):
			return 25
		case strings.Contains(low, "gpt-5.4") && !strings.Contains(low, "mini"):
			return 30
		case strings.Contains(low, "sol"):
			return 35
		case strings.Contains(low, "codex-spark"):
			return 40
		default:
			return 45
		}
	}
	return 100
}

// detectOpenClawOperationsLocalModel asks only the local Ollama inventory and
// never reads model files or credentials. It deliberately prefers compact
// tool/coding-capable families that are plausible infrastructure operators.
func detectOpenClawOperationsLocalModel(ctx context.Context, prefs Preferences) string {
	probeCtx, cancel := context.WithTimeout(ctx, openClawFallbackProbeTimeout)
	defer cancel()

	remote := strings.TrimSpace(prefs.OpenClawSSHHost) != ""
	name := "ollama"
	args := []string{"list"}
	if remote {
		name = "ssh"
		args = []string{
			"-o", "BatchMode=yes",
			"-o", "ConnectTimeout=10",
			"-o", "StrictHostKeyChecking=accept-new",
			strings.TrimSpace(prefs.OpenClawSSHHost),
			"ollama", "list",
		}
	}

	cmd := exec.CommandContext(probeCtx, name, args...)
	if !remote {
		cmd.Env = environmentForUserExecutable(name)
	}
	configureChildProcess(cmd, false)
	stdout := newBoundedWorkerCapture(64 << 10)
	stderr := newBoundedWorkerCapture(16 << 10)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil || stdout.Truncated() || stderr.Truncated() {
		return ""
	}
	model := preferredOllamaOperationsModel(stdout.String())
	if model == "" {
		return ""
	}
	return "ollama/" + model
}

func preferredOllamaOperationsModel(listOutput string) string {
	type candidate struct {
		name  string
		rank  int
		order int
	}
	best := candidate{rank: 1 << 30, order: 1 << 30}
	order := 0
	for _, line := range strings.Split(listOutput, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) == 0 {
			continue
		}
		name := strings.TrimSpace(fields[0])
		low := strings.ToLower(name)
		if low == "name" || strings.Contains(low, "embed") || strings.Contains(low, "nomic") || strings.Contains(low, "bge-") {
			continue
		}

		rank := 100
		switch {
		case strings.Contains(low, "qwen") && strings.Contains(low, "coder"):
			rank = 0
		case strings.Contains(low, "qwen"):
			rank = 10
		case strings.Contains(low, "coder") || strings.Contains(low, "code"):
			rank = 20
		case strings.Contains(low, "llama"):
			rank = 30
		case strings.Contains(low, "mistral") || strings.Contains(low, "gemma"):
			rank = 40
		default:
			continue
		}
		if rank < best.rank || (rank == best.rank && order < best.order) {
			best = candidate{name: name, rank: rank, order: order}
		}
		order++
	}
	return best.name
}

func runOpenClawOperationModelOverride(ctx context.Context, p Provider, task Task, prefs Preferences, prompt, sessionID, model string) (RunResult, bool, error) {
	invokeCtx, cancel := context.WithTimeout(ctx, operationInvocationTimeout)
	defer cancel()
	if sessionID != openClawOperationSessionID(task) {
		return RunResult{}, false, errors.New("OpenClaw operations session identity mismatch")
	}

	var name string
	var args []string
	remote := strings.TrimSpace(prefs.OpenClawSSHHost) != ""
	if remote {
		name = "ssh"
		args = []string{"-o", "BatchMode=yes", "-o", "ConnectTimeout=10", "-o", "StrictHostKeyChecking=accept-new", strings.TrimSpace(prefs.OpenClawSSHHost), "openclaw"}
		args = append(args, openClawOperationAgentArgsWithSession(prompt, model, sessionID)...)
	} else if strings.TrimSpace(p.Command) != "" {
		name = p.Command
		args = openClawOperationAgentArgsWithSession(prompt, model, sessionID)
	} else {
		return RunResult{Retryable: true}, false, errors.New("OpenClaw operations adapter is not configured")
	}

	cmd := exec.CommandContext(invokeCtx, name, args...)
	if !remote {
		cmd.Dir = task.ProjectPath
		cmd.Env = environmentForUserExecutable(name)
	}
	configureChildProcess(cmd, false)
	stdout := newBoundedWorkerCapture(maxWorkerStreamCaptureBytes)
	stderr := newBoundedWorkerCapture(maxWorkerStreamCaptureBytes)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	runErr := cmd.Run()

	out := strings.TrimSpace(stdout.String())
	if se := strings.TrimSpace(stderr.String()); se != "" {
		if out != "" {
			out += "\n\n"
		}
		out += se
	}
	out = normalizeProviderOutput("openclaw", out)
	clean, complete := stripOperationCompletionMarker(out)
	res := classifyRunOutput(clean)

	if stdout.Truncated() || stderr.Truncated() {
		res.Retryable = true
		return boundRunResultForPersistence(res), false, errors.New("OpenClaw fallback operations response exceeded Workbench's bounded output limit")
	}
	if strings.TrimSpace(res.WorkerUnavailable) != "" {
		res.Retryable = true
		return boundRunResultForPersistence(res), false, fmt.Errorf("OpenClaw fallback is unavailable for this operational task: %s", res.WorkerUnavailable)
	}
	if strings.TrimSpace(res.Attention) != "" && isWorkerSetupAttention(res.Attention) {
		q := res.Attention
		res.Attention = ""
		res.Retryable = true
		return boundRunResultForPersistence(res), false, fmt.Errorf("OpenClaw fallback cannot act autonomously under its current local tool permissions: %s", q)
	}
	if runErr != nil {
		if errors.Is(invokeCtx.Err(), context.DeadlineExceeded) && ctx.Err() == nil {
			res.Retryable = true
			if strings.TrimSpace(res.Output) == "" {
				res.Output = "OpenClaw fallback became unresponsive before confirming completion."
			}
			return boundRunResultForPersistence(res), false, errors.New("OpenClaw fallback operations invocation timed out")
		}
		low := strings.ToLower(out + " " + runErr.Error())
		res.Authentication = strings.Contains(low, "login") || strings.Contains(low, "sign in") || strings.Contains(low, "authenticate") || strings.Contains(low, "unauthorized") || strings.Contains(low, "credential") || strings.Contains(low, "publickey") || strings.Contains(low, "permission denied")
		res.Retryable = true
		if strings.TrimSpace(res.Output) == "" {
			res.Output = runErr.Error()
		}
		return boundRunResultForPersistence(res), false, fmt.Errorf("OpenClaw fallback operations invocation exited with error: %w", runErr)
	}
	return boundRunResultForPersistence(res), complete, nil
}
