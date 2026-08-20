//go:build windows

package desktop

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

const (
	pageDashboard shellPage = 2

	idNavDashboard = 3301
	idTopNewTask   = 3302
	idTopNeedsYou  = 3303
	idTopReview    = 3304

	productionSidebarWidth = 184
	productionHeaderHeight = 72
)

func (s *Shell) createDashboardChrome() {
	s.control(idNavDashboard, "BUTTON", "Dashboard", wsChild|wsVisible|wsTabStop|bsOwnerDraw)
	s.control(idTopNewTask, "BUTTON", "+ Delegate Task", wsChild|wsVisible|wsTabStop|bsOwnerDraw)
	s.control(idTopNeedsYou, "BUTTON", "Needs you", wsChild|wsVisible|wsTabStop|bsOwnerDraw)
	s.control(idTopReview, "BUTTON", "Review & Publish", wsChild|wsVisible|wsTabStop|bsOwnerDraw)
}

func (s *Shell) layoutProductionChrome(width int) {
	pad := 16
	moveWindow(s.controls[idBrand], pad+8, 14, productionSidebarWidth-24, 24)
	moveWindow(s.controls[idGlobalStatus], pad+8, 40, productionSidebarWidth-24, 20)
	moveWindow(s.controls[idNavDashboard], pad, productionHeaderHeight+12, productionSidebarWidth-2*pad, 38)
	moveWindow(s.controls[idNavWork], pad, productionHeaderHeight+58, productionSidebarWidth-2*pad, 38)
	moveWindow(s.controls[idNavSettings], pad, productionHeaderHeight+104, productionSidebarWidth-2*pad, 38)

	buttonY := 18
	right := width - 18
	moveWindow(s.controls[idTopReview], right-154, buttonY, 154, 36)
	moveWindow(s.controls[idTopNeedsYou], right-272, buttonY, 108, 36)
	moveWindow(s.controls[idTopNewTask], right-418, buttonY, 136, 36)
}

func (s *Shell) handleDashboardCommand(id int) bool {
	switch id {
	case idNavDashboard:
		s.page = pageDashboard
		s.applyPageVisibility()
		s.refresh()
		s.layout()
		invalidateWindow(s.hwnd)
		return true
	case idTopNewTask:
		s.page = pageWork
		s.applyPageVisibility()
		s.refresh()
		s.layout()
		focusWindow(s.controls[idIntent])
		return true
	case idTopNeedsYou:
		if !s.jumpToNeedsAttention() {
			messageBox(s.hwnd, "Nothing needs you", "Workbench has no task waiting for a human decision.", mbOK|mbIconInformation)
			return true
		}
		s.page = pageWork
		s.applyPageVisibility()
		s.refresh()
		s.layout()
		return true
	case idTopReview:
		if !s.jumpToLatestReview() {
			messageBox(s.hwnd, "No review waiting", "There is no completed Workbench review artifact waiting to be opened or delivered.", mbOK|mbIconInformation)
			return true
		}
		s.page = pageWork
		s.applyPageVisibility()
		s.refresh()
		s.layout()
		return true
	}
	return false
}

func (s *Shell) jumpToLatestReview() bool {
	for _, project := range s.eng.Projects() {
		for _, task := range s.eng.TasksForProject(project.ID) {
			if task.Status != core.TaskCompleted || task.Review == nil || !task.Review.Changed {
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

func (s *Shell) paintProductionWindow() uintptr {
	var ps paintStruct
	hdc, _, _ := procBeginPaint.Call(s.hwnd, uintptr(unsafe.Pointer(&ps)))
	if hdc == 0 {
		return 0
	}
	defer procEndPaint.Call(s.hwnd, uintptr(unsafe.Pointer(&ps)))

	var client nativeRect
	procGetClientRect.Call(s.hwnd, uintptr(unsafe.Pointer(&client)))
	fillRectColor(hdc, client, productionPalette.Background)
	fillRectColor(hdc, nativeRect{Left: 0, Top: 0, Right: productionSidebarWidth, Bottom: client.Bottom}, productionPalette.Sidebar)
	fillRectColor(hdc, nativeRect{Left: productionSidebarWidth, Top: 0, Right: client.Right, Bottom: productionHeaderHeight}, productionPalette.Header)
	fillRectColor(hdc, nativeRect{Left: productionSidebarWidth, Top: productionHeaderHeight - 1, Right: client.Right, Bottom: productionHeaderHeight}, productionPalette.Border)
	fillRectColor(hdc, nativeRect{Left: productionSidebarWidth - 1, Top: 0, Right: productionSidebarWidth, Bottom: client.Bottom}, productionPalette.Border)

	if s.page == pageDashboard {
		s.paintDashboard(hdc, client)
	}
	return 0
}

func (s *Shell) paintDashboard(hdc uintptr, client nativeRect) {
	d := BuildDashboardSnapshot(s.eng)
	host := strings.TrimSpace(s.eng.State().Preferences.OpenClawSSHHost)
	if host != "" {
		s.ensureRunnerProviderInventory(false)
		d = applyRunnerProviderDashboard(d, runnerProviderInventory(host))
	}
	ops := BuildDashboardOperationsSnapshot(s.eng)

	contentX := productionSidebarWidth + 20
	contentY := productionHeaderHeight + 18
	contentW := int(client.Right) - contentX - 18
	contentH := int(client.Bottom) - contentY - 18
	if contentW < 850 || contentH < 620 {
		return
	}

	drawTextStyled(hdc, "Operations", rectWH(contentX, contentY, contentW, 32), productionPalette.Text, 27, fwBold, dtLeft|dtSingleLine|dtVCenter)
	drawTextStyled(hdc, "What is running, where it is running, and what is waiting", rectWH(contentX, contentY+34, contentW, 22), productionPalette.Muted, 14, fwNormal, dtLeft|dtSingleLine|dtVCenter)

	healthText := "ChatGPT primary ready"
	healthColor := productionPalette.Green
	if s.mcpURL == "" {
		healthText = "ChatGPT bridge offline"
		healthColor = productionPalette.Amber
	}
	pill := rectWH(int(client.Right)-196, 78, 176, 30)
	roundedPanel(hdc, pill, productionPalette.Panel, healthColor, 12)
	drawTextStyled(hdc, "●  "+healthText, insetRect(pill, 10, 0), healthColor, 12, fwSemiBold, dtLeft|dtSingleLine|dtVCenter)

	metricsY := contentY + 68
	metricGap := 10
	metricH := 72
	metricW := (contentW - metricGap*4) / 5
	metrics := []struct {
		label, value, detail string
		accent               uint32
	}{
		{"Running", fmt.Sprintf("%d", ops.Running), "executing now", productionPalette.Green},
		{"Queued", fmt.Sprintf("%d", ops.Queued), "scheduler waiting", productionPalette.Accent},
		{"Waiting", fmt.Sprintf("%d", ops.Waiting), "dependency or retry", productionPalette.Purple},
		{"Needs you", fmt.Sprintf("%d", ops.NeedsHuman), "human decisions", productionPalette.Amber},
		{"Worker capacity", fmt.Sprintf("%d / %d", d.ProviderReady, d.ProviderTotal), "autonomous ready", productionPalette.Teal},
	}
	for i, metric := range metrics {
		rect := rectWH(contentX+i*(metricW+metricGap), metricsY, metricW, metricH)
		drawMetricCard(hdc, rect, metric.label, metric.value, metric.detail, metric.accent)
	}

	boardY := metricsY + metricH + 12
	resourceH := 164
	resourceY := contentY + contentH - resourceH
	boardH := resourceY - boardY - 12
	colGap := 10
	rowGap := 10
	colW := (contentW - colGap*2) / 3
	rowH := (boardH - rowGap) / 2
	lanes := []core.WorkLane{
		core.WorkLaneServerOps,
		core.WorkLaneCIBuilds,
		core.WorkLaneWindowsWorkstation,
		core.WorkLaneAIWorkers,
		core.WorkLaneWaiting,
		core.WorkLaneNeedsYou,
	}
	for i, lane := range lanes {
		col := i % 3
		row := i / 3
		rect := rectWH(contentX+col*(colW+colGap), boardY+row*(rowH+rowGap), colW, rowH)
		drawOperationsLane(hdc, rect, lane, ops.ByLane[lane])
	}

	resourcesW := (contentW * 58) / 100
	resourcesRect := rectWH(contentX, resourceY, resourcesW-6, resourceH)
	projectsRect := rectWH(contentX+resourcesW+6, resourceY, contentW-resourcesW-6, resourceH)
	drawOperationsResources(hdc, resourcesRect, d.Providers)
	drawOperationsProjects(hdc, projectsRect, d.Projects)
}

func drawOperationsLane(hdc uintptr, rect nativeRect, lane core.WorkLane, items []core.WorkItem) {
	body := drawPanelFrame(hdc, rect, fmt.Sprintf("%s · %d", workLaneTitle(lane), len(items)))
	if len(items) == 0 {
		drawTextStyled(hdc, "No tracked work in this lane.", body, productionPalette.Muted, 11, fwNormal, dtLeft|dtWordBreak)
		return
	}
	rowH := 72
	max := int(body.Bottom-body.Top) / rowH
	if max < 1 {
		max = 1
	}
	if max > 2 {
		max = 2
	}
	if max > len(items) {
		max = len(items)
	}
	for i := 0; i < max; i++ {
		drawWorkItemCard(hdc, rectWH(int(body.Left), int(body.Top)+i*rowH, int(body.Right-body.Left), rowH-6), items[i])
	}
	if len(items) > max {
		drawTextStyled(hdc, fmt.Sprintf("+%d more", len(items)-max), rectWH(int(body.Left), int(body.Bottom)-18, int(body.Right-body.Left), 16), productionPalette.Muted, 9, fwSemiBold, dtRight|dtSingleLine|dtVCenter)
	}
}

func drawWorkItemCard(hdc uintptr, rect nativeRect, item core.WorkItem) {
	fillRectColor(hdc, nativeRect{Left: rect.Left, Top: rect.Bottom - 1, Right: rect.Right, Bottom: rect.Bottom}, productionPalette.Border)
	status := workStateLabel(item)
	color := workStateColor(item)
	project := strings.TrimSpace(item.ProjectName)
	if project == "" {
		project = "Workbench"
	}
	title := project + " — " + strings.TrimSpace(item.Title)
	drawTextStyled(hdc, title, rectWH(int(rect.Left), int(rect.Top), int(rect.Right-rect.Left)-96, 20), productionPalette.Text, 11, fwSemiBold, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis)
	drawTextStyled(hdc, status, rectWH(int(rect.Right)-94, int(rect.Top), 92, 20), color, 9, fwSemiBold, dtRight|dtSingleLine|dtVCenter|dtEndEllipsis)

	executor := strings.TrimSpace(item.Provider)
	if executor == "" {
		executor = "Workbench"
	}
	location := executor
	if strings.TrimSpace(item.Machine) != "" {
		location += " → " + item.Machine
	}
	drawTextStyled(hdc, location, rectWH(int(rect.Left), int(rect.Top)+21, int(rect.Right-rect.Left)-80, 16), productionPalette.Muted, 9, fwNormal, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis)
	drawTextStyled(hdc, workItemElapsed(item, time.Now()), rectWH(int(rect.Right)-76, int(rect.Top)+21, 74, 16), productionPalette.Muted, 9, fwNormal, dtRight|dtSingleLine|dtVCenter)

	detail := strings.TrimSpace(item.Dependency)
	if detail == "" {
		detail = strings.TrimSpace(item.Progress.Phase)
	}
	if item.QueuePosition > 0 {
		if detail != "" {
			detail += " · "
		}
		detail += fmt.Sprintf("queue #%d", item.QueuePosition)
	}
	if detail == "" {
		detail = "Progress is indeterminate; Workbench will not invent a percentage."
	}
	drawTextStyled(hdc, detail, rectWH(int(rect.Left), int(rect.Top)+40, int(rect.Right-rect.Left), 17), productionPalette.Muted, 9, fwNormal, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis)
}

func drawOperationsResources(hdc uintptr, rect nativeRect, providers []DashboardProviderItem) {
	body := drawPanelFrame(hdc, rect, fmt.Sprintf("Execution capacity · %d", len(providers)))
	if len(providers) == 0 {
		drawTextStyled(hdc, "No autonomous execution capacity detected.", body, productionPalette.Muted, 11, fwNormal, dtLeft|dtWordBreak)
		return
	}
	cols := 2
	colGap := 16
	colW := (int(body.Right-body.Left) - colGap) / cols
	max := len(providers)
	if max > 6 {
		max = 6
	}
	for i := 0; i < max; i++ {
		provider := providers[i]
		col := i % cols
		row := i / cols
		x := int(body.Left) + col*(colW+colGap)
		y := int(body.Top) + row*27
		color := productionPalette.Muted
		status := "Unavailable"
		if provider.Ready {
			color = productionPalette.Green
			status = "Ready"
		}
		if provider.Cooling {
			color = productionPalette.Purple
			status = "Cooling"
		}
		roundedPanel(hdc, rectWH(x, y+8, 7, 7), color, color, 7)
		drawTextStyled(hdc, provider.Name, rectWH(x+14, y, colW-94, 23), productionPalette.Text, 10, fwNormal, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis)
		drawTextStyled(hdc, status, rectWH(x+colW-76, y, 76, 23), color, 9, fwSemiBold, dtRight|dtSingleLine|dtVCenter)
	}
}

func drawOperationsProjects(hdc uintptr, rect nativeRect, projects []DashboardProjectItem) {
	active := make([]DashboardProjectItem, 0, len(projects))
	for _, project := range projects {
		if project.Summary.Active > 0 || project.Summary.NeedsHuman > 0 {
			active = append(active, project)
		}
	}
	body := drawPanelFrame(hdc, rect, fmt.Sprintf("Projects with work · %d", len(active)))
	if len(active) == 0 {
		drawTextStyled(hdc, fmt.Sprintf("%d registered projects · none currently active", len(projects)), body, productionPalette.Muted, 11, fwNormal, dtLeft|dtWordBreak)
		return
	}
	max := len(active)
	if max > 3 {
		max = 3
	}
	for i := 0; i < max; i++ {
		project := active[i]
		y := int(body.Top) + i*27
		drawTextStyled(hdc, project.Name, rectWH(int(body.Left), y, int(body.Right-body.Left)-120, 22), productionPalette.Text, 10, fwSemiBold, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis)
		detail := fmt.Sprintf("%d active · %d need you", project.Summary.Active, project.Summary.NeedsHuman)
		drawTextStyled(hdc, detail, rectWH(int(body.Right)-116, y, 114, 22), productionPalette.Muted, 9, fwNormal, dtRight|dtSingleLine|dtVCenter)
	}
}

func workLaneTitle(lane core.WorkLane) string {
	switch lane {
	case core.WorkLaneServerOps:
		return "Server Operations"
	case core.WorkLaneCIBuilds:
		return "CI / Builds"
	case core.WorkLaneWindowsWorkstation:
		return "Windows Workstation"
	case core.WorkLaneAIWorkers:
		return "AI Workers"
	case core.WorkLaneWaiting:
		return "Waiting"
	case core.WorkLaneNeedsYou:
		return "Needs You"
	default:
		return "Work"
	}
}

func workStateLabel(item core.WorkItem) string {
	if item.State == core.TaskQueued && item.QueuePosition > 0 {
		return fmt.Sprintf("%s · QUEUED #%d", strings.ToUpper(item.Priority.String()), item.QueuePosition)
	}
	return strings.ToUpper(dashboardStatusLabel(item.State, string(item.State)))
}

func workStateColor(item core.WorkItem) uint32 {
	if item.State == core.TaskNeedsAttention {
		return productionPalette.Amber
	}
	if item.State == core.TaskFailed {
		return productionPalette.Red
	}
	if item.State == core.TaskWaitingDependency || item.State == core.TaskWaitingRetry {
		return productionPalette.Purple
	}
	if item.State == core.TaskRunning || item.State == core.TaskRouting {
		return productionPalette.Green
	}
	return productionPalette.Accent
}

func drawMetricCard(hdc uintptr, rect nativeRect, label, value, detail string, accent uint32) {
	roundedPanel(hdc, rect, productionPalette.Panel, productionPalette.Border, 12)
	fillRectColor(hdc, nativeRect{Left: rect.Left, Top: rect.Top, Right: rect.Left + 4, Bottom: rect.Bottom}, accent)
	drawTextStyled(hdc, label, nativeRect{Left: rect.Left + 15, Top: rect.Top + 8, Right: rect.Right - 10, Bottom: rect.Top + 27}, productionPalette.Muted, 11, fwSemiBold, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis)
	drawTextStyled(hdc, value, nativeRect{Left: rect.Left + 15, Top: rect.Top + 25, Right: rect.Right - 10, Bottom: rect.Top + 51}, productionPalette.Text, 21, fwBold, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis)
	drawTextStyled(hdc, detail, nativeRect{Left: rect.Left + 15, Top: rect.Top + 50, Right: rect.Right - 10, Bottom: rect.Bottom - 5}, productionPalette.Muted, 9, fwNormal, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis)
}

func drawPanelFrame(hdc uintptr, rect nativeRect, title string) nativeRect {
	roundedPanel(hdc, rect, productionPalette.Panel, productionPalette.Border, 12)
	drawTextStyled(hdc, title, nativeRect{Left: rect.Left + 14, Top: rect.Top + 8, Right: rect.Right - 14, Bottom: rect.Top + 32}, productionPalette.Text, 14, fwSemiBold, dtLeft|dtSingleLine|dtVCenter)
	fillRectColor(hdc, nativeRect{Left: rect.Left + 1, Top: rect.Top + 38, Right: rect.Right - 1, Bottom: rect.Top + 39}, productionPalette.Border)
	return nativeRect{Left: rect.Left + 12, Top: rect.Top + 46, Right: rect.Right - 12, Bottom: rect.Bottom - 8}
}

func drawActivityPanel(hdc uintptr, rect nativeRect, items []DashboardActivityItem) {
	body := drawPanelFrame(hdc, rect, "Recent activity")
	if len(items) == 0 {
		drawTextStyled(hdc, "No durable task activity yet.", body, productionPalette.Muted, 12, fwNormal, dtLeft|dtWordBreak)
		return
	}
	rowH := 48
	for i, item := range items {
		if i >= 6 || body.Top+int32((i+1)*rowH) > body.Bottom {
			break
		}
		y := int(body.Top) + i*rowH
		color := statusColor(item.StatusLabel)
		roundedPanel(hdc, rectWH(int(body.Left), y+7, 8, 8), color, color, 8)
		drawTextStyled(hdc, item.Title, rectWH(int(body.Left)+17, y, int(body.Right-body.Left)-90, 22), productionPalette.Text, 12, fwSemiBold, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis)
		drawTextStyled(hdc, item.StatusLabel, rectWH(int(body.Right)-80, y, 78, 22), color, 11, fwSemiBold, dtRight|dtSingleLine|dtVCenter|dtEndEllipsis)
		detail := strings.ReplaceAll(strings.TrimSpace(item.Detail), "\n", " ")
		drawTextStyled(hdc, detail, rectWH(int(body.Left)+17, y+21, int(body.Right-body.Left)-85, 20), productionPalette.Muted, 10, fwNormal, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis)
		drawTextStyled(hdc, activityTime(item.UpdatedAt), rectWH(int(body.Right)-72, y+22, 70, 18), productionPalette.Muted, 10, fwNormal, dtRight|dtSingleLine|dtVCenter)
	}
}

func drawActiveTasksPanel(hdc uintptr, rect nativeRect, tasks []DashboardTaskItem) {
	body := drawPanelFrame(hdc, rect, fmt.Sprintf("Active tasks · %d", len(tasks)))
	if len(tasks) == 0 {
		drawTextStyled(hdc, "Nothing is currently running or waiting.", body, productionPalette.Muted, 12, fwNormal, dtLeft|dtWordBreak)
		return
	}
	visible, hidden := dashboardActiveTaskWindow(len(tasks), int(body.Bottom-body.Top))
	for i := 0; i < visible; i++ {
		task := tasks[i]
		y := int(body.Top) + i*dashboardActiveTaskRowHeight
		color := statusColor(task.StatusLabel)
		drawTextStyled(hdc, task.Title, rectWH(int(body.Left), y, int(body.Right-body.Left)-90, 19), productionPalette.Text, 11, fwSemiBold, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis)
		drawTextStyled(hdc, task.StatusLabel, rectWH(int(body.Right)-86, y, 84, 19), color, 10, fwSemiBold, dtRight|dtSingleLine|dtVCenter|dtEndEllipsis)
		provider := strings.TrimSpace(task.Provider)
		if provider == "" {
			provider = "Router"
		}
		drawTextStyled(hdc, provider, rectWH(int(body.Left), y+19, 128, 15), productionPalette.Muted, 9, fwNormal, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis)
		next := task.NextAction
		if task.RetryAt != nil {
			next = "Retries " + task.RetryAt.Local().Format("15:04")
		}
		drawTextStyled(hdc, next, rectWH(int(body.Left)+134, y+19, int(body.Right-body.Left)-136, 15), productionPalette.Muted, 9, fwNormal, dtRight|dtSingleLine|dtVCenter|dtEndEllipsis)
	}
	if hidden > 0 {
		footerTop := int(body.Bottom) - dashboardActiveTaskFooterHeight
		fillRectColor(hdc, nativeRect{Left: body.Left, Top: int32(footerTop), Right: body.Right, Bottom: int32(footerTop + 1)}, productionPalette.Border)
		footer := fmt.Sprintf("+%d more active · %d of %d shown", hidden, visible, len(tasks))
		drawTextStyled(hdc, footer, rectWH(int(body.Left), footerTop+2, int(body.Right-body.Left), dashboardActiveTaskFooterHeight-2), productionPalette.Muted, 9, fwSemiBold, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis)
	}
}

func (s *Shell) drawSystemPanel(hdc uintptr, rect nativeRect, d DashboardSnapshot) {
	body := drawPanelFrame(hdc, rect, "System status")
	rows := []struct {
		name, status string
		ok           bool
	}{
		{"ChatGPT brain", ternary(s.mcpURL != "", "Ready", "Bridge offline"), s.mcpURL != ""},
		{"Runner", ternary(d.RunnerConfigured, "Configured", "Local only"), true},
		{"Durable state", "Ready", true},
		{"Verified updater", ternary(updaterInstalled(), "Installed", "Not beside app"), updaterInstalled()},
	}
	for i, row := range rows {
		y := int(body.Top) + i*38
		color := productionPalette.Green
		if !row.ok {
			color = productionPalette.Amber
		}
		roundedPanel(hdc, rectWH(int(body.Left), y+10, 8, 8), color, color, 8)
		drawTextStyled(hdc, row.name, rectWH(int(body.Left)+17, y, 112, 28), productionPalette.Text, 11, fwNormal, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis)
		drawTextStyled(hdc, row.status, rectWH(int(body.Right)-92, y, 90, 28), color, 10, fwSemiBold, dtRight|dtSingleLine|dtVCenter|dtEndEllipsis)
	}
	if d.Summary.NeedsHuman > 0 {
		alertRect := rectWH(int(body.Left), int(body.Bottom)-54, int(body.Right-body.Left), 46)
		roundedPanel(hdc, alertRect, rgb(55, 42, 20), productionPalette.Amber, 10)
		drawTextStyled(hdc, fmt.Sprintf("%d task(s) need a decision", d.Summary.NeedsHuman), insetRect(alertRect, 10, 0), productionPalette.Amber, 11, fwSemiBold, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis)
	}
}

func drawProjectsPanel(hdc uintptr, rect nativeRect, projects []DashboardProjectItem) {
	body := drawPanelFrame(hdc, rect, fmt.Sprintf("Projects · %d", len(projects)))
	if len(projects) == 0 {
		drawTextStyled(hdc, "Add a repository from Work to start.", body, productionPalette.Muted, 12, fwNormal, dtLeft|dtWordBreak)
		return
	}
	visible, hidden := dashboardProjectWindow(len(projects), int(body.Bottom-body.Top))
	for i := 0; i < visible; i++ {
		project := projects[i]
		y := int(body.Top) + i*dashboardProjectRowHeight
		name := project.Name
		if project.Pinned {
			name = "★ " + name
		}
		drawTextStyled(hdc, name, rectWH(int(body.Left), y, int(body.Right-body.Left)-185, 19), productionPalette.Text, 11, fwSemiBold, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis)
		detail := fmt.Sprintf("%d active  ·  %d need you  ·  %d ready", project.Summary.Active, project.Summary.NeedsHuman, project.Summary.Completed)
		drawTextStyled(hdc, detail, rectWH(int(body.Right)-180, y, 178, 19), productionPalette.Muted, 9, fwNormal, dtRight|dtSingleLine|dtVCenter|dtEndEllipsis)
		drawTextStyled(hdc, project.Path, rectWH(int(body.Left), y+19, int(body.Right-body.Left), 15), productionPalette.Muted, 9, fwNormal, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis)
	}
	if hidden > 0 {
		footerTop := int(body.Bottom) - dashboardProjectFooterHeight
		fillRectColor(hdc, nativeRect{Left: body.Left, Top: int32(footerTop), Right: body.Right, Bottom: int32(footerTop + 1)}, productionPalette.Border)
		footer := fmt.Sprintf("+%d more projects · %d of %d shown", hidden, visible, len(projects))
		drawTextStyled(hdc, footer, rectWH(int(body.Left), footerTop+2, int(body.Right-body.Left), dashboardProjectFooterHeight-2), productionPalette.Muted, 9, fwSemiBold, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis)
	}
}

func drawProvidersPanel(hdc uintptr, rect nativeRect, providers []DashboardProviderItem) {
	body := drawPanelFrame(hdc, rect, fmt.Sprintf("Autonomous worker health · %d", len(providers)))
	if len(providers) == 0 {
		drawTextStyled(hdc, "No autonomous workers detected yet.", body, productionPalette.Muted, 12, fwNormal, dtLeft|dtWordBreak)
		return
	}
	visible, hidden := dashboardProviderWindow(len(providers), int(body.Bottom-body.Top))
	for i := 0; i < visible; i++ {
		provider := providers[i]
		y := int(body.Top) + i*dashboardProviderRowHeight
		color := productionPalette.Muted
		status := "Unavailable"
		if provider.Ready {
			color = productionPalette.Green
			status = "Ready"
		}
		if provider.Cooling {
			color = productionPalette.Purple
			status = "Cooling"
		}
		roundedPanel(hdc, rectWH(int(body.Left), y+9, 8, 8), color, color, 8)
		drawTextStyled(hdc, provider.Name, rectWH(int(body.Left)+17, y, 150, 25), productionPalette.Text, 11, fwNormal, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis)
		current := provider.CurrentTask
		if strings.TrimSpace(current) == "" {
			current = string(provider.Cost)
		}
		drawTextStyled(hdc, current, rectWH(int(body.Left)+170, y, int(body.Right-body.Left)-265, 25), productionPalette.Muted, 9, fwNormal, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis)
		drawTextStyled(hdc, status, rectWH(int(body.Right)-82, y, 80, 25), color, 10, fwSemiBold, dtRight|dtSingleLine|dtVCenter)
	}
	if hidden > 0 {
		footerTop := int(body.Bottom) - dashboardProviderFooterHeight
		fillRectColor(hdc, nativeRect{Left: body.Left, Top: int32(footerTop), Right: body.Right, Bottom: int32(footerTop + 1)}, productionPalette.Border)
		footer := fmt.Sprintf("+%d more workers · %d of %d shown", hidden, visible, len(providers))
		drawTextStyled(hdc, footer, rectWH(int(body.Left), footerTop+2, int(body.Right-body.Left), dashboardProviderFooterHeight-2), productionPalette.Muted, 9, fwSemiBold, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis)
	}
}

func (s *Shell) drawProductionButton(lParam uintptr) uintptr {
	if lParam == 0 {
		return 0
	}
	item := (*drawItemStruct)(unsafe.Pointer(lParam))
	id := int(item.CtlID)
	if id != idNavDashboard && id != idNavWork && id != idNavSettings && id != idTopNewTask && id != idTopNeedsYou && id != idTopReview {
		return 0
	}
	fill := productionPalette.Panel
	border := productionPalette.Border
	textColor := productionPalette.Text
	active := (id == idNavDashboard && s.page == pageDashboard) || (id == idNavWork && s.page == pageWork) || (id == idNavSettings && s.page == pageSettings)
	if active || id == idTopNewTask {
		fill = productionPalette.AccentSoft
		border = productionPalette.Accent
	}
	if id == idTopNeedsYou && BuildDashboardSnapshot(s.eng).Summary.NeedsHuman > 0 {
		fill = rgb(62, 44, 18)
		border = productionPalette.Amber
		textColor = productionPalette.Amber
	}
	if item.ItemState&odsSelected != 0 {
		fill = productionPalette.Accent
		border = productionPalette.Accent
	}
	if item.ItemState&odsDisabled != 0 {
		textColor = productionPalette.Muted
	}
	roundedPanel(item.HDC, item.RCItem, fill, border, 10)
	drawTextStyled(item.HDC, windowText(item.HwndItem), insetRect(item.RCItem, 8, 0), textColor, 12, fwSemiBold, dtCenter|dtSingleLine|dtVCenter|dtEndEllipsis)
	return 1
}

func activityTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	local := value.Local()
	if time.Since(local) < 24*time.Hour {
		return local.Format("15:04")
	}
	return local.Format("02 Jan")
}

func updaterInstalled() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	info, err := os.Stat(filepath.Join(filepath.Dir(exe), "Workbench-Updater.exe"))
	return err == nil && info.Mode().IsRegular()
}

func ternary(condition bool, yes, no string) string {
	if condition {
		return yes
	}
	return no
}
