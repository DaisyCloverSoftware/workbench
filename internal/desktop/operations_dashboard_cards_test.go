package desktop

import (
	"testing"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestBuildDashboardWorkCardsUsesRealMeasuredProgress(t *testing.T) {
	started := time.Date(2026, 8, 20, 8, 0, 0, 0, time.UTC)
	cards := BuildDashboardWorkCards([]core.WorkItem{{
		ID:          "render",
		ProjectName: "Override",
		Title:       "Render frames",
		State:       core.WorkItemRunning,
		Priority:    core.WorkPriorityHigh,
		Location:    core.WorkLocation{Lane: core.WorkLaneWindowsHost, Machine: "Workstation", Tool: "Blender"},
		Progress:    core.WorkProgress{Kind: core.WorkProgressMeasured, Current: 17, Total: 120, Unit: "frames"},
		StartedAt:   &started,
	}}, started.Add(5*time.Minute+7*time.Second))
	if len(cards) != 1 {
		t.Fatalf("cards = %d", len(cards))
	}
	card := cards[0]
	if !card.HasPercent || card.ProgressPercent != 14 {
		t.Fatalf("progress = %d, has=%v", card.ProgressPercent, card.HasPercent)
	}
	if card.ProgressLabel != "17 / 120 frames" || card.LocationLabel != "Workstation · Blender" {
		t.Fatalf("card labels = %#v", card)
	}
	if card.ElapsedLabel != "5m 07s elapsed" {
		t.Fatalf("elapsed = %q", card.ElapsedLabel)
	}
}

func TestBuildDashboardWorkCardsNeverFakesIndeterminatePercent(t *testing.T) {
	cards := BuildDashboardWorkCards([]core.WorkItem{{
		ID:            "deploy",
		Title:         "Deploy public alpha",
		State:         core.WorkItemRunning,
		Priority:      core.WorkPriorityCritical,
		Location:      core.WorkLocation{Lane: core.WorkLaneServerOperations, Executor: "Operations runner"},
		Progress:      core.WorkProgress{Kind: core.WorkProgressIndeterminate, StageName: "Rolling out"},
	}}, time.Now())
	card := cards[0]
	if card.HasPercent || card.ProgressPercent != 0 {
		t.Fatalf("indeterminate work must not expose a percentage: %#v", card)
	}
	if card.ProgressLabel != "Rolling out" || card.PriorityLabel != "CRITICAL" {
		t.Fatalf("labels = %#v", card)
	}
}

func TestBuildDashboardWorkCardsShowsQueuePosition(t *testing.T) {
	cards := BuildDashboardWorkCards([]core.WorkItem{{
		ID:            "queued",
		Title:         "Unreal smoke",
		State:         core.WorkItemQueued,
		Priority:      core.WorkPriorityNormal,
		QueuePosition: 2,
		Location:      core.WorkLocation{Lane: core.WorkLaneWindowsHost, Machine: "Workstation", Tool: "Unreal"},
	}}, time.Now())
	if cards[0].QueueLabel != "Queued #2" {
		t.Fatalf("queue label = %q", cards[0].QueueLabel)
	}
}
