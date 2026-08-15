//go:build windows

package desktop

// refreshProductionPage materialises only the controls that are currently
// visible. The legacy shell refresh path rebuilds Work and Settings together;
// doing that from the production shell means hidden native listboxes are reset
// and repopulated on unrelated navigation and engine notifications. Keeping the
// work scoped to the visible page prevents that hidden-control message backlog
// while preserving the same durable engine state.
func (s *Shell) refreshProductionPage() {
	snapshot := BuildSnapshot(s.eng, s.selectedTaskID)
	if snapshot.SelectedTaskID != "" {
		s.selectedTaskID = snapshot.SelectedTaskID
	} else if s.selectedTaskID != "" {
		if _, ok := s.eng.Task(s.selectedTaskID); !ok {
			s.selectedTaskID = ""
		}
	}

	s.refreshGlobalStatus()
	switch s.page {
	case pageWork:
		s.refreshProjects(snapshot)
		s.refreshTasks(snapshot)
	case pageSettings:
		s.refreshSettings(snapshot)
	}
}
