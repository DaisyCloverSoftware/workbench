package desktop

import "testing"

func TestCanRecoverInterruptedTasksRequiresNamedMutexOwnership(t *testing.T) {
	if CanRecoverInterruptedTasks(false) {
		t.Fatal("desktop recovery was allowed without the per-user ownership mutex")
	}
	if !CanRecoverInterruptedTasks(true) {
		t.Fatal("desktop recovery was refused despite confirmed per-user mutex ownership")
	}
}
