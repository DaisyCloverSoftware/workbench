package core

import "testing"

func TestReorderQueuedTasksPersistsExactOrder(t *testing.T) {
	store, err := NewStoreAt(t.TempDir() + "/state.json")
	if err != nil {
		t.Fatal(err)
	}
	eng, err := NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	projectPath := t.TempDir()
	project, err := eng.SelectProject(projectPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := eng.SetProjectPriority(project.ID, WorkPriorityNormal); err != nil {
		t.Fatal(err)
	}
	eng.mu.Lock()
	eng.state.Tasks = append(eng.state.Tasks,
		Task{ID: "task-a", Title: "A", ProjectPath: projectPath, Status: TaskQueued},
		Task{ID: "task-b", Title: "B", ProjectPath: projectPath, Status: TaskQueued},
		Task{ID: "task-c", Title: "C", ProjectPath: projectPath, Status: TaskQueued},
	)
	eng.mu.Unlock()

	if err := eng.ReorderQueuedTasks(WorkLaneAIWorkers, WorkPriorityNormal, []string{"task-c", "task-a", "task-b"}); err != nil {
		t.Fatal(err)
	}
	items := WorkItemsFromTasks(eng.State().Tasks, eng.Projects())
	positions := map[string]int{}
	for _, item := range items {
		positions[item.ID] = item.QueuePosition
	}
	if positions["task-c"] != 1 || positions["task-a"] != 2 || positions["task-b"] != 3 {
		t.Fatalf("positions = %#v", positions)
	}
}

func TestReorderQueuedTasksRejectsStalePartialQueue(t *testing.T) {
	store, err := NewStoreAt(t.TempDir() + "/state.json")
	if err != nil {
		t.Fatal(err)
	}
	eng, err := NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	projectPath := t.TempDir()
	if _, err := eng.SelectProject(projectPath); err != nil {
		t.Fatal(err)
	}
	eng.mu.Lock()
	eng.state.Tasks = append(eng.state.Tasks,
		Task{ID: "task-a", Title: "A", ProjectPath: projectPath, Status: TaskQueued},
		Task{ID: "task-b", Title: "B", ProjectPath: projectPath, Status: TaskQueued},
	)
	eng.mu.Unlock()

	if err := eng.ReorderQueuedTasks(WorkLaneAIWorkers, WorkPriorityNormal, []string{"task-a"}); err == nil {
		t.Fatal("stale partial queue reorder should be rejected")
	}
}

func TestManualQueueRankOnlyReordersInsideSamePriority(t *testing.T) {
	items := AssignLaneQueuePositions([]WorkItem{
		{ID: "normal-ranked", State: WorkItemQueued, Priority: WorkPriorityNormal, QueueRank: 1000, Location: WorkLocation{Lane: WorkLaneServerOperations}},
		{ID: "critical-unranked", State: WorkItemQueued, Priority: WorkPriorityCritical, Location: WorkLocation{Lane: WorkLaneServerOperations}},
	})
	positions := map[string]int{}
	for _, item := range items {
		positions[item.ID] = item.QueuePosition
	}
	if positions["critical-unranked"] != 1 || positions["normal-ranked"] != 2 {
		t.Fatalf("priority must still win over manual rank: %#v", positions)
	}
}
