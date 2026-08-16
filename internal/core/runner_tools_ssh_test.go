package core

import (
	"context"
	"errors"
	"os/exec"
	"testing"
)

func TestClassifyRunnerToolSSHFailureAuthentication(t *testing.T) {
	err := classifyRunnerToolSSHFailure("Permission denied (publickey).", errors.New("exit status 255"), nil)
	if !errors.Is(err, ErrRunnerSSHAuthentication) {
		t.Fatalf("error=%v want unattended SSH authentication category", err)
	}
}

func TestClassifyRunnerToolSSHFailureMissingClient(t *testing.T) {
	err := classifyRunnerToolSSHFailure("", exec.ErrNotFound, nil)
	if !errors.Is(err, ErrRunnerSSHClientUnavailable) {
		t.Fatalf("error=%v want missing Windows SSH client category", err)
	}
}

func TestClassifyRunnerToolSSHFailureRemoteRunnerMissing(t *testing.T) {
	err := classifyRunnerToolSSHFailure("remote: .local/bin/workbench-runner: not found", errors.New("exit status 127"), nil)
	if !errors.Is(err, ErrRunnerExecutableUnavailable) {
		t.Fatalf("error=%v want missing runner executable category", err)
	}
}

func TestClassifyRunnerToolSSHFailureTimeout(t *testing.T) {
	err := classifyRunnerToolSSHFailure("", context.DeadlineExceeded, context.DeadlineExceeded)
	if !errors.Is(err, ErrRunnerSSHConnectionTimeout) {
		t.Fatalf("error=%v want timeout category", err)
	}
}
