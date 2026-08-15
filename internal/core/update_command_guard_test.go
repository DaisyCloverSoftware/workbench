package core

import "testing"

func TestSafeCommandGateRejectsOperatorUpdateCLI(t *testing.T) {
	for _, command := range []string{
		"workbench-runner update check",
		"workbench-runner update apply",
		"workbench-updater check",
		"workbench-updater apply",
	} {
		if IsSafeCommand(command) {
			t.Fatalf("operator-only maintenance command became model-safe: %q", command)
		}
	}
}
