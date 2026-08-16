package core

import (
	"strings"
	"testing"
)

func TestParseOpenClawModelListKeepsAvailableCloudModelsOnly(t *testing.T) {
	body := []byte(`{
	  "models": [
	    {"key":"openai/gpt-5.3-codex-spark","name":"GPT-5.3 Codex Spark","input":"text","contextWindow":131072,"local":false,"available":true,"tags":["default"]},
	    {"key":"openai/gpt-5.6-sol","name":"GPT-5.6 Sol","input":"text+image","contextWindow":400000,"contextTokens":272000,"local":false,"available":true},
	    {"key":"anthropic/claude-sonnet-4-6","name":"Claude Sonnet","input":["text","image"],"contextWindow":200000,"local":false,"available":true},
	    {"key":"ollama/qwen3.5:9b","name":"qwen","input":"text","local":true,"available":true},
	    {"key":"openai/gpt-retired","name":"Retired","input":"text","local":false,"available":false}
	  ]
	}`)
	models, err := parseOpenClawModelList(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 3 {
		t.Fatalf("expected 3 cloud models, got %#v", models)
	}
	byKey := map[string]OpenClawCloudModel{}
	for _, model := range models {
		byKey[model.Key] = model
	}
	if !byKey["openai/gpt-5.6-sol"].Image || byKey["openai/gpt-5.6-sol"].ContextTokens != 272000 {
		t.Fatalf("expected image/runtime context metadata, got %#v", byKey["openai/gpt-5.6-sol"])
	}
	if !byKey["anthropic/claude-sonnet-4-6"].Image {
		t.Fatalf("expected array input metadata to expose image capability: %#v", byKey["anthropic/claude-sonnet-4-6"])
	}
}

func TestRoutineOpenClawRoutingStartsAtCurrentDefaultAndRetainsFlagshipEscalation(t *testing.T) {
	catalog := OpenClawCloudCatalog{
		DefaultModel: "openai/gpt-5.3-codex-spark",
		Models: []OpenClawCloudModel{
			{Key: "openai/gpt-5.3-codex-spark", Provider: "openai", Available: true, Default: true},
			{Key: "openai/gpt-5.4-mini", Provider: "openai", Available: true},
			{Key: "openai/gpt-5.6-terra", Provider: "openai", Available: true},
			{Key: "anthropic/claude-sonnet-4-6", Provider: "anthropic", Available: true},
			{Key: "openai/gpt-5.6-sol", Provider: "openai", Available: true},
		},
	}
	ranked := RankOpenClawCloudModels(catalog, "Update routine CI YAML and fix a small script bug")
	if len(ranked) == 0 || ranked[0].Key != catalog.DefaultModel {
		t.Fatalf("routine routing must start at current OpenClaw default, got %#v", ranked)
	}
	foundSol := false
	foundAnthropic := false
	for _, model := range ranked {
		foundSol = foundSol || model.Key == "openai/gpt-5.6-sol"
		foundAnthropic = foundAnthropic || canonicalOpenClawProvider(model.Provider) == "anthropic"
	}
	if !foundSol {
		t.Fatalf("routine ladder must retain a flagship escalation path: %#v", ranked)
	}
	if !foundAnthropic {
		t.Fatalf("routine ladder must retain cross-provider cloud fallback: %#v", ranked)
	}
}

func TestHighRiskOpenClawRoutingEscalatesBeforeSpark(t *testing.T) {
	catalog := OpenClawCloudCatalog{
		DefaultModel: "openai/gpt-5.3-codex-spark",
		Models: []OpenClawCloudModel{
			{Key: "openai/gpt-5.3-codex-spark", Provider: "openai", Available: true, Default: true},
			{Key: "openai/gpt-5.6-sol", Provider: "openai", Available: true},
			{Key: "anthropic/claude-opus-4-6", Provider: "anthropic", Available: true},
			{Key: "anthropic/claude-sonnet-4-6", Provider: "anthropic", Available: true},
			{Key: "openai/gpt-5.6-terra", Provider: "openai", Available: true},
		},
	}
	ranked := RankOpenClawCloudModels(catalog, "Investigate a security incident involving firewall rules and possible data loss")
	if len(ranked) < 2 {
		t.Fatalf("expected an escalation ladder, got %#v", ranked)
	}
	if ranked[0].Key != "openai/gpt-5.6-sol" {
		t.Fatalf("hardest/high-risk work should prefer Sol when available, got %#v", ranked)
	}
	for i, model := range ranked {
		if model.Key == catalog.DefaultModel && i < 2 {
			t.Fatalf("Spark default should not outrank flagship models for high-risk work: %#v", ranked)
		}
	}
}

func TestOpenClawRoutingDoesNotRequirePermanentCatalogue(t *testing.T) {
	body := []byte(`[{"key":"openai/gpt-5.7-new-family","name":"Future Model","input":"text","local":false,"available":true}]`)
	models, err := parseOpenClawModelList(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 || models[0].Key != "openai/gpt-5.7-new-family" {
		t.Fatalf("future discovered OpenAI model must remain routable without a code catalogue update: %#v", models)
	}
	ranked := RankOpenClawCloudModels(OpenClawCloudCatalog{Models: models}, "Implement a normal engineering change")
	if len(ranked) != 1 || ranked[0].Key != models[0].Key {
		t.Fatalf("future discovered model should remain eligible: %#v", ranked)
	}
}

func TestOpenClawWorkerPromptRiskUsesIntentNotSafetyBoilerplate(t *testing.T) {
	prompt := BuildWorkerPrompt(Task{ProjectPath: "/example/project", Intent: "Fix a CSS alignment issue"})
	intent := workbenchIntentFromWorkerPrompt(prompt)
	if intent != "Fix a CSS alignment issue" {
		t.Fatalf("unexpected extracted intent %q", intent)
	}
	if openClawCloudEscalationNeeded(intent) {
		t.Fatalf("routine UI task must not escalate merely because worker rules mention security boundaries")
	}
}

func TestOpenClawWrapperRejectsArbitraryCLIArguments(t *testing.T) {
	if _, err := parseOpenClawAgentArgs([]string{"--message", "hello", "--model", "openai/gpt-5.6-sol"}); err == nil {
		t.Fatal("wrapper must not accept caller-controlled model or arbitrary OpenClaw flags")
	}
	prompt, err := parseOpenClawAgentArgs([]string{"--message", "hello", "--headless"})
	if err != nil || prompt != "hello" {
		t.Fatalf("expected existing Workbench invocation shape to remain valid, prompt=%q err=%v", prompt, err)
	}
}

func TestRunnerOpenClawOverlayChangesOnlyRunnerExecutionCommand(t *testing.T) {
	base := []Provider{
		{ID: "openclaw", Name: "OpenClaw", Command: "/opt/example/openclaw", Installed: true, Authenticated: true},
		{ID: "claude", Name: "Claude", Command: "/opt/example/claude", Installed: true, Authenticated: true},
	}
	wrapped := routeRunnerOpenClawThroughWorkbench(base, "/opt/example/workbench-runner")
	if wrapped[0].Command != "/opt/example/workbench-runner" {
		t.Fatalf("runner OpenClaw must route through Workbench shim: %#v", wrapped[0])
	}
	if wrapped[1].Command != base[1].Command {
		t.Fatalf("non-OpenClaw provider must remain untouched: %#v", wrapped[1])
	}
	if base[0].Command == wrapped[0].Command {
		t.Fatal("overlay must not mutate the original provider slice")
	}
	if !strings.Contains(wrapped[0].Status, "dynamic cloud model routing") {
		t.Fatalf("runner inventory should explain the model-routing layer: %#v", wrapped[0])
	}
}
