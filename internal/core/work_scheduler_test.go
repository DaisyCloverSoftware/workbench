package core

import (
	"testing"
	"time"
)

func TestPlanWorkDispatchDoesNotPreemptRunningWork(t *testing.T) {
	base := time.Date(2026, 8, 20, 8, 0, 0, 0, time.UTC)
	items := []WorkItem{
		{ID: "running", State: WorkItemRunning, Priority: WorkPriorityNormal, CreatedAt: base, Location: WorkLocation{Lane: WorkLaneServerOperations}},
		{ID: "critical", State: WorkItemQueued, Priority: WorkPriorityCritical, CreatedAt: base.Add(time.Minute), Location: WorkLocation{Lane: WorkLaneServerOperations}},
	}
	plan := PlanWorkDispatch(items, []WorkLaneCapacity{{Lane: WorkLaneServerOperations, Capacity: 1}})
	if len(plan.Dispatch) != 0 {
		t.Fatalf("running work must retain its slot; dispatch = %#v", plan.Dispatch)
	}
	if plan.Available[WorkLaneServerOperations] != 0 {
		t.Fatalf("available server slots = %d, want 0", plan.Available[WorkLaneServerOperations])
	}
}

func TestPlanWorkDispatchStartsHighestPriorityNext(t *testing.T) {
	base := time.Date(2026, 8, 20, 8, 0, 0, 0, time.UTC)
	items := []WorkItem{
		{ID: "running", State: WorkItemRunning, CreatedAt: base, Location: WorkLocation{Lane: WorkLaneServerOperations}},
		{ID: "normal", State: WorkItemQueued, Priority: WorkPriorityNormal, CreatedAt: base.Add(time.Minute), Location: WorkLocation{Lane: WorkLaneServerOperations}},
		{ID: "critical", State: WorkItemQueued, Priority: WorkPriorityCritical, CreatedAt: base.Add(2 * time.Minute), Location: WorkLocation{Lane: WorkLaneServerOperations}},
	}
	plan := PlanWorkDispatch(items, []WorkLaneCapacity{{Lane: WorkLaneServerOperations, Capacity: 2}})
	if len(plan.Dispatch) != 1 || plan.Dispatch[0].ID != "critical" {
		t.Fatalf("dispatch = %#v, want critical only", plan.Dispatch)
	}
}

func TestPlanWorkDispatchTreatsLanesIndependently(t *testing.T) {
	base := time.Date(2026, 8, 20, 8, 0, 0, 0, time.UTC)
	items := []WorkItem{
		{ID: "ops", State: WorkItemQueued, Priority: WorkPriorityHigh, CreatedAt: base, Location: WorkLocation{Lane: WorkLaneServerOperations}},
		{ID: "windows", State: WorkItemQueued, Priority: WorkPriorityNormal, CreatedAt: base, Location: WorkLocation{Lane: WorkLaneWindowsHost}},
		{ID: "waiting", State: WorkItemWaiting, Priority: WorkPriorityCritical, CreatedAt: base, Location: WorkLocation{Lane: WorkLaneWaiting}},
	}
	plan := PlanWorkDispatch(items, []WorkLaneCapacity{
		{Lane: WorkLaneServerOperations, Capacity: 1},
		{Lane: WorkLaneWindowsHost, Capacity: 1},
		{Lane: WorkLaneWaiting, Capacity: 0},
	})
	if len(plan.Dispatch) != 2 {
		t.Fatalf("dispatch = %#v, want one server and one Windows job", plan.Dispatch)
	}
	seen := map[string]bool{}
	for _, item := range plan.Dispatch {
		seen[item.ID] = true
	}
	if !seen["ops"] || !seen["windows"] || seen["waiting"] {
		t.Fatalf("dispatch set = %#v", seen)
	}
}
