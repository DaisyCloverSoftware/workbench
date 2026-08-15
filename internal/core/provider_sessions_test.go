package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func isolateProviderSessions(t *testing.T) string {
	t.Helper()
	old := providerSessionPathOverride
	path := filepath.Join(t.TempDir(), "config", "provider-sessions.json")
	providerSessionPathOverride = path
	t.Cleanup(func() { providerSessionPathOverride = old })
	return path
}

func TestProviderSessionRoundTripIsolationAndTaskCleanup(t *testing.T) {
	path := isolateProviderSessions(t)
	if _, err := SaveProviderSession("task-a", "claude", "550e8400-e29b-41d4-a716-446655440000"); err != nil {
		t.Fatal(err)
	}
	if _, err := SaveProviderSession("task-a", "codex", "019f244a-489a-7482-803e-1644660fafb7"); err != nil {
		t.Fatal(err)
	}
	if _, err := SaveProviderSession("task-b", "claude", "123e4567-e89b-12d3-a456-426614174000"); err != nil {
		t.Fatal(err)
	}

	claude, ok, err := ProviderSessionFor("task-a", "claude")
	if err != nil || !ok || claude.SessionID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("claude session=%#v ok=%t err=%v", claude, ok, err)
	}
	codex, ok, err := ProviderSessionFor("task-a", "codex")
	if err != nil || !ok || codex.SessionID != "019f244a-489a-7482-803e-1644660fafb7" {
		t.Fatalf("codex session=%#v ok=%t err=%v", codex, ok, err)
	}
	if claude.SessionID == codex.SessionID {
		t.Fatal("provider session identities were conflated")
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() {
		t.Fatalf("provider session store is not regular: %v", info.Mode())
	}

	if err := DeleteProviderSession("task-a", "claude"); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := ProviderSessionFor("task-a", "claude"); err != nil || ok {
		t.Fatalf("single provider session survived delete: ok=%t err=%v", ok, err)
	}
	if _, ok, err := ProviderSessionFor("task-a", "codex"); err != nil || !ok {
		t.Fatalf("deleting Claude removed Codex session: ok=%t err=%v", ok, err)
	}

	if err := DeleteTaskProviderSessions("task-a"); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := ProviderSessionFor("task-a", "codex"); err != nil || ok {
		t.Fatalf("task-wide cleanup left task-a session: ok=%t err=%v", ok, err)
	}
	if _, ok, err := ProviderSessionFor("task-b", "claude"); err != nil || !ok {
		t.Fatalf("task-wide cleanup removed another task: ok=%t err=%v", ok, err)
	}
}

func TestProviderSessionStoreIsStrictBoundedAndPrivacyMinimal(t *testing.T) {
	path := isolateProviderSessions(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	cases := []string{
		`{"version":1,"sessions":[],"unknown":true}`,
		`{"version":1,"sessions":[]}{"version":1,"sessions":[]}`,
		`{"version":2,"sessions":[]}`,
		`{"version":1,"sessions":[{"task_id":"t","provider_id":"p","session_id":"bad session"}]}`,
		`{"version":1,"sessions":[{"task_id":"t","provider_id":"p","session_id":"one"},{"task_id":"t","provider_id":"p","session_id":"two"}]}`,
	}
	for _, body := range cases {
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, _, err := ProviderSessionFor("t", "p"); err == nil {
			t.Fatalf("invalid provider-session store was accepted: %s", body)
		}
	}
	if err := os.WriteFile(path, []byte(strings.Repeat("x", maxProviderSessionStoreBytes+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ProviderSessionFor("t", "p"); err == nil {
		t.Fatal("oversized provider-session store was accepted")
	}

	if _, err := SaveProviderSession("task", "claude", strings.Repeat("x", maxProviderSessionIDBytes+1)); err == nil {
		t.Fatal("oversized provider session id was accepted")
	}
}

func TestProviderSessionStoreContainsNoTaskPayloadOrProjectMetadata(t *testing.T) {
	path := isolateProviderSessions(t)
	sessionID := "550e8400-e29b-41d4-a716-446655440000"
	if _, err := SaveProviderSession("task-private", "claude", sessionID); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ToLower(string(body))
	if !strings.Contains(text, sessionID) {
		t.Fatalf("session id not persisted: %s", body)
	}
	for _, forbidden := range []string{"project_path", "intent", "prompt", "output", "transcript", "remote_url", "credential", "account"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("provider-session store exposed forbidden field %q: %s", forbidden, body)
		}
	}
}
