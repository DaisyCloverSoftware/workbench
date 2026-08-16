package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

const (
	maxOpenClawDiscoveryBytes = 4 << 20
	maxOpenClawCloudCandidates = 5
	openClawModelHealthPrefix = "openclaw-model:"
)

// OpenClawCloudModel is deliberately account-agnostic model metadata suitable
// for Workbench routing and UI display. Auth profile ids, account identities,
// tokens, raw usage payloads and provider diagnostics never enter this shape.
type OpenClawCloudModel struct {
	Key            string `json:"key"`
	Provider       string `json:"provider"`
	Name           string `json:"name,omitempty"`
	Input          string `json:"input,omitempty"`
	ContextWindow  int64  `json:"context_window,omitempty"`
	ContextTokens  int64  `json:"context_tokens,omitempty"`
	Image          bool   `json:"image"`
	Available      bool   `json:"available"`
	Default        bool   `json:"default"`
	Cooling        bool   `json:"cooling,omitempty"`
	CooldownUntil  string `json:"cooldown_until,omitempty"`
	CooldownReason string `json:"cooldown_reason,omitempty"`
}

// OpenClawCloudCatalog is the bounded cloud-only view Workbench consumes after
// its existing local/cheap routing has already decided OpenClaw is eligible.
type OpenClawCloudCatalog struct {
	DefaultModel string               `json:"default_model,omitempty"`
	Models       []OpenClawCloudModel `json:"models"`
}

func resolveOpenClawCommand(command string) (string, error) {
	command = strings.TrimSpace(command)
	if command != "" {
		return command, nil
	}
	path, err := exec.LookPath("openclaw")
	if err != nil {
		return "", errors.New("OpenClaw CLI is not installed")
	}
	return path, nil
}

// DiscoverOpenClawCloudModels asks OpenClaw for its effective model view rather
// than hard-coding a catalogue. The normal list view intentionally respects the
// configured/allowed model policy, so Workbench does not select a model that an
// operator has excluded. Only safe model metadata is returned.
func DiscoverOpenClawCloudModels(ctx context.Context, command string) (OpenClawCloudCatalog, error) {
	resolved, err := resolveOpenClawCommand(command)
	if err != nil {
		return OpenClawCloudCatalog{}, err
	}

	defaultModel := ""
	if status, statusErr := runOpenClawJSON(ctx, resolved, 10*time.Second, "models", "status", "--json"); statusErr == nil {
		defaultModel = parseOpenClawDefaultModel(status)
	}

	body, err := runOpenClawJSON(ctx, resolved, 20*time.Second, "models", "list", "--json")
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
		models = append(models, OpenClawCloudModel{
			Key:       defaultModel,
			Provider:  provider,
			Name:      name,
			Available: true,
			Default:   true,
		})
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

func runOpenClawJSON(ctx context.Context, command string, timeout time.Duration, args ...string) ([]byte, error) {
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(callCtx, command, args...)
	configureChildProcess(cmd, false)
	stdout := newBoundedWorkerCapture(maxOpenClawDiscoveryBytes)
	stderr := newBoundedWorkerCapture(256 << 10)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		if callCtx.Err() != nil {
			return nil, callCtx.Err()
		}
		return nil, fmt.Errorf("OpenClaw model discovery command failed: %w", err)
	}
	if stdout.Truncated() {
		return nil, errors.New("OpenClaw model discovery response exceeded Workbench's bounded limit")
	}
	body := []byte(stdout.String())
	if len(strings.TrimSpace(string(body))) == 0 {
		return nil, errors.New("OpenClaw model discovery returned no JSON")
	}
	return body, nil
}

func parseOpenClawDefaultModel(body []byte) string {
	var root map[string]any
	dec := json.NewDecoder(strings.NewReader(string(body)))
	dec.UseNumber()
	if err := dec.Decode(&root); err != nil {
		return ""
	}
	for _, key := range []string{"resolvedDefault", "defaultModel"} {
		if value, ok := root[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func parseOpenClawModelList(body []byte) ([]OpenClawCloudModel, error) {
	var root any
	dec := json.NewDecoder(strings.NewReader(string(body)))
	dec.UseNumber()
	if err := dec.Decode(&root); err != nil {
		return nil, fmt.Errorf("decode OpenClaw model list: %w", err)
	}
	byKey := map[string]OpenClawCloudModel{}
	walkOpenClawModelRows(root, func(row map[string]any) {
		model, ok := openClawModelFromRow(row)
		if !ok || !model.Available || !openClawCloudModelKeyAllowed(model.Key) {
			return
		}
		key := strings.ToLower(model.Key)
		if old, exists := byKey[key]; !exists || model.ContextWindow > old.ContextWindow || model.ContextTokens > old.ContextTokens {
			byKey[key] = model
		}
	})
	models := make([]OpenClawCloudModel, 0, len(byKey))
	for _, model := range byKey {
		models = append(models, model)
	}
	return models, nil
}

func walkOpenClawModelRows(value any, visit func(map[string]any)) {
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			walkOpenClawModelRows(item, visit)
		}
	case map[string]any:
		if key, _ := typed["key"].(string); strings.Contains(strings.TrimSpace(key), "/") {
			visit(typed)
		}
		for _, child := range typed {
			walkOpenClawModelRows(child, visit)
		}
	}
}

func openClawModelFromRow(row map[string]any) (OpenClawCloudModel, bool) {
	key, _ := row["key"].(string)
	key = strings.TrimSpace(key)
	provider, modelID := splitOpenClawModelKey(key)
	if provider == "" || modelID == "" {
		return OpenClawCloudModel{}, false
	}
	available := true
	if value, ok := boolValue(row["available"]); ok {
		available = value
	}
	if local, ok := boolValue(row["local"]); ok && local {
		return OpenClawCloudModel{}, false
	}
	name, _ := row["name"].(string)
	input := openClawInputLabel(row["input"])
	model := OpenClawCloudModel{
		Key:           key,
		Provider:      provider,
		Name:          strings.TrimSpace(name),
		Input:         input,
		ContextWindow: numericInt64(row["contextWindow"]),
		ContextTokens: numericInt64(row["contextTokens"]),
		Image:         strings.Contains(strings.ToLower(input), "image"),
		Available:     available,
		Default:       openClawTagsContain(row["tags"], "default"),
	}
	if model.Name == "" {
		model.Name = modelID
	}
	return model, true
}

func boolValue(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "yes", "available":
			return true, true
		case "false", "no", "unavailable":
			return false, true
		}
	}
	return false, false
}

func numericInt64(value any) int64 {
	switch typed := value.(type) {
	case json.Number:
		if n, err := typed.Int64(); err == nil {
			return n
		}
	case float64:
		return int64(typed)
	case float32:
		return int64(typed)
	case int:
		return int64(typed)
	case int64:
		return typed
	}
	return 0
}

func openClawInputLabel(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
				parts = append(parts, strings.TrimSpace(text))
			}
		}
		return strings.Join(parts, "+")
	}
	return ""
}

func openClawTagsContain(value any, target string) bool {
	target = strings.ToLower(strings.TrimSpace(target))
	switch typed := value.(type) {
	case string:
		for _, part := range strings.FieldsFunc(typed, func(r rune) bool { return r == ',' || r == ' ' }) {
			if strings.ToLower(strings.TrimSpace(part)) == target {
				return true
			}
		}
	case []any:
		for _, item := range typed {
			if text, ok := item.(string); ok && strings.ToLower(strings.TrimSpace(text)) == target {
				return true
			}
		}
	}
	return false
}

func splitOpenClawModelKey(key string) (string, string) {
	key = strings.TrimSpace(key)
	parts := strings.SplitN(key, "/", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func canonicalOpenClawProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "openai", "openai-codex", "codex":
		return "openai"
	case "anthropic", "claude", "claude-cli", "anthropic-cli":
		return "anthropic"
	default:
		return strings.ToLower(strings.TrimSpace(provider))
	}
}

func openClawCloudModelKeyAllowed(key string) bool {
	provider, _ := splitOpenClawModelKey(key)
	provider = canonicalOpenClawProvider(provider)
	return provider == "openai" || provider == "anthropic"
}

func openClawModelHealthID(key string) string {
	return openClawModelHealthPrefix + strings.ToLower(strings.TrimSpace(key))
}

func annotateOpenClawModelHealth(models []OpenClawCloudModel, now time.Time) {
	state, err := loadProviderHealthState()
	if err != nil {
		return
	}
	active := activeProviderHealth(state, now)
	for i := range models {
		if record, ok := active[openClawModelHealthID(models[i].Key)]; ok {
			models[i].Cooling = true
			models[i].CooldownUntil = record.CooldownUntil.UTC().Format(time.RFC3339)
			models[i].CooldownReason = record.Reason
		}
	}
}

func workbenchIntentFromWorkerPrompt(prompt string) string {
	const startMarker = "\n\nIntent:\n"
	const endMarker = "\n\nRules:\n"
	start := strings.Index(prompt, startMarker)
	if start < 0 {
		return strings.TrimSpace(prompt)
	}
	start += len(startMarker)
	end := strings.Index(prompt[start:], endMarker)
	if end < 0 {
		return strings.TrimSpace(prompt[start:])
	}
	return strings.TrimSpace(prompt[start : start+end])
}

func openClawCloudEscalationNeeded(intent string) bool {
	low := strings.ToLower(strings.TrimSpace(intent))
	if low == "" {
		return false
	}
	score := 0
	weights := []struct {
		needle string
		weight int
	}{
		{"security", 3}, {"vulnerability", 3}, {"exploit", 3}, {"firewall", 3},
		{"data loss", 4}, {"corruption", 4}, {"incident", 3}, {"outage", 3},
		{"race condition", 3}, {"deadlock", 3}, {"authorization", 2}, {"encryption", 2},
		{"migration", 2}, {"recovery", 2}, {"rollback", 2}, {"architecture", 2},
		{"multi-system", 2}, {"distributed", 1}, {"authentication", 1}, {"oauth", 1},
		{"root cause", 1}, {"intermittent", 1}, {"complex", 1},
	}
	for _, signal := range weights {
		if strings.Contains(low, signal.needle) {
			score += signal.weight
		}
	}
	if len(intent) > 8000 {
		score++
	}
	return score >= 3
}

func openClawModelRole(model OpenClawCloudModel) string {
	low := strings.ToLower(model.Key + " " + model.Name)
	switch {
	case strings.Contains(low, "gpt-5.6-sol"), strings.Contains(low, "claude-opus"):
		return "flagship"
	case strings.Contains(low, "codex-spark"), strings.Contains(low, "gpt-5.4-mini"), strings.Contains(low, "gpt-5.6-luna"), strings.Contains(low, "claude-haiku"):
		return "economical"
	default:
		return "balanced"
	}
}

func openClawEconomicScore(model OpenClawCloudModel) int {
	low := strings.ToLower(model.Key + " " + model.Name)
	switch {
	case strings.Contains(low, "codex-spark"):
		return 0
	case strings.Contains(low, "gpt-5.6-luna"):
		return 5
	case strings.Contains(low, "gpt-5.4-mini"):
		return 10
	case strings.Contains(low, "claude-haiku"):
		return 12
	default:
		return 50
	}
}

func openClawBalancedScore(model OpenClawCloudModel) int {
	low := strings.ToLower(model.Key + " " + model.Name)
	switch {
	case strings.Contains(low, "gpt-5.6-terra"):
		return 0
	case strings.Contains(low, "claude-sonnet"):
		return 5
	case strings.Contains(low, "gpt-5.5"):
		return 10
	case strings.Contains(low, "gpt-5.4") && !strings.Contains(low, "mini"):
		return 15
	default:
		return 30
	}
}

func openClawFlagshipScore(model OpenClawCloudModel) int {
	low := strings.ToLower(model.Key + " " + model.Name)
	switch {
	case strings.Contains(low, "gpt-5.6-sol"):
		return 0
	case strings.Contains(low, "claude-opus"):
		return 5
	default:
		return 50
	}
}

func openClawEscalationScore(model OpenClawCloudModel) int {
	low := strings.ToLower(model.Key + " " + model.Name)
	switch {
	case strings.Contains(low, "gpt-5.6-sol"):
		return 0
	case strings.Contains(low, "claude-opus"):
		return 5
	case strings.Contains(low, "claude-sonnet"):
		return 12
	case strings.Contains(low, "gpt-5.6-terra"):
		return 15
	case strings.Contains(low, "gpt-5.5"):
		return 22
	case strings.Contains(low, "gpt-5.4") && !strings.Contains(low, "mini"):
		return 28
	case strings.Contains(low, "codex-spark"):
		return 45
	case strings.Contains(low, "gpt-5.6-luna"):
		return 60
	case strings.Contains(low, "gpt-5.4-mini"), strings.Contains(low, "claude-haiku"):
		return 70
	default:
		return 35
	}
}

// RankOpenClawCloudModels keeps the existing Workbench provider hierarchy
// intact. It only chooses among cloud models once the OpenClaw provider itself
// has been reached. Routine work follows the OpenClaw default first (the
// operator's current abundance preference), while genuinely difficult/high-risk
// work starts with flagship-capable models. Both paths retain cross-provider
// failover and a bounded candidate count.
func RankOpenClawCloudModels(catalog OpenClawCloudCatalog, intent string) []OpenClawCloudModel {
	ready := make([]OpenClawCloudModel, 0, len(catalog.Models))
	for _, model := range catalog.Models {
		if model.Available && !model.Cooling {
			ready = append(ready, model)
		}
	}
	if len(ready) == 0 {
		return nil
	}
	if openClawCloudEscalationNeeded(intent) {
		sort.SliceStable(ready, func(i, j int) bool {
			a, b := openClawEscalationScore(ready[i]), openClawEscalationScore(ready[j])
			if a != b {
				return a < b
			}
			return strings.ToLower(ready[i].Key) < strings.ToLower(ready[j].Key)
		})
		return boundedProviderDiverseModels(ready, maxOpenClawCloudCandidates)
	}
	return routineOpenClawModelLadder(catalog, ready)
}

func routineOpenClawModelLadder(catalog OpenClawCloudCatalog, ready []OpenClawCloudModel) []OpenClawCloudModel {
	var out []OpenClawCloudModel
	add := func(model OpenClawCloudModel) {
		for _, existing := range out {
			if strings.EqualFold(existing.Key, model.Key) {
				return
			}
		}
		if len(out) < maxOpenClawCloudCandidates {
			out = append(out, model)
		}
	}
	for _, model := range ready {
		if strings.EqualFold(model.Key, catalog.DefaultModel) || model.Default {
			add(model)
			break
		}
	}

	economical := filterAndSortOpenClawModels(ready, "economical", openClawEconomicScore)
	if len(economical) > 0 {
		add(economical[0])
	}
	balanced := filterAndSortOpenClawModels(ready, "balanced", openClawBalancedScore)
	if len(balanced) > 0 {
		add(balanced[0])
	}

	firstProvider := ""
	if len(out) > 0 {
		firstProvider = canonicalOpenClawProvider(out[0].Provider)
	}
	for _, model := range balanced {
		if firstProvider == "" || canonicalOpenClawProvider(model.Provider) != firstProvider {
			add(model)
			break
		}
	}
	flagships := filterAndSortOpenClawModels(ready, "flagship", openClawFlagshipScore)
	if len(flagships) > 0 {
		add(flagships[0])
	}

	if len(out) < maxOpenClawCloudCandidates {
		fallback := append([]OpenClawCloudModel(nil), ready...)
		sort.SliceStable(fallback, func(i, j int) bool {
			roleI, roleJ := openClawModelRole(fallback[i]), openClawModelRole(fallback[j])
			roleScore := func(role string) int {
				switch role {
				case "economical":
					return 0
				case "balanced":
					return 20
				default:
					return 80
				}
			}
			a, b := roleScore(roleI), roleScore(roleJ)
			if a != b {
				return a < b
			}
			return strings.ToLower(fallback[i].Key) < strings.ToLower(fallback[j].Key)
		})
		for _, model := range fallback {
			add(model)
		}
	}
	return out
}

func filterAndSortOpenClawModels(models []OpenClawCloudModel, role string, score func(OpenClawCloudModel) int) []OpenClawCloudModel {
	out := make([]OpenClawCloudModel, 0, len(models))
	for _, model := range models {
		if openClawModelRole(model) == role {
			out = append(out, model)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := score(out[i]), score(out[j])
		if a != b {
			return a < b
		}
		return strings.ToLower(out[i].Key) < strings.ToLower(out[j].Key)
	})
	return out
}

func boundedProviderDiverseModels(models []OpenClawCloudModel, limit int) []OpenClawCloudModel {
	if limit <= 0 || len(models) <= limit {
		return append([]OpenClawCloudModel(nil), models...)
	}
	out := append([]OpenClawCloudModel(nil), models[:limit]...)
	primaryProvider := canonicalOpenClawProvider(out[0].Provider)
	hasAlternate := false
	for _, model := range out[1:] {
		if canonicalOpenClawProvider(model.Provider) != primaryProvider {
			hasAlternate = true
			break
		}
	}
	if !hasAlternate {
		for _, model := range models[limit:] {
			if canonicalOpenClawProvider(model.Provider) != primaryProvider {
				out[len(out)-1] = model
				break
			}
		}
	}
	return out
}

func parseOpenClawAgentArgs(args []string) (string, error) {
	prompt := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--message", "-m":
			if i+1 >= len(args) || prompt != "" {
				return "", errors.New("OpenClaw Workbench wrapper requires exactly one message")
			}
			prompt = args[i+1]
			i++
		case "--headless":
			// Existing Workbench invocation flag. It is forwarded unchanged.
		default:
			return "", fmt.Errorf("OpenClaw Workbench wrapper rejected unsupported argument %q", args[i])
		}
	}
	if strings.TrimSpace(prompt) == "" {
		return "", errors.New("OpenClaw Workbench wrapper requires a non-empty message")
	}
	return prompt, nil
}

type openClawAttempt struct {
	stdout []byte
	stderr []byte
	err    error
	code   int
	result RunResult
}

func runOpenClawAgentAttempt(ctx context.Context, command string, agentArgs []string, model string) openClawAttempt {
	args := []string{"agent"}
	if strings.TrimSpace(model) != "" {
		args = append(args, "--model", model)
	}
	args = append(args, agentArgs...)
	cmd := exec.CommandContext(ctx, command, args...)
	configureChildProcess(cmd, false)
	stdout := newBoundedWorkerCapture(maxWorkerStreamCaptureBytes)
	stderr := newBoundedWorkerCapture(maxWorkerStreamCaptureBytes)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		code = 1
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() >= 0 {
			code = exitErr.ExitCode()
		}
	}
	combined := strings.TrimSpace(stdout.String())
	if se := strings.TrimSpace(stderr.String()); se != "" {
		if combined != "" {
			combined += "\n\n"
		}
		combined += se
	}
	result := classifyRunOutput(combined)
	if err != nil {
		low := strings.ToLower(combined + " " + err.Error())
		result.Authentication = containsAny(low, "login", "sign in", "authenticate", "authentication", "unauthorized", "credential", "permission denied")
		result.Retryable = result.Retryable || result.Authentication || containsAny(low,
			"rate limit", "rate-limit", "too many requests", "quota", "429", "billing",
			"unknown model", "model unavailable", "runtime unavailable", "temporarily unavailable",
			"timeout", "timed out", "deadline exceeded")
	}
	return openClawAttempt{stdout: []byte(stdout.String()), stderr: []byte(stderr.String()), err: err, code: code, result: result}
}

func writeOpenClawAttempt(attempt openClawAttempt, stdout, stderr io.Writer) {
	if stdout != nil && len(attempt.stdout) > 0 {
		_, _ = stdout.Write(attempt.stdout)
	}
	if stderr != nil && len(attempt.stderr) > 0 {
		_, _ = stderr.Write(attempt.stderr)
	}
}

func contextWatchingParent(ctx context.Context) (context.Context, context.CancelFunc) {
	watched, cancel := context.WithCancel(ctx)
	parent := os.Getppid()
	if parent <= 1 {
		return watched, cancel
	}
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-watched.Done():
				return
			case <-ticker.C:
				if os.Getppid() != parent {
					cancel()
					return
				}
			}
		}
	}()
	return watched, cancel
}

// RunOpenClawCloudAgentCLI is the runner-host shim used only after Workbench's
// existing provider router has selected OpenClaw. It dynamically discovers the
// effective OpenAI/Anthropic catalogue, prefers the current OpenClaw default for
// routine work, escalates high-risk work, and falls back across models without
// changing Workbench's outer provider ordering. If discovery fails, it executes
// the exact legacy OpenClaw invocation with no model override.
func RunOpenClawCloudAgentCLI(ctx context.Context, agentArgs []string, stdout, stderr io.Writer) int {
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
	candidates := RankOpenClawCloudModels(catalog, intent)
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

// routeRunnerOpenClawThroughWorkbench swaps only the runner-host OpenClaw
// executable for Workbench Runner's own bounded shim. The desktop/local OpenClaw
// provider remains unchanged, and the outer provider order is untouched.
func routeRunnerOpenClawThroughWorkbench(base []Provider, executable string) []Provider {
	executable = strings.TrimSpace(executable)
	if executable == "" {
		return append([]Provider(nil), base...)
	}
	out := append([]Provider(nil), base...)
	for i := range out {
		if out[i].ID != "openclaw" || !out[i].Installed || strings.TrimSpace(out[i].Command) == "" {
			continue
		}
		out[i].Command = executable
		status := strings.TrimSpace(out[i].Status)
		if status != "" {
			status += " · "
		}
		out[i].Status = status + "dynamic cloud model routing"
		out[i].Notes = "OpenClaw remains one Workbench provider; once selected on the runner it discovers allowed cloud models dynamically, uses the current OpenClaw default for routine work, and escalates within that cloud stage when needed."
	}
	return out
}
