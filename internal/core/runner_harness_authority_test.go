package core

import "testing"

func TestRunnerHarnessOperatorCommandsAreNeverModelSafe(t *testing.T) {
	for _, command := range []string{
		"workbench-runner harness get",
		"workbench-runner harness set /tmp/adapter",
		"workbench-runner harness delete",
	} {
		if IsSafeCommand(command) {
			t.Fatalf("operator-only runner harness command became model-safe: %q", command)
		}
	}
}
