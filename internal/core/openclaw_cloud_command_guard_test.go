package core

import "testing"

func TestOpenClawCloudRunnerControlsAreNotModelSafeCommands(t *testing.T) {
	for _, command := range []string{
		"workbench-runner agent --message hello --headless",
		"workbench-runner cloud-models",
		"workbench-runner cloud-model-set openai/gpt-5.6-sol",
	} {
		if IsSafeCommand(command) {
			t.Fatalf("runner cloud-model controls must remain outside model-safe commands: %q", command)
		}
	}
}
