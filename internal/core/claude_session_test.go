package core

import (
	"strings"
	"testing"
)

func TestParseClaudePrintResponseCapturesResultAndSession(t *testing.T) {
	response, ok := parseClaudePrintResponse(`{"type":"result","subtype":"success","result":"done","session_id":"550e8400-e29b-41d4-a716-446655440000","total_cost_usd":0.01}`)
	if !ok {
		t.Fatal("valid Claude JSON response was not parsed")
	}
	if response.Result != "done" || response.SessionID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("unexpected Claude response: %#v", response)
	}
	if _, ok := parseClaudePrintResponse("not-json"); ok {
		t.Fatal("non-JSON Claude output was treated as structured response")
	}
}

func TestClaudeInvocationArgsResumeOnlyWithExplicitStoredSession(t *testing.T) {
	fresh := claudeInvocationArgs("do work", "", false)
	if strings.Contains(strings.Join(fresh, "\x00"), "--resume") {
		t.Fatalf("fresh Claude invocation unexpectedly resumes: %#v", fresh)
	}
	resumeID := "550e8400-e29b-41d4-a716-446655440000"
	resumed := claudeInvocationArgs("continue work", resumeID, true)
	joined := strings.Join(resumed, "\x00")
	if !strings.Contains(joined, "--resume\x00"+resumeID) {
		t.Fatalf("Claude resume args do not bind the stored session: %#v", resumed)
	}
	if !strings.Contains(joined, "--output-format\x00json") || !strings.Contains(joined, "--permission-mode\x00acceptEdits") {
		t.Fatalf("Claude resume lost existing autonomous execution controls: %#v", resumed)
	}
}

func TestClaudeSessionIDRequiresUUIDShape(t *testing.T) {
	for _, valid := range []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"019F244A-489A-7482-803E-1644660FAFB7",
	} {
		if !validClaudeSessionID(valid) {
			t.Fatalf("valid UUID-shaped Claude session rejected: %q", valid)
		}
	}
	for _, invalid := range []string{"", "abc", "550e8400-e29b-41d4-a716-44665544000z", "550e8400e29b41d4a716446655440000"} {
		if validClaudeSessionID(invalid) {
			t.Fatalf("invalid Claude session accepted: %q", invalid)
		}
	}
}

func TestClaudeResumeFallbackDetectionIsNarrow(t *testing.T) {
	for _, text := range []string{
		"Session not found",
		"invalid session id",
		"Cannot resume conversation because it does not exist",
	} {
		if !claudeResumeUnavailable(text) {
			t.Fatalf("clear missing-session error not detected: %q", text)
		}
	}
	for _, text := range []string{
		"rate limit reached",
		"authentication failed",
		"session started successfully",
		"file not found",
	} {
		if claudeResumeUnavailable(text) {
			t.Fatalf("unrelated provider failure triggered fresh-session fallback: %q", text)
		}
	}
}
