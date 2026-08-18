//go:build windows

package desktop

import "github.com/DaisyCloverSoftware/workbench/internal/core"

const (
	idShowArchivedTasks = 3129
	idArchiveTask       = 3130
)

func (s *Shell) createTaskHistoryControls() {
	s.control(idShowArchivedTasks, "BUTTON", "Show archived", wsChild|wsTabStop|bsAutoCheckbox)
	s.control(idArchiveTask, "BUTTON", "Archive", wsChild|wsTabStop|bsPushButton)
	setChecked(s.controls[idShowArchivedTasks], isArchivedTaskHistoryVisible())
}

func (s *Shell) handleTaskHistoryCommand(id int) bool {
	switch id {
	case idShowArchivedTasks:
		setArchivedTaskHistoryVisible(isChecked(s.controls[idShowArchivedTasks]))
		s.refresh()
		s.refreshTaskHistoryControls(BuildSnapshot(s.eng, s.selectedTaskID))
		return true
	case idArchiveTask:
		s.toggleSelectedTaskArchive()
		return true
	default:
		return false
	}
}

func (s *Shell) refreshTaskHistoryControls(snapshot Snapshot) {
	showArchived := s.controls[idShowArchivedTasks]
	archive := s.controls[idArchiveTask]
	if showArchived == 0 || archive == 0 {
		return
	}
	setChecked(showArchived, isArchivedTaskHistoryVisible())
	item, ok := snapshot.SelectedTask()
	if !ok {
		setWindowText(archive, "Archive")
		procEnableWindow.Call(archive, 0)
		return
	}
	if item.Archived {
		setWindowText(archive, "Restore")
		procEnableWindow.Call(archive, 1)
		return
	}
	setWindowText(archive, "Archive")
	procEnableWindow.Call(archive, boolWord(item.Terminal && core.TaskCanArchive(item.Status)))
}

func (s *Shell) toggleSelectedTaskArchive() {
	task, ok := s.selectedTask()
	if !ok {
		return
	}
	target := !task.Archived
	if err := s.eng.SetTaskArchived(task.ID, target); err != nil {
		messageBox(s.hwnd, "Task history", err.Error(), mbOK|mbIconWarning)
		return
	}
	if target && !isArchivedTaskHistoryVisible() {
		s.selectedTaskID = ""
	}
	s.refresh()
	s.refreshTaskHistoryControls(BuildSnapshot(s.eng, s.selectedTaskID))
}

// jumpToLatestVisibleReview mirrors the dashboard review shortcut but respects
// the default archived-history filter. Filing a completed review means it no
// longer steals the global Review & Publish action; the user can still reveal
// it explicitly with Show archived and restore it.
func (s *Shell) jumpToLatestVisibleReview() bool {
	for _, project := range s.eng.Projects() {
		for _, task := range s.eng.TasksForProject(project.ID) {
			if task.Archived || task.Status != core.TaskCompleted || task.Review == nil || !task.Review.Changed {
				continue
			}
			if _, err := s.eng.SelectProject(project.Path); err != nil {
				return false
			}
			s.selectedTaskID = task.ID
			s.editorProjectID = ""
			s.settingsProjectID = ""
			return true
		}
	}
	return false
}
