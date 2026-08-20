package core

import (
	"strings"
	"testing"
)

func TestBlenderVersionFromCapturedOutputAcceptsValidatedVersionWithTruncatedDiagnostics(t *testing.T) {
	stdout := newBoundedWorkerCapture(64)
	_, _ = stdout.Write([]byte("Blender 4.5.3\n"))
	stderr := newBoundedWorkerCapture(16)
	_, _ = stderr.Write([]byte(strings.Repeat("warning\n", 32)))
	if !stderr.Truncated() {
		t.Fatal("test did not produce truncated Blender diagnostics")
	}

	got, err := blenderVersionFromCapturedOutput(stdout, stderr)
	if err != nil {
		t.Fatal(err)
	}
	if got != "Blender 4.5.3" {
		t.Fatalf("unexpected Blender version %q", got)
	}
}

func TestBlenderVersionFromCapturedOutputRejectsTruncatedNoiseWithoutVersion(t *testing.T) {
	stdout := newBoundedWorkerCapture(16)
	_, _ = stdout.Write([]byte(strings.Repeat("noise\n", 32)))
	stderr := newBoundedWorkerCapture(16)
	if !stdout.Truncated() {
		t.Fatal("test did not produce truncated Blender output")
	}

	if _, err := blenderVersionFromCapturedOutput(stdout, stderr); err == nil || !strings.Contains(err.Error(), "exceeded Workbench limits") {
		t.Fatalf("expected bounded-output rejection, got %v", err)
	}
}

func TestBlenderVersionFromCapturedOutputPreservesUntruncatedValidation(t *testing.T) {
	stdout := newBoundedWorkerCapture(128)
	stderr := newBoundedWorkerCapture(128)
	_, _ = stdout.Write([]byte("Blender 4.3.2\n"))

	got, err := blenderVersionFromCapturedOutput(stdout, stderr)
	if err != nil {
		t.Fatal(err)
	}
	if got != "Blender 4.3.2" {
		t.Fatalf("unexpected Blender version %q", got)
	}
}
