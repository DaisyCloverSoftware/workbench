package core

import (
	"testing"
	"time"
)

func TestOrderQueuedWorkItemsPrioritisesThenPreservesFIFO(t *testing.T) {
	base := time.Date(2026, 8, 20, 8, 0, 0, 0, time.UTC)
	items := []WorkItem{
		{ID: "normal-old", State: WorkItemQueued, Priority: WorkPriorityNormal, CreatedAt: base},
		{ID: "critical-new", State: WorkItemQueued, Priority: WorkPriorityCritical, CreatedAt: base.Add(2 * time.Minute)},
		{ID: "high-new", State: WorkItemQueued, Priority: WorkPriorityHigh, CreatedAt: base.Add(3 * time.Minute)},
		{ID: "high-old", State: WorkItemQueued, Priority: WorkPriorityHigh, CreatedAt: base.Add(time.Minute)},
		{ID: "running", State: WorkItemRunning, Priority: WorkPriorityCritical, CreatedAt: base.Add(-time.Minute)},
	}

	ordered := OrderQueuedWorkItems(items)
	if len(ordered) != 4 {
		t.Fatalf("got %d queued items, want 4", len(ordered))
	}
	want := []string{"critical-new", "high-old", "high-new", "normal-old"}
	for i, id := range want {
		if ordered[i].ID != id {
			t.Fatalf("position %d = %q, want %q", i+1, ordered[i].ID, id)
		}
		if ordered[i].QueuePosition != i+1 {
			t.Fatalf("%q queue position = %d, want %d", id, ordered[i].QueuePosition, i+1)
		}
	}
}

func TestOrderQueuedWorkItemsDefaultsUnknownPriorityToNormal(t *testing.T) {
	base := time.Date(2026, 8, 20, 8, 0, 0, 0, time.UTC)
	ordered := OrderQueuedWorkItems([]WorkItem{
		{ID: "unknown", State: WorkItemQueued, Priority: "", CreatedAt: base},
		{ID: "low", State: WorkItemQueued, Priority: WorkPriorityLow, CreatedAt: base.Add(-time.Hour)},
	})
	if len(ordered) != 2 {
		t.Fatalf("got %d queued items, want 2", len(ordered))
	}
	if ordered[0].ID != "unknown" || ordered[0].Priority != WorkPriorityNormal {
		t.Fatalf("unknown priority should normalize to normal, got %#v", ordered[0])
	}
}

func TestWorkProgressPercentMeasuredIsBounded(t *testing.T) {
	tests := []struct {
		name string
		p    WorkProgress
		want int
		ok   bool
	}{
		{"half", WorkProgress{Kind: WorkProgressMeasured, Current: 5, Total: 10}, 50, true},
		{"bounded", WorkProgress{Kind: WorkProgressMeasured, Current: 12, Total: 10}, 100, true},
		{"invalid total", WorkProgress{Kind: WorkProgressMeasured, Current: 1, Total: 0}, 0, false},
		{"indeterminate", WorkProgress{Kind: WorkProgressIndeterminate}, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := tt.p.Percent()
			if got != tt.want || ok != tt.ok {
				t.Fatalf("Percent() = (%d, %v), want (%d, %v)", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestWorkProgressPercentStages(t *testing.T) {
	p := WorkProgress{Kind: WorkProgressStages, Stage: 3, StageTotal: 5, StageName: "Rollout"}
	got, ok := p.Percent()
	if !ok || got != 60 {
		t.Fatalf("Percent() = (%d, %v), want (60, true)", got, ok)
	}
}
