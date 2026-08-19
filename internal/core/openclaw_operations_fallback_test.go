package core

import (
	"errors"
	"reflect"
	"testing"
)

func TestOpenClawOperationAgentArgsForModelAddsExplicitOverrideInSameJobSession(t *testing.T) {
	task := Task{ID: "task-operation-001"}
	prompt := "verify cluster health"
	sessionID := openClawOperationSessionID(task)
	got := openClawOperationAgentArgsForModel(task, prompt, "anthropic/claude-sonnet")
	want := []string{"agent", "--agent", "main", "--session-id", sessionID, "--model", "anthropic/claude-sonnet", "--message", prompt}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("fallback args=%q want %q", got, want)
	}
	primary := openClawOperationAgentArgs(task, prompt)
	if primary[4] != got[4] {
		t.Fatalf("model failover changed OpenClaw job conversation: primary=%q fallback=%q", primary, got)
	}
}

func TestPreferredOpenClawOperationsCloudFallbackMovesFromCodexToClaude(t *testing.T) {
	catalog := OpenClawCloudCatalog{
		DefaultModel: "openai/gpt-5.3-codex-spark",
		Models: []OpenClawCloudModel{
			{Key: "openai/gpt-5.3-codex-spark", Provider: "openai", Available: true, Default: true},
			{Key: "anthropic/claude-opus", Provider: "anthropic", Available: true},
			{Key: "anthropic/claude-sonnet", Provider: "anthropic", Available: true},
		},
	}
	model, ok := preferredOpenClawOperationsCloudFallback(catalog, "openai")
	if !ok || model.Key != "anthropic/claude-sonnet" {
		t.Fatalf("Claude operations fallback=%+v ok=%t", model, ok)
	}
}

func TestPreferredOpenClawOperationsCloudFallbackSkipsCoolingClaude(t *testing.T) {
	catalog := OpenClawCloudCatalog{Models: []OpenClawCloudModel{
		{Key: "anthropic/claude-sonnet", Provider: "anthropic", Available: true, Cooling: true},
		{Key: "anthropic/claude-opus", Provider: "anthropic", Available: true},
	}}
	model, ok := preferredOpenClawOperationsCloudFallback(catalog, "openai")
	if !ok || model.Key != "anthropic/claude-opus" {
		t.Fatalf("fallback should skip cooling Sonnet: %+v ok=%t", model, ok)
	}
}

func TestPreferredOllamaOperationsModelPrefersQwenCoder(t *testing.T) {
	inventory := `NAME                    ID              SIZE      MODIFIED
nomic-embed-text:latest 0a109f422b47    274 MB    3 days ago
llama3.2:3b             a80c4f17acd5    2.0 GB    2 days ago
qwen2.5:7b              845dbda0ea48    4.7 GB    1 day ago
qwen2.5-coder:7b        dae161e27b0e    4.7 GB    1 day ago
`
	if got := preferredOllamaOperationsModel(inventory); got != "qwen2.5-coder:7b" {
		t.Fatalf("selected local fallback=%q", got)
	}
}

func TestPreferredOllamaOperationsModelIgnoresEmbeddingOnlyInventory(t *testing.T) {
	inventory := `NAME                    ID              SIZE
nomic-embed-text:latest 0a109f422b47    274 MB
bge-m3:latest           790764642607    1.2 GB
`
	if got := preferredOllamaOperationsModel(inventory); got != "" {
		t.Fatalf("embedding model selected as operations fallback: %q", got)
	}
}

func TestOperationModelCapacityFailureRecognisesProviderUsageExhaustion(t *testing.T) {
	res := RunResult{Output: "You've reached your Codex subscription usage limit. Next reset later."}
	if !operationModelCapacityFailure(res, errors.New("OpenClaw operations invocation exited with error")) {
		t.Fatal("subscription usage exhaustion should trigger cloud/local model fallback")
	}
	if provider := operationFailedCloudProvider(res, errors.New("provider failure"), OpenClawCloudCatalog{}); provider != "openai" {
		t.Fatalf("failed provider=%q want openai", provider)
	}
	if operationModelCapacityFailure(RunResult{Authentication: true}, errors.New("quota text but authentication failed")) {
		t.Fatal("authentication failures must not be disguised as model-capacity fallback")
	}
	if operationModelCapacityFailure(RunResult{WorkerUnavailable: "kubectl missing"}, errors.New("quota text")) {
		t.Fatal("explicit worker-local unavailability must remain authoritative")
	}
}

func TestOperationModelCapacityFailureRecognisesUndersizedFallbackContext(t *testing.T) {
	for _, output := range []string{
		"Context overflow: prompt too large for the model. Try /reset (or /new) to start a fresh session, or use a larger-context model.",
		"context length exceeded",
		"maximum context length reached",
		"operation does not fit the context window",
	} {
		if !operationModelCapacityFailure(RunResult{Output: output}, errors.New("OpenClaw operations invocation exited with error")) {
			t.Fatalf("context-size failure was not treated as model-route capacity: %q", output)
		}
	}
}

func TestContextOverflowDefaultsFailedProviderToConfiguredOpenAIDefault(t *testing.T) {
	catalog := OpenClawCloudCatalog{DefaultModel: "openai/gpt-5.3-codex-spark"}
	res := RunResult{Output: "Context overflow: prompt too large for the model."}
	if got := operationFailedCloudProvider(res, errors.New("model could not fit prompt"), catalog); got != "openai" {
		t.Fatalf("failed provider=%q want openai", got)
	}
}
