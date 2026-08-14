package core

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestProviderRetryableFailureCreatesCategoricalBackoff(t *testing.T) {
	isolateProviderHealthCache(t)
	now := time.Date(2026, 8, 14, 20, 0, 0, 0, time.UTC)
	res := RunResult{Retryable: true, Authentication: true, Output: "secret-account-detail-should-not-be-stored"}

	first, err := recordProviderRetryableFailureAt("claude", res, errors.New("authentication failed for hidden account"), now)
	if err != nil {
		t.Fatal(err)
	}
	if first.Reason != "authentication unavailable" || first.ConsecutiveFailures != 1 || first.CooldownUntil.Sub(now) != 10*time.Minute {
		t.Fatalf("unexpected first cooldown: %#v", first)
	}
	secondNow := now.Add(time.Minute)
	second, err := recordProviderRetryableFailureAt("claude", res, errors.New("authentication failed again"), secondNow)
	if err != nil {
		t.Fatal(err)
	}
	if second.ConsecutiveFailures != 2 || second.CooldownUntil.Sub(secondNow) != 20*time.Minute {
		t.Fatalf("unexpected second cooldown: %#v", second)
	}

	path, err := ProviderHealthStatePath()
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	if strings.Contains(text, "secret-account-detail") || strings.Contains(text, "hidden account") {
		t.Fatalf("provider health cache persisted raw provider output/error: %s", text)
	}
}

func TestFilterProviderCooldownsSkipsOnlyActiveProvider(t *testing.T) {
	isolateProviderHealthCache(t)
	now := time.Date(2026, 8, 14, 20, 0, 0, 0, time.UTC)
	if _, err := recordProviderRetryableFailureAt("openclaw", RunResult{Retryable: true, WorkerUnavailable: "quota exceeded"}, errors.New("quota exceeded"), now); err != nil {
		t.Fatal(err)
	}
	providers := []Provider{
		{ID: "openclaw", Name: "OpenClaw"},
		{ID: "claude", Name: "Claude"},
	}
	ready, skipped := FilterProviderCooldowns(providers, now.Add(time.Second))
	if len(ready) != 1 || ready[0].ID != "claude" {
		t.Fatalf("ready=%#v", ready)
	}
	if len(skipped) != 1 || !strings.Contains(skipped[0], "quota or rate limit") {
		t.Fatalf("skipped=%#v", skipped)
	}
	ready, skipped = FilterProviderCooldowns(providers, now.Add(6*time.Minute))
	if len(ready) != 2 || len(skipped) != 0 {
		t.Fatalf("expired cooldown still filtered: ready=%#v skipped=%#v", ready, skipped)
	}
}

func TestApplyProviderHealthAddsSafeStatus(t *testing.T) {
	isolateProviderHealthCache(t)
	now := time.Date(2026, 8, 14, 20, 0, 0, 0, time.UTC)
	if _, err := recordProviderRetryableFailureAt("openclaw", RunResult{Retryable: true, WorkerUnavailable: "tool calls are denied"}, errors.New("tool calls are denied"), now); err != nil {
		t.Fatal(err)
	}
	providers := ApplyProviderHealth([]Provider{{ID: "openclaw", Status: "CLI detected"}}, now.Add(time.Second))
	if len(providers) != 1 || !strings.Contains(providers[0].Status, "cooldown") || !strings.Contains(providers[0].Status, "worker tool permissions unavailable") {
		t.Fatalf("unexpected enriched provider: %#v", providers)
	}
}

func TestProviderSuccessClearsCooldownAndNonRetryableFailureDoesNotCreateOne(t *testing.T) {
	isolateProviderHealthCache(t)
	now := time.Date(2026, 8, 14, 20, 0, 0, 0, time.UTC)
	if _, err := recordProviderRetryableFailureAt("claude", RunResult{Retryable: true}, errors.New("temporary"), now); err != nil {
		t.Fatal(err)
	}
	if _, recorded := RecordProviderRunOutcome("claude", RunResult{}, nil); recorded {
		t.Fatal("successful outcome unexpectedly reported a cooldown")
	}
	ready, skipped := FilterProviderCooldowns([]Provider{{ID: "claude"}}, now.Add(time.Second))
	if len(ready) != 1 || len(skipped) != 0 {
		t.Fatalf("success did not clear cooldown: ready=%#v skipped=%#v", ready, skipped)
	}

	if _, recorded := RecordProviderRunOutcome("codex", RunResult{Retryable: false}, errors.New("task-specific test failure")); recorded {
		t.Fatal("non-retryable task failure created provider cooldown")
	}
	ready, skipped = FilterProviderCooldowns([]Provider{{ID: "codex"}}, now.Add(time.Second))
	if len(ready) != 1 || len(skipped) != 0 {
		t.Fatalf("non-retryable failure poisoned provider: ready=%#v skipped=%#v", ready, skipped)
	}
}

func TestClearAllProviderCooldownsSupportsExplicitRescan(t *testing.T) {
	isolateProviderHealthCache(t)
	now := time.Date(2026, 8, 14, 20, 0, 0, 0, time.UTC)
	for _, id := range []string{"claude", "openclaw"} {
		if _, err := recordProviderRetryableFailureAt(id, RunResult{Retryable: true}, errors.New("temporary"), now); err != nil {
			t.Fatal(err)
		}
	}
	if err := ClearAllProviderCooldowns(); err != nil {
		t.Fatal(err)
	}
	ready, skipped := FilterProviderCooldowns([]Provider{{ID: "claude"}, {ID: "openclaw"}}, now.Add(time.Second))
	if len(ready) != 2 || len(skipped) != 0 {
		t.Fatalf("clear-all did not reset cooldowns: ready=%#v skipped=%#v", ready, skipped)
	}
}

func isolateProviderHealthCache(t *testing.T) {
	t.Helper()
	cache := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cache)
	t.Setenv("LOCALAPPDATA", cache)
}
