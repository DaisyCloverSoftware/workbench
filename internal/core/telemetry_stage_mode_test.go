package core

import "testing"

func TestUnmeasuredStageNamesReflectTaskMode(t *testing.T) {
	dev := TaskProgress(Task{Status: TaskRunning, Mode: TaskModeDevelopment, Progress: WorkProgress{Kind: ProgressIndeterminate}})
	ops := TaskProgress(Task{Status: TaskRunning, Mode: TaskModeOperations, Progress: WorkProgress{Kind: ProgressIndeterminate}})
	if dev.Phase == ops.Phase || dev.Phase == "" || ops.Phase == "" {
		t.Fatalf("dev=%#v ops=%#v", dev, ops)
	}
}
