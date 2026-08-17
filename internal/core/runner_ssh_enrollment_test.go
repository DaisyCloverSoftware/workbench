package core

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestNormalizeRunnerSSHPublicKeyDropsLocalComment(t *testing.T) {
	payload := base64.StdEncoding.EncodeToString(make([]byte, 48))
	got, err := normalizeRunnerSSHPublicKey("ssh-ed25519 " + payload + " private-machine-name@example")
	if err != nil {
		t.Fatalf("normalize public key: %v", err)
	}
	want := "ssh-ed25519 " + payload + " workbench-runner"
	if got != want {
		t.Fatalf("normalized key=%q, want %q", got, want)
	}
}

func TestRunnerSSHEnrollmentPromptPreservesHardening(t *testing.T) {
	payload := base64.StdEncoding.EncodeToString(make([]byte, 48))
	prompt := runnerSSHEnrollmentPrompt("operator", "ssh-ed25519 "+payload+" workbench-runner")
	for _, want := range []string{
		"existing runner account operator",
		"do NOT enable password authentication",
		"do not replace existing authorized_keys entries",
		"mode 700",
		"mode 600",
		"no-agent-forwarding,no-port-forwarding,no-X11-forwarding,no-user-rc",
		"present exactly once",
		"ssh-ed25519 " + payload + " workbench-runner",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("enrollment prompt missing %q: %s", want, prompt)
		}
	}
	if strings.Contains(strings.ToLower(prompt), "private key") {
		t.Fatalf("enrollment prompt must contain only public enrollment material: %s", prompt)
	}
}

func TestNormalizeRunnerSSHPublicKeyRejectsWrongType(t *testing.T) {
	payload := base64.StdEncoding.EncodeToString(make([]byte, 48))
	if _, err := normalizeRunnerSSHPublicKey("ssh-rsa " + payload); err == nil {
		t.Fatal("expected non-Ed25519 key to be rejected")
	}
}
