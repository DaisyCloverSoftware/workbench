package core

import (
	"strings"
	"testing"
)

func TestRunnerCloudModelProviderInfoCarriesSafeMetadata(t *testing.T) {
	info, ok := runnerCloudModelProviderInfo(OpenClawCloudModel{
		Key:           "openai/gpt-5.4",
		Provider:      "openai",
		Name:          "GPT-5.4",
		Input:         "text+image",
		ContextWindow: 400000,
		ContextTokens: 272000,
		Available:     true,
	})
	if !ok {
		t.Fatal("expected available OpenAI model to be exposed")
	}
	if info.Ready || !info.Installed || !info.Authenticated {
		t.Fatalf("unexpected selectable model state: %#v", info)
	}
	if !strings.HasPrefix(info.ID, RunnerCloudModelProviderIDPrefix) || !strings.Contains(info.Capability, "context 400k") || !strings.Contains(info.Capability, "text+image") {
		t.Fatalf("expected safe model metadata in runner inventory: %#v", info)
	}
	if strings.Contains(strings.ToLower(info.Status), "token") || strings.Contains(strings.ToLower(info.Status), "account") {
		t.Fatalf("runner model inventory must not expose auth/account material: %#v", info)
	}
}

func TestRunnerCloudModelDefaultIsShownReadyWithoutChangingOuterProvider(t *testing.T) {
	info, ok := runnerCloudModelProviderInfo(OpenClawCloudModel{
		Key:       "anthropic/claude-sonnet-4-6",
		Provider:  "anthropic",
		Name:      "Claude Sonnet",
		Available: true,
		Default:   true,
	})
	if !ok || !info.Ready || !strings.Contains(info.Status, "current OpenClaw default") {
		t.Fatalf("expected current default marker, got %#v", info)
	}
}
