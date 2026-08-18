package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestProductionWorkPageOwnsReversibleTaskHistoryControls(t *testing.T) {
	checks := map[string][]string{
		"task_history_windows.go": {
			`idShowArchivedTasks = 3129`,
			`idArchiveTask       = 3130`,
			`"Show archived"`,
			`"Restore"`,
			`s.eng.SetTaskArchived(task.ID, target)`,
			`func (s *Shell) jumpToLatestVisibleReview() bool`,
			`if task.Archived || task.Status != core.TaskCompleted`,
		},
		"production_shell_windows.go": {
			`s.createTaskHistoryControls()`,
			`s.handleTaskHistoryCommand(id)`,
			`showWindow(s.controls[idShowArchivedTasks], s.page == pageWork)`,
			`showWindow(s.controls[idArchiveTask], s.page == pageWork)`,
			`s.jumpToLatestVisibleReview()`,
		},
		"production_layout_windows.go": {
			`moveWindow(s.controls[idShowArchivedTasks]`,
			`moveWindow(s.controls[idArchiveTask]`,
		},
		"production_buttons_windows.go": {
			`idDelegate, idArchiveTask, idCancelTask`,
		},
		"production_controls_windows.go": {
			`idShowArchivedTasks, idProtectWork`,
		},
	}
	for path, wants := range checks {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(b)
		for _, want := range wants {
			if !strings.Contains(text, want) {
				t.Fatalf("%s missing task-history contract %q", path, want)
			}
		}
	}
}
