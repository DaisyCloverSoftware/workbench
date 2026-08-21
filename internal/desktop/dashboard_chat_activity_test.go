package desktop

import (
	"testing"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestRunnerChatActivityKeepsNormalChatSessionPresenceAcrossUnattendedBlock(t *testing.T) {
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
		t.Fatalf("chat session presence was not counted: %#v", got)
	}
	if len(got.ActiveTasks) != 1 || got.ActiveTasks[0].Provider != "ChatGPT via Workbench" || got.ActiveTasks[0].StatusLabel != "Session active" {
		t.Fatalf("chat session presence missing or not labelled separately: %#v", got.ActiveTasks)
	}
	if len(got.RecentActivity) != 1 || got.RecentActivity[0].Detail != "ChatGPT via Workbench · read source" {
		t.Fatalf("chat activity missing from recent feed: %#v", got.RecentActivity)
	}
	if got.RecentActivity[0].Status != core.TaskCompleted || got.RecentActivity[0].StatusLabel != "Completed" {
		t.Fatalf("completed operation was relabelled by session presence: %#v", got.RecentActivity)
	}
}

func TestRunnerAuthoritativeActiveSessionDoesNotRewriteCompletedEvent(t *testing.T) {
	runnerObserved := time.Date(2026, 8, 18, 14, 29, 0, 0, time.UTC)
	eventTime := time.Date(2026, 8, 18, 11, 33, 31, 0, time.UTC)
	got := applyRunnerChatActivity(DashboardSnapshot{}, []core.RunnerChatActivityInfo{{
		ID:          "read_12345678",
		ProjectRef:  "runner://override",
		Action:      "read_file",
		State:       "completed",
		UpdatedAt:   eventTime,
		Active:      true,
		ActiveKnown: true,
	}}, runnerObserved.Add(10*time.Hour))
	if got.Summary.Active != 1 || len(got.ActiveTasks) != 1 || got.ActiveTasks[0].StatusLabel != "Session active" {
		t.Fatalf("runner-authoritative session presence was not retained: %#v", got)
	}
	if len(got.RecentActivity) != 1 || got.RecentActivity[0].Status != core.TaskCompleted || got.RecentActivity[0].StatusLabel != "Completed" {
		t.Fatalf("runner-authoritative session rewrote terminal history: %#v", got.RecentActivity)
	}
}

func TestRunnerAuthoritativeActiveSessionDoesNotRewriteFailedEvent(t *testing.T) {
	now := time.Date(2026, 8, 18, 14, 29, 0, 0, time.UTC)
	got := applyRunnerChatActivity(DashboardSnapshot{}, []core.RunnerChatActivityInfo{{
		ID:          "failed_12345678",
		ProjectRef:  "runner://override",
		Action:      "run_safe_command",
		State:       "failed",
		UpdatedAt:   now.Add(-time.Minute),
		Active:      true,
		ActiveKnown: true,
	}}, now)
	if got.Summary.Active != 1 || len(got.ActiveTasks) != 1 || got.ActiveTasks[0].StatusLabel != "Session active" {
		t.Fatalf("active session presence disappeared after failed operation: %#v", got)
	}
	if len(got.RecentActivity) != 1 || got.RecentActivity[0].Status != core.TaskFailed || got.RecentActivity[0].StatusLabel != "Failed" {
		t.Fatalf("failed operation was relabelled by session presence: %#v", got.RecentActivity)
	}
}

func TestRunnerAuthoritativeInactiveStateWinsOverDesktopLease(t *testing.T) {
	now := time.Date(2026, 8, 18, 14, 29, 0, 0, time.UTC)
	got := applyRunnerChatActivity(DashboardSnapshot{}, []core.RunnerChatActivityInfo{{
		ID:          "done_12345678",
		ProjectRef:  "runner://override",
		Action:      "read_file",
		State:       "completed",
		UpdatedAt:   now.Add(-time.Minute),
		Active:      false,
		ActiveKnown: true,
	}}, now)
	if got.Summary.Active != 0 || len(got.ActiveTasks) != 0 {
		t.Fatalf("runner-authoritative inactive session remained present: %#v", got)
	}
	if len(got.RecentActivity) != 1 || got.RecentActivity[0].Status != core.TaskCompleted {
		t.Fatalf("completed history disappeared with inactive session: %#v", got.RecentActivity)
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
		t.Fatalf("stale chat session remained active: %#v", got)
	}
	if len(got.RecentActivity) != 1 || got.RecentActivity[0].Status != core.TaskCompleted {
		t.Fatalf("old-but-recent completed activity should remain terminal in the feed: %#v", got.RecentActivity)
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
	if len(got.RecentActivity) != 1 || got.RecentActivity[0].Status != core.TaskCompleted {
		t.Fatalf("completed autonomous history was not terminal: %#v", got.RecentActivity)
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
		t.Fatalf("running autonomous task/session was expired despite real running state: %#v", got)
	}
	if len(got.RecentActivity) != 1 || got.RecentActivity[0].Status != core.TaskRunning || got.RecentActivity[0].StatusLabel != "Working" {
		t.Fatalf("running autonomous activity lost real running state: %#v", got.RecentActivity)
	}
}
