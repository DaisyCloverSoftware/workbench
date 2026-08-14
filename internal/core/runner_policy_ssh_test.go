package core

import (
	"context"
	"strings"
	"testing"
)

func TestValidateSSHHostTarget(t *testing.T) {
	got, err := validateSSHHostTarget("  operator@runner.example  ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "operator@runner.example" {
		t.Fatalf("host=%q", got)
	}
	for _, bad := range []string{
		"",
		"-oProxyCommand=evil",
		"runner.example other",
		"runner.example\nother",
		"runner.example\tother",
	} {
		if _, err := validateSSHHostTarget(bad); err == nil {
			t.Fatalf("unsafe SSH host accepted: %q", bad)
		}
	}
}

func TestClusterRunnerRejectsUnsafeSSHHostBeforeExecution(t *testing.T) {
	result, err := RunClusterRunnerSSH(context.Background(), "-oProxyCommand=evil", Task{ProjectPath: "ignored"}, Preferences{})
	if err == nil || !result.Retryable {
		t.Fatalf("unsafe SSH host was not rejected as retryable: result=%#v err=%v", result, err)
	}
	if !strings.Contains(err.Error(), "unsafe") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenClawConnectionTestRejectsUnsafeSSHHostBeforeExecution(t *testing.T) {
	if _, err := TestOpenClawSSH("-oProxyCommand=evil"); err == nil {
		t.Fatal("unsafe OpenClaw SSH host was not rejected")
	} else if !strings.Contains(err.Error(), "unsafe") {
		t.Fatalf("unexpected error: %v", err)
	}
}
