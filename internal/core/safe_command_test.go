package core

import "testing"

func TestSafeGitInspectionPolicy(t *testing.T) {
	for _, line := range []string{
		"git status --short",
		"git diff --stat",
		"git log --oneline -5",
		"git show HEAD",
		"git branch",
		"git branch --show-current",
	} {
		if !IsSafeCommand(line) {
			t.Fatalf("expected safe command: %s", line)
		}
	}

	for _, line := range []string{
		"git branch topic",
		"git branch -D topic",
		"git diff --no-index a b",
		"git diff --ext-diff",
		"git show --textconv HEAD",
		"git diff --output=report.txt",
	} {
		if IsSafeCommand(line) {
			t.Fatalf("expected command to be rejected: %s", line)
		}
	}
}
