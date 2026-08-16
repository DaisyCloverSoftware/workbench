//go:build windows

package desktop

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func (s *Shell) handleCommand(id int, notify uint16) {
	switch id {
	case idNavWork:
		s.page = pageWork
		s.jumpToNeedsAttention()
		s.applyPageVisibility()
		s.refresh()
		s.layout()
		return
	case idNavSettings:
		s.page = pageSettings
		s.applyPageVisibility()
		s.refreshSettings(BuildSnapshot(s.eng, s.selectedTaskID))
		s.layout()
		return
	case idProjectList:
		if notify == lbnSelChange {
			s.selectProjectFromList()
		}
		return
	case idTaskList:
		if notify == lbnSelChange {
			s.selectTaskFromList()
		}
		return
	case idAddProject:
		s.addProject()
		return
	case idRenameProject:
		s.renameActiveProject()
		return
	case idPinProject:
		s.toggleActiveProjectPin()
		return
	case idRemoveProject:
		s.removeActiveProject()
		return
	case idDelegate:
		s.delegateActiveProject()
		return
	case idCancelTask:
		s.cancelSelectedTask()
		return
	case idResumeTask:
		s.resumeSelectedTask()
		return
	case idOpenReview:
		s.openSelectedReview()
		return
	case idRetryReview:
		s.retrySelectedReview()
		return
	case idCopyBranch:
		s.copySelectedReviewBranch()
		return
	case idSaveNotes:
		s.saveActiveProjectNotes()
		return
	}
	s.handleSettingsCommand(id, notify)
}

func (s *Shell) jumpToNeedsAttention() bool {
	target, ok := FirstAttentionTarget(s.eng)
	if !ok {
		return false
	}
	for _, project := range s.eng.Projects() {
		if project.ID != target.ProjectID {
			continue
		}
		if _, err := s.eng.SelectProject(project.Path); err != nil {
			messageBox(s.hwnd, "Cannot open attention task", err.Error(), mbOK|mbIconWarning)
			return false
		}
		s.selectedTaskID = target.TaskID
		s.editorProjectID = ""
		s.settingsProjectID = ""
		return true
	}
	return false
}

func (s *Shell) selectProjectFromList() {
	idx := listSelection(s.controls[idProjectList])
	if idx < 0 || idx >= len(s.projectIDs) {
		return
	}
	projectID := s.projectIDs[idx]
	for _, project := range s.eng.Projects() {
		if project.ID != projectID {
			continue
		}
		if _, err := s.eng.SelectProject(project.Path); err != nil {
			messageBox(s.hwnd, "Cannot open project", err.Error(), mbOK|mbIconWarning)
			return
		}
		s.selectedTaskID = ""
		s.editorProjectID = ""
		s.settingsProjectID = ""
		s.refresh()
		return
	}
}

func (s *Shell) addProject() {
	host := strings.TrimSpace(s.eng.State().Preferences.OpenClawSSHHost)
	if host == "" {
		choice := messageBox(
			s.hwnd,
			"Add project",
			"Use projects that live on a Workbench cluster runner?\r\n\r\nYes — configure the cluster runner first\r\nNo — choose a folder on this PC",
			mbYesNo|mbIconInformation,
		)
		if choice == idYes {
			s.page = pageSettings
			s.applyPageVisibility()
			s.invalidateSettingsCache()
			s.refreshSettings(BuildSnapshot(s.eng, s.selectedTaskID))
			s.layout()
			messageBox(s.hwnd, "Configure cluster runner", "Enter the Workbench Runner SSH host, save routing, then return to Work and click Add project. Workbench will discover the Git repositories on that runner; you do not need to copy them to Windows.", mbOK|mbIconInformation)
			return
		}
	} else {
		choice := messageBox(
			s.hwnd,
			"Add project",
			"Import Git repositories from the configured Workbench cluster runner?\r\n\r\nYes — discover and import cluster projects\r\nNo — choose a folder on this PC",
			mbYesNo|mbIconInformation,
		)
		if choice == idYes {
			s.importClusterProjects(host)
			return
		}
	}

	path := browseFolder(s.hwnd, "Choose a project/repository for Workbench")
	if path == "" {
		return
	}
	project, err := s.eng.SelectProject(path)
	if err != nil {
		messageBox(s.hwnd, "Cannot add project", err.Error(), mbOK|mbIconWarning)
		return
	}
	s.selectedTaskID = ""
	s.editorProjectID = ""
	s.settingsProjectID = ""
	messageBox(s.hwnd, "Project added", project.Name+" is now the active Workbench project.", mbOK|mbIconInformation)
}

func (s *Shell) importClusterProjects(host string) {
	go func() {
		response, err := s.discoverClusterProjects(host)
		if err != nil {
			version, probeErr := core.TestWorkbenchRunnerSSH(host)
			if probeErr == nil && strings.TrimSpace(version) != core.Version {
				question := "The cluster runner reports Workbench " + strings.TrimSpace(version) + ", while this app is Workbench " + core.Version + ".\r\n\r\nCluster project discovery requires the matching runner protocol. Install the latest verified Workbench cluster update on that runner now?"
				if messageBox(s.hwnd, "Update cluster runner", question, mbYesNo|mbIconInformation) == idYes {
					ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
					result, updateErr := core.UpdateWorkbenchRunnerSSH(ctx, host)
					cancel()
					if updateErr != nil {
						messageBox(s.hwnd, "Runner update failed", updateErr.Error(), mbOK|mbIconWarning)
						return
					}
					if result.Applied || !result.UpdateAvailable {
						response, err = s.discoverClusterProjects(host)
					}
				}
			}
		}
		if err != nil {
			messageBox(s.hwnd, "Cluster projects unavailable", "Workbench could not read the configured runner project list. Check the runner connection in Settings, then try again.", mbOK|mbIconWarning)
			return
		}
		if len(response.Projects) == 0 {
			messageBox(s.hwnd, "No cluster projects found", "The configured runner responded, but no Git repositories were found directly under its authorised project root.", mbOK|mbIconInformation)
			return
		}
		added, err := s.eng.RegisterRunnerProjects(response.Projects)
		if err != nil {
			messageBox(s.hwnd, "Cannot import cluster projects", err.Error(), mbOK|mbIconWarning)
			return
		}
		message := fmt.Sprintf("Workbench found %d cluster Git repositories.", len(response.Projects))
		if added > 0 {
			message += fmt.Sprintf(" %d new project(s) were added to the Work list.", added)
		} else {
			message += " They were already registered."
		}
		message += "\r\n\r\nCluster projects stay on the runner; Workbench does not copy them onto this PC. ChatGPT safe read/search/patch/test operations use the same bounded runner transport."
		messageBox(s.hwnd, "Cluster projects ready", message, mbOK|mbIconInformation)
	}()
}

func (s *Shell) discoverClusterProjects(host string) (core.RunnerToolResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return core.RunRunnerToolSSH(ctx, host, core.RunnerToolRequest{Action: "list_projects"})
}

func (s *Shell) renameActiveProject() {
	project, ok := s.eng.ActiveProject()
	if !ok {
		messageBox(s.hwnd, "Project", "Add or select a project first.", mbOK|mbIconInformation)
		return
	}
	name := strings.TrimSpace(windowText(s.controls[idProjectName]))
	if err := s.eng.RenameProject(project.ID, name); err != nil {
		messageBox(s.hwnd, "Cannot rename project", err.Error(), mbOK|mbIconWarning)
		return
	}
	s.editorProjectID = ""
}

func (s *Shell) toggleActiveProjectPin() {
	project, ok := s.eng.ActiveProject()
	if !ok {
		messageBox(s.hwnd, "Project", "Add or select a project first.", mbOK|mbIconInformation)
		return
	}
	if err := s.eng.SetProjectPinned(project.ID, !project.Pinned); err != nil {
		messageBox(s.hwnd, "Cannot update project", err.Error(), mbOK|mbIconWarning)
	}
}

func (s *Shell) removeActiveProject() {
	project, ok := s.eng.ActiveProject()
	if !ok {
		return
	}
	body := fmt.Sprintf("Remove %s from the Workbench project list?\r\n\r\nThis does not delete the repository or its task history.", project.Name)
	if messageBox(s.hwnd, "Remove project", body, mbYesNo|mbIconWarning) != idYes {
		return
	}
	if err := s.eng.RemoveProject(project.ID); err != nil {
		messageBox(s.hwnd, "Cannot remove project", err.Error(), mbOK|mbIconWarning)
		return
	}
	s.selectedTaskID = ""
	s.editorProjectID = ""
	s.settingsProjectID = ""
}

func (s *Shell) delegateActiveProject() {
	project, ok := s.eng.ActiveProject()
	if !ok {
		messageBox(s.hwnd, "Choose a project", "Add or select the project Workbench should work in first.", mbOK|mbIconInformation)
		return
	}
	intent := strings.TrimSpace(windowText(s.controls[idIntent]))
	task, err := s.eng.Delegate("desktop", intent, project.Path)
	if err != nil {
		messageBox(s.hwnd, "Cannot delegate", err.Error(), mbOK|mbIconWarning)
		return
	}
	s.selectedTaskID = task.ID
	setWindowText(s.controls[idIntent], "")
}

func (s *Shell) selectTaskFromList() {
	idx := listSelection(s.controls[idTaskList])
	if idx < 0 || idx >= len(s.taskIDs) {
		return
	}
	s.selectedTaskID = s.taskIDs[idx]
	s.refreshTasks(BuildSnapshot(s.eng, s.selectedTaskID))
}

func (s *Shell) selectedTask() (core.Task, bool) {
	id := strings.TrimSpace(s.selectedTaskID)
	if id == "" {
		return core.Task{}, false
	}
	return s.eng.Task(id)
}

func (s *Shell) cancelSelectedTask() {
	task, ok := s.selectedTask()
	if !ok {
		return
	}
	if err := s.eng.Cancel(task.ID); err != nil {
		messageBox(s.hwnd, "Cannot cancel task", err.Error(), mbOK|mbIconWarning)
	}
}

func (s *Shell) resumeSelectedTask() {
	task, ok := s.selectedTask()
	if !ok {
		return
	}
	answer := strings.TrimSpace(windowText(s.controls[idAnswer]))
	if err := s.eng.ResolveAttention(task.ID, answer); err != nil {
		messageBox(s.hwnd, "Cannot resume task", err.Error(), mbOK|mbIconWarning)
		return
	}
	setWindowText(s.controls[idAnswer], "")
}

func (s *Shell) openSelectedReview() {
	task, ok := s.selectedTask()
	if !ok {
		return
	}
	url := core.ReviewPullRequestURL(task)
	if url == "" {
		messageBox(s.hwnd, "Review unavailable", "This task does not currently have a verified GitHub pull request to open.", mbOK|mbIconInformation)
		return
	}
	if err := openExternal(s.hwnd, url); err != nil {
		messageBox(s.hwnd, "Cannot open review", err.Error(), mbOK|mbIconWarning)
	}
}

func (s *Shell) retrySelectedReview() {
	task, ok := s.selectedTask()
	if !ok {
		return
	}
	procEnableWindow.Call(s.controls[idRetryReview], 0)
	go func(taskID string) {
		err := s.eng.RetryTaskReviewDelivery(taskID)
		if err != nil {
			messageBox(s.hwnd, "Review delivery", err.Error()+"\r\n\r\nThe completed code is preserved. Coding will not run again.", mbOK|mbIconWarning)
			return
		}
		messageBox(s.hwnd, "Review ready", "Review delivery completed. Coding did not run again.", mbOK|mbIconInformation)
	}(task.ID)
}

func (s *Shell) copySelectedReviewBranch() {
	task, ok := s.selectedTask()
	if !ok || task.Review == nil || strings.TrimSpace(task.Review.Branch) == "" {
		return
	}
	if err := copyText(s.hwnd, task.Review.Branch); err != nil {
		messageBox(s.hwnd, "Clipboard", err.Error(), mbOK|mbIconWarning)
		return
	}
	messageBox(s.hwnd, "Copied", "Review branch copied to the clipboard.", mbOK|mbIconInformation)
}

func (s *Shell) saveActiveProjectNotes() {
	project, ok := s.eng.ActiveProject()
	if !ok {
		messageBox(s.hwnd, "Project notes", "Add or select a project first.", mbOK|mbIconInformation)
		return
	}
	notes := windowText(s.controls[idNotes])
	if core.LooksSecret(notes) {
		messageBox(s.hwnd, "Secret detected", "Workbench refused to save these notes because they look like they contain a credential. Store the value under Settings → Vault and keep only a vault:// reference here.", mbOK|mbIconWarning)
		return
	}
	if err := s.eng.SaveNotes(project.Path, notes); err != nil {
		messageBox(s.hwnd, "Cannot save notes", err.Error(), mbOK|mbIconWarning)
		return
	}
	s.editorProjectID = ""
	messageBox(s.hwnd, "Project context saved", "These notes are now scoped to "+project.Name+".", mbOK|mbIconInformation)
}
