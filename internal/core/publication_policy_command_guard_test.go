package core

import "testing"

func TestSafeCommandGateRejectsPublicationPolicyCLI(t *testing.T) {
	for _, command := range []string{
		"workbench-runner policy get .",
		"workbench-runner policy prepare .",
		"workbench-runner policy publish . https://example.invalid/repo.git",
		"workbench-runner policy delete .",
		"workbench-runner policy-json",
	} {
		if IsSafeCommand(command) {
			t.Fatalf("operator-only publication policy command became model-safe: %q", command)
		}
	}
}
