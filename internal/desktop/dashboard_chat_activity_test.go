package desktop

import (
	"testing"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestRunnerChatActivityKeepsNormalChatWorkActiveAcrossUnattendedBlock(t *testing.T) {
	now := time.Date(2026, 8, 18, 13, 45, 0, 0, time.UTC)
	snapshot := DashboardSnapshot{
		Projects: []DashboardProjectItem{{ID: "garage", Name: "garage", Path: "runner://garage"}},
	}
	got := applyRunnerChatActivity(snapshot, []core.RunnerChatActivityInfo{{
		ID:         "read_12345678",
		ProjectRef: "runner://garage",
		Action:     "read_file",
		State:      "completed",
		UpdatedAt:  now.Add(-2 * time.Hour),
	}}, now)
	if got.Summary.Active != 1 || got.Projects[0].Summary.Active != 1 {
		t.Fatalf("chat session activity was not counted: %#v", got)
	}
	if len(got.ActiveTasks) != 1 || got.ActiveTasks[0].Provider != "ChatGPT via Workbench" {
		t.Fatalf("chat session missing from active work: %#v", got.ActiveTasks)
	}
	if len(got.RecentActivity) != 1 || got.RecentActivity[0].Detail != "ChatGPT via Workbench · read source" {
		t.Fatalf("chat activity missing from recent feed: %#v", got.RecentActivity)
	}
}

func TestRunnerChatActivityExpiresInsteadOfPretendingOldChatsAreActive(t *testing.T) {
	now := time.Date(2026, 8, 18, 13, 45, 0, 0, time.UTC)
	got := applyRunnerChatActivity(DashboardSnapshot{}, []core.RunnerChatActivityInfo{{
		ID:         "old_12345678",
		ProjectRef: "runner://garage",
		Action:     "read_file",
		State:      "completed",
		UpdatedAt:  now.Add(-5 * time.Hour),
	}}, now)
	if got.Summary.Active != 0 || len(got.ActiveTasks) != 0 {
		t.Fatalf("stale chat activity remained active: %#v", got)
	}
	if len(got.RecentActivity) != 1 {
		t.Fatalf("old-but-recent activity should remain in the feed: %#v", got.RecentActivity)
	}
}

func TestCompletedAutonomousTaskIsNotKeptActiveByChatLease(t *testing.T) {
	now := time.Date(2026, 8, 18, 13, 45, 0, 0, time.UTC)
	got := applyRunnerChatActivity(DashboardSnapshot{}, []core.RunnerChatActivityInfo{{
		ID:         "task_12345678",
		ProjectRef: "runner://garage",
		Action:     "delegate_task",
		State:      "completed",
		UpdatedAt:  now.Add(-2 * time.Minute),
	}}, now)
	if got.Summary.Active != 0 || len(got.ActiveTasks) != 0 {
		t.Fatalf("completed autonomous task remained active: %#v", got)
	}
}

func TestRunningAutonomousTaskRemainsActiveFromRealState(t *testing.T) {
	now := time.Date(2026, 8, 18, 13, 45, 0, 0, time.UTC)
	got := applyRunnerChatActivity(DashboardSnapshot{}, []core.RunnerChatActivityInfo{{
		ID:         "task_12345678",
		ProjectRef: "runner://garage",
		Action:     "delegate_task",
		State:      "running",
		UpdatedAt:  now.Add(-8 * time.Hour),
	}}, now)
	if got.Summary.Active != 1 || len(got.ActiveTasks) != 1 {
		t.Fatalf("running autonomous task was expired despite real running state: %#v", got)
	}
}
