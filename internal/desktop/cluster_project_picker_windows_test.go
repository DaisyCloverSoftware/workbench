//go:build windows

package desktop

import (
	"testing"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestRunnerProjectSelectionChoosesOneRepository(t *testing.T) {
	projects := []core.RunnerProjectInfo{
		{Name: "garage", Ref: "runner://garage"},
		{Name: "workbench", Ref: "runner://workbench"},
	}
	got := runnerProjectSelection(projects, 1, false)
	if len(got) != 1 || got[0].Ref != "runner://workbench" {
		t.Fatalf("selection=%+v", got)
	}
}

func TestRunnerProjectSelectionCanChooseAll(t *testing.T) {
	projects := []core.RunnerProjectInfo{
		{Name: "garage", Ref: "runner://garage"},
		{Name: "workbench", Ref: "runner://workbench"},
	}
	got := runnerProjectSelection(projects, -1, true)
	if len(got) != len(projects) {
		t.Fatalf("selection=%+v", got)
	}
	got[0].Name = "changed"
	if projects[0].Name == "changed" {
		t.Fatal("selection must copy project slice")
	}
}

func TestRunnerProjectSelectionRejectsInvalidIndex(t *testing.T) {
	projects := []core.RunnerProjectInfo{{Name: "garage", Ref: "runner://garage"}}
	if got := runnerProjectSelection(projects, 4, false); got != nil {
		t.Fatalf("invalid selection=%+v", got)
	}
}
