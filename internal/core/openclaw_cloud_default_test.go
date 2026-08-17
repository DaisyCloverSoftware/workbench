package core

import "testing"

func TestOpenClawCloudModelProviderIDRoundTrip(t *testing.T) {
	for _, model := range []string{
		"openai/gpt-5.3-codex-spark",
		"openai/gpt-5.6-sol",
		"anthropic/claude-sonnet-4-6",
	} {
		id, err := RunnerCloudModelProviderID(model)
		if err != nil {
			t.Fatalf("provider id for %q: %v", model, err)
		}
		got, ok := RunnerCloudModelRefFromProviderID(id)
		if !ok || got != model {
			t.Fatalf("round trip %q -> %q, ok=%v", model, got, ok)
		}
	}
}

func TestOpenClawCloudModelReferenceRejectsUnsafeOrOtherProviders(t *testing.T) {
	for _, model := range []string{
		"",
		"ollama/qwen3:8b",
		"google/gemini-3.5-pro",
		"openai/gpt-5.6-sol & whoami",
		"anthropic/claude-sonnet-4-6;id",
	} {
		if _, err := normalizeOpenClawCloudModelRef(model); err == nil {
			t.Fatalf("expected %q to be rejected", model)
		}
	}
}
