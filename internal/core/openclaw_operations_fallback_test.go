package core

import (
	"errors"
	"reflect"
	"testing"
)

func TestOpenClawOperationAgentArgsForModelAddsExplicitOverride(t *testing.T) {
	prompt := "verify cluster health"
	got := openClawOperationAgentArgsForModel(prompt, "ollama/qwen2.5-coder:7b")
	want := []string{"agent", "--agent", "main", "--model", "ollama/qwen2.5-coder:7b", "--message", prompt}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("fallback args=%q want %q", got, want)
	}
}

func TestOpenClawOperationAgentArgsForModelPreservesPrimaryShapeWithoutOverride(t *testing.T) {
	prompt := "verify cluster health"
	got := openClawOperationAgentArgsForModel(prompt, "")
	want := openClawOperationAgentArgs(prompt)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unoverridden args=%q want %q", got, want)
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
		t.Fatal("subscription usage exhaustion should trigger local model fallback")
	}
	if operationModelCapacityFailure(RunResult{Authentication: true}, errors.New("quota text but authentication failed")) {
		t.Fatal("authentication failures must not be disguised as model-capacity fallback")
	}
	if operationModelCapacityFailure(RunResult{WorkerUnavailable: "kubectl missing"}, errors.New("quota text")) {
		t.Fatal("explicit worker-local unavailability must remain authoritative")
	}
}
