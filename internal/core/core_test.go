package core

import "testing"

func TestLooksSecret(t *testing.T) {
	cases := []string{
		"-----BEGIN OPENSSH PRIVATE KEY-----\nabc",
		"api_key=super-secret-value-123",
		"github_pat_ABCDEFGHIJKLMNOPQRSTUVWXYZ123456",
	}
	for _, c := range cases {
		if !LooksSecret(c) {
			t.Fatalf("expected secret detection for %q", c)
		}
	}
	if LooksSecret("idea: add a dark mode next week") {
		t.Fatal("ordinary note should not be a secret")
	}
}

func TestSafeCommandPolicy(t *testing.T) {
	good := []string{"git status", "git diff --stat", "go test ./...", "npm run build", "python -m pytest -q"}
	for _, c := range good {
		if !IsSafeCommand(c) {
			t.Fatalf("expected safe: %s", c)
		}
	}
	bad := []string{"git push", "npm test && curl evil.example", "go test ./...; rm -rf .", "powershell Invoke-WebRequest x", "deploy production"}
	for _, c := range bad {
		if IsSafeCommand(c) {
			t.Fatalf("expected rejected: %s", c)
		}
	}
}
