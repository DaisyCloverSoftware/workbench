//go:build windows

package desktop

import (
	"fmt"
	"strings"
	"time"
	"unsafe"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

const (
	idOpsServerHeader = 3600
	idOpsServerList   = 3601
	idOpsCIHeader     = 3602
	idOpsCIList       = 3603
	idOpsWindowsHeader = 3604
	idOpsWindowsList   = 3605
	idOpsAIHeader      = 3606
	idOpsAIList        = 3607
	idOpsWaitingHeader = 3608
	idOpsWaitingList   = 3609
	idOpsNeedsHeader   = 3610
	idOpsNeedsList     = 3611

	idOpsFullHeader   = 3612
	idOpsFullList     = 3613
	idOpsDetails      = 3614
	idOpsPriorityUp   = 3615
	idOpsPriorityDown = 3616
	idOpsOpenTask     = 3617
	idOpsCloseDetails = 3618

	idOpsWorkersLabel = 3619
	idOpsWorkersList  = 3620
	idOpsProjectsLabel = 3621
	idOpsProjectsList  = 3622
	idOpsRecentLabel   = 3623
	idOpsRecentList    = 3624
)

type operationsLaneControl struct {
	Lane   core.WorkLane
	Header int
	List   int
}

var operationsLaneControls = []operationsLaneControl{
	{Lane: core.WorkLaneServerOps, Header: idOpsServerHeader, List: idOpsServerList},
	{Lane: core.WorkLaneCIBuilds, Header: idOpsCIHeader, List: idOpsCIList},
	{Lane: core.WorkLaneWindowsWorkstation, Header: idOpsWindowsHeader, List: idOpsWindowsList},
	{Lane: core.WorkLaneAIWorkers, Header: idOpsAIHeader, List: idOpsAIList},
	{Lane: core.WorkLaneWaiting, Header: idOpsWaitingHeader, List: idOpsWaitingList},
	{Lane: core.WorkLaneNeedsYou, Header: idOpsNeedsHeader, List: idOpsNeedsList},
}

type operationsDashboardUIState struct {
	Surface      DashboardOperationsSurface
	SelectedID   string
	ExpandedLane core.WorkLane
	LaneIDs      map[core.WorkLane][]string
	FullIDs      []string
	RecentIDs    []string
}

var operationsDashboardUI = operationsDashboardUIState{LaneIDs: map[core.WorkLane][]string{}}

func (s *Shell) createOperationsDashboardControls() {
	for _, control := range operationsLaneControls {
		s.control(control.Header, "BUTTON", workLaneTitle(control.Lane), wsChild|wsVisible|wsTabStop|bsPushButton)
		s.control(control.List, "LISTBOX", "", wsChild|wsVisible|wsBorder|wsVScroll|lbsNotify)
		applyDarkExplorerTheme(s.controls[control.List])
	}
	s.control(idOpsFullHeader, "BUTTON", "All operations", wsChild|wsVisible|wsTabStop|bsPushButton)
	s.control(idOpsFullList, "LISTBOX", "", wsChild|wsVisible|wsBorder|wsVScroll|lbsNotify)
	applyDarkExplorerTheme(s.controls[idOpsFullList])
	s.control(idOpsDetails, "EDIT", "", wsChild|wsVisible|wsBorder|wsVScroll|esMultiline|esAutoVScroll|esReadOnly)
	applyDarkExplorerTheme(s.controls[idOpsDetails])
	s.control(idOpsPriorityUp, "BUTTON", "Priority ↑", wsChild|wsVisible|wsTabStop|bsPushButton)
	s.control(idOpsPriorityDown, "BUTTON", "Priority ↓", wsChild|wsVisible|wsTabStop|bsPushButton)
	s.control(idOpsOpenTask, "BUTTON", "Open task", wsChild|wsVisible|wsTabStop|bsPushButton)
	s.control(idOpsCloseDetails, "BUTTON", "Close details", wsChild|wsVisible|wsTabStop|bsPushButton)

	s.static(idOpsWorkersLabel, "Worker assignments")
	s.control(idOpsWorkersList, "LISTBOX", "", wsChild|wsVisible|wsBorder|wsVScroll)
	applyDarkExplorerTheme(s.controls[idOpsWorkersList])
	s.static(idOpsProjectsLabel, "Project activity")
	s.control(idOpsProjectsList, "LISTBOX", "", wsChild|wsVisible|wsBorder|wsVScroll)
	applyDarkExplorerTheme(s.controls[idOpsProjectsList])
	s.static(idOpsRecentLabel, "Recent outcomes")
	s.control(idOpsRecentList, "LISTBOX", "", wsChild|wsVisible|wsBorder|wsVScroll|lbsNotify)
	applyDarkExplorerTheme(s.controls[idOpsRecentList])

	s.hideOperationsDashboardControls()
}

func operationsDashboardControlIDs() []int {
	ids := make([]int, 0, len(operationsLaneControls)*2+13)
	for _, control := range operationsLaneControls {
		ids = append(ids, control.Header, control.List)
	}
	ids = append(ids,
		idOpsFullHeader, idOpsFullList, idOpsDetails, idOpsPriorityUp, idOpsPriorityDown, idOpsOpenTask, idOpsCloseDetails,
		idOpsWorkersLabel, idOpsWorkersList, idOpsProjectsLabel, idOpsProjectsList, idOpsRecentLabel, idOpsRecentList,
	)
	return ids
}

func (s *Shell) hideOperationsDashboardControls() {
	for _, id := range operationsDashboardControlIDs() {
		showWindow(s.controls[id], false)
	}
}

func (s *Shell) refreshOperationsDashboardControls() {
	if s.controls[idOpsDetails] == 0 {
		return
	}
	surface := BuildDashboardOperationsSurface(s.eng)
	operationsDashboardUI.Surface = surface
	if operationsDashboardUI.LaneIDs == nil {
		operationsDashboardUI.LaneIDs = map[core.WorkLane][]string{}
	}
	if operationsDashboardUI.SelectedID != "" {
		if _, ok := surface.Details[operationsDashboardUI.SelectedID]; !ok {
			operationsDashboardUI.SelectedID = ""
		}
	}

	for _, control := range operationsLaneControls {
		items := surface.Live.ByLane[control.Lane]
		ids := make([]string, 0, len(items))
		lines := make([]string, 0, len(items))
		selection := -1
		for i, item := range items {
			ids = append(ids, item.ID)
			lines = append(lines, operationsCompactListLine(item))
			if item.ID == operationsDashboardUI.SelectedID {
				selection = i
			}
		}
		operationsDashboardUI.LaneIDs[control.Lane] = ids
		setOperationsList(s.controls[control.List], lines, selection)
		setWindowText(s.controls[control.Header], fmt.Sprintf("%s · %d  —  Open all", workLaneTitle(control.Lane), len(items)))
	}

	if operationsDashboardUI.ExpandedLane != "" {
		items := surface.Live.ByLane[operationsDashboardUI.ExpandedLane]
		operationsDashboardUI.FullIDs = make([]string, 0, len(items))
		lines := make([]string, 0, len(items))
		selection := -1
		for i, item := range items {
			operationsDashboardUI.FullIDs = append(operationsDashboardUI.FullIDs, item.ID)
			lines = append(lines, operationsExpandedListLine(item))
			if item.ID == operationsDashboardUI.SelectedID {
				selection = i
			}
		}
		setOperationsList(s.controls[idOpsFullList], lines, selection)
		setWindowText(s.controls[idOpsFullHeader], fmt.Sprintf("← All lanes  ·  %s · %d", workLaneTitle(operationsDashboardUI.ExpandedLane), len(items)))
	} else {
		operationsDashboardUI.FullIDs = nil
		setOperationsList(s.controls[idOpsFullList], nil, -1)
	}

	workerLines := make([]string, 0, len(surface.Workers))
	for _, worker := range surface.Workers {
		workerLines = append(workerLines, fmt.Sprintf("%s · %d active · %s", worker.Worker, worker.Running, strings.Join(worker.TaskIDs, ", ")))
	}
	if len(workerLines) == 0 {
		workerLines = append(workerLines, "No worker is currently assigned to running work.")
	}
	setWindowText(s.controls[idOpsWorkersLabel], fmt.Sprintf("Worker assignments · %d running", surface.Live.Running))
	setOperationsList(s.controls[idOpsWorkersList], workerLines, -1)

	projectLines := make([]string, 0, len(surface.Projects))
	for _, project := range surface.Projects {
		projectLines = append(projectLines, fmt.Sprintf("%s · %d run · %d queued · %d waiting · %d need you", project.Name, project.Running, project.Queued, project.Waiting, project.NeedsHuman))
	}
	if len(projectLines) == 0 {
		projectLines = append(projectLines, "No project currently has live work.")
	}
	setWindowText(s.controls[idOpsProjectsLabel], fmt.Sprintf("Project activity · %d live jobs", len(surface.Live.Items)))
	setOperationsList(s.controls[idOpsProjectsList], projectLines, -1)

	operationsDashboardUI.RecentIDs = make([]string, 0, len(surface.RecentOutcomes))
	recentLines := make([]string, 0, len(surface.RecentOutcomes))
	recentSelection := -1
	for i, item := range surface.RecentOutcomes {
		operationsDashboardUI.RecentIDs = append(operationsDashboardUI.RecentIDs, item.ID)
		recentLines = append(recentLines, operationsRecentOutcomeLine(item))
		if item.ID == operationsDashboardUI.SelectedID {
			recentSelection = i
		}
	}
	if len(recentLines) == 0 {
		recentLines = append(recentLines, "No recent completed or failed jobs.")
	}
	setWindowText(s.controls[idOpsRecentLabel], fmt.Sprintf("Recent outcomes · %d", len(surface.RecentOutcomes)))
	setOperationsList(s.controls[idOpsRecentList], recentLines, recentSelection)

	s.refreshOperationsDashboardDetail()
}

func (s *Shell) refreshOperationsDashboardDetail() {
	detail, ok := operationsDashboardUI.Surface.Details[operationsDashboardUI.SelectedID]
	if !ok || operationsDashboardUI.SelectedID == "" {
		setWindowText(s.controls[idOpsDetails], "Select any job to inspect its current operational state.")
		procEnableWindow.Call(s.controls[idOpsPriorityUp], 0)
		procEnableWindow.Call(s.controls[idOpsPriorityDown], 0)
		procEnableWindow.Call(s.controls[idOpsOpenTask], 0)
		return
	}
	setWindowText(s.controls[idOpsDetails], operationsDetailText(detail, time.Now()))
	_, canRaise := higherOperationsPriority(detail.Item.Priority)
	_, canLower := lowerOperationsPriority(detail.Item.Priority)
	procEnableWindow.Call(s.controls[idOpsPriorityUp], boolWord(detail.CanReprioritize && canRaise))
	procEnableWindow.Call(s.controls[idOpsPriorityDown], boolWord(detail.CanReprioritize && canLower))
	procEnableWindow.Call(s.controls[idOpsOpenTask], boolWord(detail.LocalTask))
}

func setOperationsList(hwnd uintptr, lines []string, selection int) {
	if hwnd == 0 {
		return
	}
	procSendMessageW.Call(hwnd, lbResetContent, 0, 0)
	for _, line := range lines {
		ptr := wstr(line)
		procSendMessageW.Call(hwnd, lbAddString, 0, uintptr(unsafe.Pointer(ptr)))
	}
	if selection >= 0 {
		procSendMessageW.Call(hwnd, lbSetCurSel, uintptr(selection), 0)
	}
}

func operationsCompactListLine(item core.WorkItem) string {
	project := operationsProjectLabel(item)
	phase := operationsProgressSummary(item.Progress, item.State)
	if item.State == core.TaskQueued && item.QueuePosition > 0 {
		return fmt.Sprintf("#%d %s · %s — %s", item.QueuePosition, strings.ToUpper(item.Priority.String()), project, item.Title)
	}
	return fmt.Sprintf("%s · %s — %s · %s", strings.ToUpper(dashboardStatusLabel(item.State, string(item.State))), project, item.Title, phase)
}

func operationsExpandedListLine(item core.WorkItem) string {
	line := operationsCompactListLine(item)
	worker := strings.TrimSpace(item.Provider)
	if worker != "" {
		line += " · " + worker
	}
	return line
}

func operationsRecentOutcomeLine(item core.WorkItem) string {
	state := "COMPLETED"
	if item.State == core.TaskFailed {
		state = "FAILED"
	}
	when := ""
	if !item.UpdatedAt.IsZero() {
		when = " · " + item.UpdatedAt.Local().Format("2 Jan 15:04")
	}
	return fmt.Sprintf("[%s] %s — %s%s", state, operationsProjectLabel(item), item.Title, when)
}

func operationsDetailText(detail DashboardOperationsDetail, now time.Time) string {
	var b strings.Builder
	writeOperationsDetailLine(&b, "Project", detail.Project)
	writeOperationsDetailLine(&b, "Task / sprint", detail.Task)
	writeOperationsDetailLine(&b, "Assigned worker", detail.Worker)
	writeOperationsDetailLine(&b, "Current state / stage", strings.TrimSpace(detail.State+" · "+detail.Progress))
	writeOperationsDetailLine(&b, "Start / elapsed", operationsStartAndElapsed(detail.Item, now))
	writeOperationsDetailLine(&b, "Latest meaningful activity", detail.LatestActivity)
	if detail.Item.State == core.TaskQueued {
		queue := fmt.Sprintf("%s priority", detail.Item.Priority.String())
		if detail.Item.QueuePosition > 0 {
			queue += fmt.Sprintf(" · queue #%d", detail.Item.QueuePosition)
		}
		writeOperationsDetailLine(&b, "Queue", queue)
	}
	if detail.WaitReason != "" {
		writeOperationsDetailLine(&b, "Waiting for", detail.WaitReason)
		writeOperationsDetailLine(&b, "Waiting elapsed", operationsElapsedSince(detail.WaitSince, now))
		writeOperationsDetailLine(&b, "Automatic retry / check", detail.AutoContinuation)
	}
	if detail.Failure != "" {
		writeOperationsDetailLine(&b, "Failure", detail.Failure)
	}
	if detail.Reference != "" {
		writeOperationsDetailLine(&b, "CI / build / deployment reference", detail.Reference)
	}
	if detail.OwnerAction != "" {
		writeOperationsDetailLine(&b, "Owner action", detail.OwnerAction)
	}
	return strings.TrimSpace(b.String())
}

func writeOperationsDetailLine(b *strings.Builder, label, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	if b.Len() > 0 {
		b.WriteString("\r\n")
	}
	b.WriteString(label)
	b.WriteString(": ")
	b.WriteString(value)
}

func operationsStartAndElapsed(item core.WorkItem, now time.Time) string {
	start := item.CreatedAt
	label := "Created"
	if item.StartedAt != nil && !item.StartedAt.IsZero() {
		start = *item.StartedAt
		label = "Started"
	}
	if start.IsZero() {
		if !item.UpdatedAt.IsZero() {
			return "Runner start time unavailable · latest state " + item.UpdatedAt.Local().Format("2 Jan 15:04:05")
		}
		return "Start time unavailable"
	}
	probe := item
	probe.CreatedAt = start
	probe.StartedAt = nil
	return fmt.Sprintf("%s %s · elapsed %s", label, start.Local().Format("2 Jan 15:04:05"), workItemElapsed(probe, now))
}

func operationsElapsedSince(start, now time.Time) string {
	if start.IsZero() {
		return "Duration unavailable from runner activity"
	}
	probe := core.WorkItem{CreatedAt: start}
	return workItemElapsed(probe, now)
}

func (s *Shell) layoutOperationsDashboardControls(clientWidth, clientHeight int) {
	s.hideOperationsDashboardControls()
	if s.page != pageDashboard {
		return
	}
	contentX := productionSidebarWidth + 20
	contentY := productionHeaderHeight + 18
	contentW := clientWidth - contentX - 18
	contentH := clientHeight - contentY - 18
	if contentW < 850 || contentH < 620 {
		return
	}

	metricsY := contentY + 68
	metricH := 72
	boardY := metricsY + metricH + 12
	resourceH := 164
	resourceY := contentY + contentH - resourceH
	boardH := resourceY - boardY - 12

	if operationsDashboardUI.ExpandedLane != "" {
		for _, id := range []int{idOpsFullHeader, idOpsFullList, idOpsDetails, idOpsPriorityUp, idOpsPriorityDown, idOpsOpenTask, idOpsCloseDetails} {
			showWindow(s.controls[id], true)
		}
		headerH := 30
		moveWindow(s.controls[idOpsFullHeader], contentX, boardY, contentW, headerH)
		bodyY := boardY + headerH + 6
		bottom := contentY + contentH
		detailW := contentW * 44 / 100
		listW := contentW - detailW - 10
		moveWindow(s.controls[idOpsFullList], contentX, bodyY, listW, bottom-bodyY)
		buttonH := 30
		buttonY := bottom - buttonH
		moveWindow(s.controls[idOpsDetails], contentX+listW+10, bodyY, detailW, buttonY-bodyY-6)
		layoutOperationsActionButtons(s, contentX+listW+10, buttonY, detailW, buttonH)
		return
	}

	colGap := 10
	rowGap := 10
	colW := (contentW - colGap*2) / 3
	rowH := (boardH - rowGap) / 2
	for i, control := range operationsLaneControls {
		col := i % 3
		row := i / 3
		x := contentX + col*(colW+colGap)
		y := boardY + row*(rowH+rowGap)
		showWindow(s.controls[control.Header], true)
		showWindow(s.controls[control.List], true)
		moveWindow(s.controls[control.Header], x+8, y+5, colW-16, 30)
		moveWindow(s.controls[control.List], x+10, y+42, colW-20, rowH-50)
	}

	if operationsDashboardUI.SelectedID != "" {
		for _, id := range []int{idOpsDetails, idOpsPriorityUp, idOpsPriorityDown, idOpsOpenTask, idOpsCloseDetails} {
			showWindow(s.controls[id], true)
		}
		buttonH := 30
		buttonY := resourceY + resourceH - buttonH
		moveWindow(s.controls[idOpsDetails], contentX, resourceY+2, contentW, buttonY-resourceY-8)
		layoutOperationsActionButtons(s, contentX, buttonY, contentW, buttonH)
		return
	}

	for _, id := range []int{idOpsWorkersLabel, idOpsWorkersList, idOpsProjectsLabel, idOpsProjectsList, idOpsRecentLabel, idOpsRecentList} {
		showWindow(s.controls[id], true)
	}
	gap := 10
	panelW := (contentW - gap*2) / 3
	bottomSpecs := []struct{ label, list int }{
		{label: idOpsWorkersLabel, list: idOpsWorkersList},
		{label: idOpsProjectsLabel, list: idOpsProjectsList},
		{label: idOpsRecentLabel, list: idOpsRecentList},
	}
	for i, spec := range bottomSpecs {
		x := contentX + i*(panelW+gap)
		moveWindow(s.controls[spec.label], x+8, resourceY+4, panelW-16, 20)
		moveWindow(s.controls[spec.list], x+8, resourceY+27, panelW-16, resourceH-34)
	}
}

func layoutOperationsActionButtons(s *Shell, x, y, width, height int) {
	gap := 8
	buttonW := (width - gap*3) / 4
	if buttonW < 80 {
		buttonW = 80
	}
	moveWindow(s.controls[idOpsPriorityUp], x, y, buttonW, height)
	moveWindow(s.controls[idOpsPriorityDown], x+buttonW+gap, y, buttonW, height)
	moveWindow(s.controls[idOpsOpenTask], x+(buttonW+gap)*2, y, buttonW, height)
	closeW := width - (buttonW+gap)*3
	if closeW < 80 {
		closeW = 80
	}
	moveWindow(s.controls[idOpsCloseDetails], x+(buttonW+gap)*3, y, closeW, height)
}

func (s *Shell) handleOperationsDashboardCommand(id int, notify uint16) bool {
	if s.page != pageDashboard {
		return false
	}
	for _, control := range operationsLaneControls {
		if id == control.Header {
			operationsDashboardUI.ExpandedLane = control.Lane
			items := operationsDashboardUI.Surface.Live.ByLane[control.Lane]
			if len(items) > 0 {
				found := false
				for _, item := range items {
					if item.ID == operationsDashboardUI.SelectedID {
						found = true
						break
					}
				}
				if !found {
					operationsDashboardUI.SelectedID = items[0].ID
				}
			}
			s.refreshOperationsDashboardControls()
			s.layoutOperationsDashboardForCurrentWindow()
			return true
		}
		if id == control.List && notify == lbnSelChange {
			if selected := operationsSelectedListID(s.controls[control.List], operationsDashboardUI.LaneIDs[control.Lane]); selected != "" {
				operationsDashboardUI.SelectedID = selected
				s.refreshOperationsDashboardControls()
				s.layoutOperationsDashboardForCurrentWindow()
			}
			return true
		}
	}

	switch id {
	case idOpsFullHeader:
		operationsDashboardUI.ExpandedLane = ""
		s.refreshOperationsDashboardControls()
		s.layoutOperationsDashboardForCurrentWindow()
		return true
	case idOpsFullList:
		if notify == lbnSelChange {
			if selected := operationsSelectedListID(s.controls[idOpsFullList], operationsDashboardUI.FullIDs); selected != "" {
				operationsDashboardUI.SelectedID = selected
				s.refreshOperationsDashboardControls()
				s.layoutOperationsDashboardForCurrentWindow()
			}
		}
		return true
	case idOpsRecentList:
		if notify == lbnSelChange {
			if selected := operationsSelectedListID(s.controls[idOpsRecentList], operationsDashboardUI.RecentIDs); selected != "" {
				operationsDashboardUI.SelectedID = selected
				s.refreshOperationsDashboardControls()
				s.layoutOperationsDashboardForCurrentWindow()
			}
		}
		return true
	case idOpsPriorityUp:
		return s.changeOperationsPriority(true)
	case idOpsPriorityDown:
		return s.changeOperationsPriority(false)
	case idOpsOpenTask:
		if !s.openOperationsTask(operationsDashboardUI.SelectedID) {
			messageBox(s.hwnd, "Task unavailable", "This runner-side item does not have a desktop-local Workbench task to open.", mbOK|mbIconInformation)
		}
		return true
	case idOpsCloseDetails:
		operationsDashboardUI.SelectedID = ""
		s.refreshOperationsDashboardControls()
		s.layoutOperationsDashboardForCurrentWindow()
		return true
	}
	return false
}

func operationsSelectedListID(hwnd uintptr, ids []string) string {
	if hwnd == 0 || len(ids) == 0 {
		return ""
	}
	selection, _, _ := procSendMessageW.Call(hwnd, lbGetCurSel, 0, 0)
	index := int(int32(selection))
	if index < 0 || index >= len(ids) {
		return ""
	}
	return ids[index]
}

func (s *Shell) changeOperationsPriority(raise bool) bool {
	detail, ok := operationsDashboardUI.Surface.Details[operationsDashboardUI.SelectedID]
	if !ok || !detail.LocalTask || !detail.CanReprioritize {
		return true
	}
	var next core.WorkPriority
	var change bool
	if raise {
		next, change = higherOperationsPriority(detail.Item.Priority)
	} else {
		next, change = lowerOperationsPriority(detail.Item.Priority)
	}
	if !change {
		return true
	}
	if err := s.eng.SetTaskPriority(detail.Item.ID, next); err != nil {
		messageBox(s.hwnd, "Priority unchanged", err.Error(), mbOK|mbIconWarning)
		return true
	}
	s.refreshOperationsDashboardControls()
	s.layoutOperationsDashboardForCurrentWindow()
	return true
}

func (s *Shell) openOperationsTask(taskID string) bool {
	task, ok := s.eng.Task(strings.TrimSpace(taskID))
	if !ok {
		return false
	}
	if _, err := s.eng.SelectProject(task.ProjectPath); err != nil {
		return false
	}
	s.selectedTaskID = task.ID
	s.editorProjectID = ""
	s.settingsProjectID = ""
	s.page = pageWork
	s.applyPageVisibility()
	s.refresh()
	s.layoutProduction()
	return true
}

func (s *Shell) layoutOperationsDashboardForCurrentWindow() {
	if s.hwnd == 0 {
		return
	}
	var rect nativeRect
	procGetClientRect.Call(s.hwnd, uintptr(unsafe.Pointer(&rect)))
	s.layoutOperationsDashboardControls(int(rect.Right-rect.Left), int(rect.Bottom-rect.Top))
}
