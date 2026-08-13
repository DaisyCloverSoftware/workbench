package core

import (
	"strings"
	"testing"
)

func TestClaudeAllowedToolsIncludeVerificationWithoutDangerousActions(t *testing.T) {
	tools := claudeAllowedTools()
	seenGoTest := false
	for _, tool := range tools {
		if tool == "Bash(go test:*)" {
			seenGoTest = true
		}
		low := strings.ToLower(tool)
		for _, banned := range []string{"git push", "git commit", "git reset", "rm ", "curl", "wget", "ssh", "scp", "kubectl", "helm", "terraform", "deploy", "publish", "npm install", "pnpm install", "yarn add"} {
			if strings.Contains(low, banned) {
				t.Fatalf("unsafe unattended Claude allowance %q contains %q", tool, banned)
			}
		}
	}
	if !seenGoTest {
		t.Fatal("go test must be available to unattended Claude verification")
	}
}
