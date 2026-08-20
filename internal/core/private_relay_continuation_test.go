package core

import (
	"strings"
	"testing"
	"time"
)

func TestPrivateRelayContinuationSealRoundTrips(t *testing.T) {
	project := t.TempDir()
	sealed, err := SealPrivateRelayContinuationIntent("Bearer secret-token", "relay_dev_001", project, "Implement the remaining verified scope and keep going until complete.")
	if err != nil {
		t.Fatal(err)
	}
	clean, ok := ValidatePrivateRelayContinuationIntent(sealed, project, "secret-token")
	if !ok {
		t.Fatal("valid private relay continuation seal was rejected")
	}
	if clean != "Implement the remaining verified scope and keep going until complete." {
		t.Fatalf("clean intent=%q", clean)
	}
	if strings.Contains(clean, "relay_dev_001") || strings.Contains(clean, "proof") {
		t.Fatalf("transport proof leaked into worker intent: %q", clean)
	}
}

func TestPrivateRelayContinuationSealRejectsSpoofing(t *testing.T) {
	project := t.TempDir()
	sealed, err := SealPrivateRelayContinuationIntent("Bearer secret-token", "relay_dev_002", project, "Continue the exact development task.")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ValidatePrivateRelayContinuationIntent(sealed, project, "wrong-token"); ok {
		t.Fatal("continuation accepted with the wrong MCP credential")
	}
	if _, ok := ValidatePrivateRelayContinuationIntent(strings.Replace(sealed, "exact development", "different development", 1), project, "secret-token"); ok {
		t.Fatal("continuation accepted after intent tampering")
	}
	if _, ok := ValidatePrivateRelayContinuationIntent(sealed+"\nnot a Workbench dependency update", project, "secret-token"); ok {
		t.Fatal("continuation accepted arbitrary text appended after the proof")
	}
}

func TestPrivateRelayDeferredContinuationSurvivesWaitEnvelopeRemoval(t *testing.T) {
	project := t.TempDir()
	continuation := "Inspect the CI result and continue automatically through the remaining in-scope work."
	intent := `WORKBENCH_WAIT_GITHUB_ACTIONS:{"repository":"DaisyCloverSoftware/workbench","run_id":12345}` + "\n" + continuation
	sealed, err := SealPrivateRelayContinuationIntent("Bearer secret-token", "relay_ci_001", project, intent)
	if err != nil {
		t.Fatal(err)
	}
	_, parsedContinuation, matched, err := parseDeferredGitHubActionsIntent(sealed, time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC))
	if err != nil || !matched {
		t.Fatalf("deferred continuation did not parse: matched=%t err=%v", matched, err)
	}
	clean, ok := ValidatePrivateRelayContinuationIntent(parsedContinuation, project, "secret-token")
	if !ok {
		t.Fatal("continuation seal did not survive durable dependency wait parsing")
	}
	if clean != continuation {
		t.Fatalf("clean continuation=%q", clean)
	}
}

func TestPrivateRelayDeferredContinuationSurvivesWorkbenchDependencyUpdate(t *testing.T) {
	project := t.TempDir()
	continuation := "Inspect the CI result and continue automatically through the remaining in-scope work."
	intent := `WORKBENCH_WAIT_GITHUB_ACTIONS:{"repository":"DaisyCloverSoftware/workbench","run_id":12345}` + "\n" + continuation
	sealed, err := SealPrivateRelayContinuationIntent("Bearer secret-token", "relay_ci_002", project, intent)
	if err != nil {
		t.Fatal(err)
	}
	_, parsedContinuation, matched, err := parseDeferredGitHubActionsIntent(sealed, time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC))
	if err != nil || !matched {
		t.Fatalf("deferred continuation did not parse: matched=%t err=%v", matched, err)
	}
	update := "Workbench dependency update:\nGitHub Actions run 12345 (build) completed with conclusion success.\nContinue the original task now. If CI failed, inspect the failure and fix it autonomously when the repair is within the existing scope. If it succeeded, continue the remaining in-scope work. Only require human attention for a genuine decision or authority boundary."
	resumedIntent := strings.TrimSpace(parsedContinuation) + "\n\n" + update
	clean, ok := ValidatePrivateRelayContinuationIntent(resumedIntent, project, "secret-token")
	if !ok {
		t.Fatal("continuation seal was rejected after Workbench appended its dependency update")
	}
	want := continuation + "\n\n" + update
	if clean != want {
		t.Fatalf("clean resumed intent=%q want %q", clean, want)
	}
	if strings.Contains(clean, "relay_ci_002") || strings.Contains(clean, "relay-continuation-proof") {
		t.Fatalf("transport proof leaked into resumed worker intent: %q", clean)
	}
}
