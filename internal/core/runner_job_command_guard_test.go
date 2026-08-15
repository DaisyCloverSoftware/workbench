package core

import "testing"

func TestRunnerJobControlsAreNotModelSafeCommands(t *testing.T) {
	for _, command := range []string{
		"workbench-runner job submit",
		"workbench-runner job status task-123",
		"workbench-runner job cancel task-123",
		"workbench-runner job-execute task-123",
	} {
		if IsSafeCommand(command) {
			t.Fatalf("durable runner control must remain outside model-safe commands: %q", command)
		}
	}
}
