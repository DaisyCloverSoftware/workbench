package desktop

import (
	"fmt"
	"testing"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestDashboardKeepsNineActiveChatProjectsAndOrdersByRecency(t *testing.T) {
	now := time.Date(2026, 8, 18, 15, 30, 0, 0, time.UTC)
	activity := make([]core.RunnerChatActivityInfo, 0, 9)
	for i := 0; i < 9; i++ {
		activity = append(activity, core.RunnerChatActivityInfo{
			ID:          fmt.Sprintf("event_%d_12345678", i),
			ProjectRef:  fmt.Sprintf("runner://project-%d", i),
			Action:      "read_file",
			State:       "completed",
			UpdatedAt:   now.Add(-time.Duration(i) * time.Minute),
			Active:      true,
			ActiveKnown: true,
		})
	}

	got := applyRunnerChatActivity(DashboardSnapshot{}, activity, now)
	if got.Summary.Active != 9 {
		t.Fatalf("active summary=%d want 9", got.Summary.Active)
	}
	if len(got.ActiveTasks) != 9 {
		t.Fatalf("active tasks=%d want 9: %#v", len(got.ActiveTasks), got.ActiveTasks)
	}
	if got.ActiveTasks[0].Title != "project-0" || got.ActiveTasks[8].Title != "project-8" {
		t.Fatalf("active tasks not ordered by recency: %#v", got.ActiveTasks)
	}
	for i := 1; i < len(got.ActiveTasks); i++ {
		if got.ActiveTasks[i].UpdatedAt.After(got.ActiveTasks[i-1].UpdatedAt) {
			t.Fatalf("active task %d is newer than the item before it: %#v", i, got.ActiveTasks)
		}
	}
}

func TestDashboardActiveTaskWindowReservesHiddenIndicator(t *testing.T) {
	visible, hidden := dashboardActiveTaskWindow(9, 3*dashboardActiveTaskRowHeight+dashboardActiveTaskFooterHeight)
	if visible != 3 || hidden != 6 {
		t.Fatalf("nine-task window=(%d,%d) want (3,6)", visible, hidden)
	}

	visible, hidden = dashboardActiveTaskWindow(3, 3*dashboardActiveTaskRowHeight)
	if visible != 3 || hidden != 0 {
		t.Fatalf("three-task window=(%d,%d) want (3,0)", visible, hidden)
	}
}

func TestDashboardActiveTasksMixDurableAndChatWorkByRecency(t *testing.T) {
	now := time.Date(2026, 8, 18, 15, 30, 0, 0, time.UTC)
	snapshot := DashboardSnapshot{ActiveTasks: []DashboardTaskItem{
		{TaskID: "durable-old", Title: "durable-old", UpdatedAt: now.Add(-10 * time.Minute)},
		{TaskID: "durable-new", Title: "durable-new", UpdatedAt: now.Add(-time.Minute)},
	}}
	activity := []core.RunnerChatActivityInfo{{
		ID:          "chat_12345678",
		ProjectRef:  "runner://chat-middle",
		Action:      "read_file",
		State:       "completed",
		UpdatedAt:   now.Add(-5 * time.Minute),
		Active:      true,
		ActiveKnown: true,
	}}

	got := applyRunnerChatActivity(snapshot, activity, now)
	if len(got.ActiveTasks) != 3 {
		t.Fatalf("active tasks=%#v", got.ActiveTasks)
	}
	want := []string{"durable-new", "chat-middle", "durable-old"}
	for i, title := range want {
		if got.ActiveTasks[i].Title != title {
			t.Fatalf("active order=%#v want=%#v", got.ActiveTasks, want)
		}
	}
}
