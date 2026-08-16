package core

import "testing"

func TestOpenClawCloudRunnerControlsAreNotModelSafeCommands(t *testing.T) {
	for _, command := range []string{
		"workbench-runner agent --message hello --headless",
		"workbench-runner cloud-models",
	} {
		if IsSafeCommand(command) {
			t.Fatalf("runner cloud-model controls must remain outside model-safe commands: %q", command)
		}
	}
}
