package desktop

import (
	"strings"
	"testing"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestVisibleTaskHistoryHidesArchivedByDefault(t *testing.T) {
	now := time.Now().UTC()
	tasks := []core.Task{
		{ID: "live-task", Status: core.TaskRunning, UpdatedAt: now},
		{ID: "ready-task", Status: core.TaskCompleted, UpdatedAt: now.Add(-time.Minute)},
		{ID: "archived-task", Status: core.TaskCompleted, Archived: true, UpdatedAt: now.Add(-time.Hour)},
	}
	visible := visibleTaskHistory(tasks, false)
	if len(visible) != 2 || visible[0].ID != "live-task" || visible[1].ID != "ready-task" {
		t.Fatalf("default task history=%+v", visible)
	}
	all := visibleTaskHistory(tasks, true)
	if len(all) != 3 || all[2].ID != "archived-task" {
		t.Fatalf("expanded task history=%+v", all)
	}
	all[0].ID = "mutated-copy"
	if tasks[0].ID != "live-task" {
		t.Fatal("history filter must not alias the engine task slice")
	}
}

func TestArchivedTaskPresentationIsExplicitAndRestorable(t *testing.T) {
	items := taskItems([]core.Task{{
		ID:       "archived-task",
		Title:    "Finished work",
		Intent:   "finish work",
		Status:   core.TaskCompleted,
		Archived: true,
	}})
	if len(items) != 1 || !items[0].Archived || !items[0].Terminal {
		t.Fatalf("archived task item=%+v", items)
	}
	if !strings.HasPrefix(items[0].StatusLabel, "Archived · ") {
		t.Fatalf("archived status label=%q", items[0].StatusLabel)
	}
}

func TestArchivedHistoryVisibilityIsAnExplicitViewToggle(t *testing.T) {
	setArchivedTaskHistoryVisible(false)
	t.Cleanup(func() { setArchivedTaskHistoryVisible(false) })
	if isArchivedTaskHistoryVisible() {
		t.Fatal("archived history should start hidden")
	}
	setArchivedTaskHistoryVisible(true)
	if !isArchivedTaskHistoryVisible() {
		t.Fatal("explicit history toggle did not enable archived records")
	}
}
