package core

import "testing"

func TestRecoverInterruptedTasksOnlyQueuesNonTerminalExecutionStates(t *testing.T) {
	st := DefaultState()
	st.Tasks = []Task{
		{ID: "queued", Status: TaskQueued, ProviderID: "worker"},
		{ID: "routing", Status: TaskRouting, ProviderID: "worker"},
		{ID: "running", Status: TaskRunning, ProviderID: "worker", ConsumesWork: true},
		{ID: "attention", Status: TaskNeedsAttention, AttentionQuestion: "choose"},
		{ID: "done", Status: TaskCompleted},
		{ID: "failed", Status: TaskFailed},
		{ID: "cancelled", Status: TaskCancelled},
	}
	ids := recoverInterruptedTasks(&st)
	if len(ids) != 3 || ids[0] != "queued" || ids[1] != "routing" || ids[2] != "running" {
		t.Fatalf("unexpected recovered ids: %#v", ids)
	}
	for i := 0; i < 3; i++ {
		if st.Tasks[i].Status != TaskQueued || st.Tasks[i].ProviderID != "" || st.Tasks[i].ConsumesWork || st.Tasks[i].FinishedAt != nil {
			t.Fatalf("task was not reset for routing: %#v", st.Tasks[i])
		}
	}
	if len(st.Tasks[1].Attempts) != 1 || len(st.Tasks[2].Attempts) != 1 {
		t.Fatal("interrupted routing/running tasks should record a recovery attempt")
	}
	if st.Tasks[3].Status != TaskNeedsAttention || st.Tasks[3].AttentionQuestion != "choose" {
		t.Fatal("attention task changed during recovery")
	}
	for _, i := range []int{4, 5, 6} {
		if st.Tasks[i].Status == TaskQueued {
			t.Fatalf("terminal task was reopened: %#v", st.Tasks[i])
		}
	}
}
