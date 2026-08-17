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
	contentX := productionSidebarWidth + 20
	contentY := productionHeaderHeight + 18
	contentW := int(client.Right) - contentX - 18
	contentH := int(client.Bottom) - contentY - 18
	if contentW < 800 || contentH < 620 {
		return
	}

	drawTextStyled(hdc, "Dashboard", rectWH(contentX, contentY, contentW, 32), productionPalette.Text, 27, fwBold, dtLeft|dtSingleLine|dtVCenter)
	drawTextStyled(hdc, "ChatGPT-first overview and autonomous escalation activity", rectWH(contentX, contentY+34, contentW, 22), productionPalette.Muted, 14, fwNormal, dtLeft|dtSingleLine|dtVCenter)

	metricsY := contentY + 70
	metricGap := 10
	metricH := 94
	metricW := (contentW - metricGap*5) / 6
	terminal := d.Summary.Completed + d.Summary.Failed
	success := "—"
	if terminal > 0 {
		success = fmt.Sprintf("%d%%", d.SuccessRate)
	}
	metrics := []struct {
		label, value, detail string
		accent               uint32
	}{
		{"Active tasks", fmt.Sprintf("%d", d.Summary.Active), "running or waiting", productionPalette.Accent},
		{"Needs you", fmt.Sprintf("%d", d.Summary.NeedsHuman), "human decisions", productionPalette.Amber},
		{"Completed", fmt.Sprintf("%d", d.Summary.Completed), "durable results", productionPalette.Green},
		{"Success rate", success, "completed vs failed", productionPalette.Purple},
		{"Autonomous ready", fmt.Sprintf("%d / %d", d.ProviderReady, d.ProviderTotal), "escalation workers", productionPalette.Teal},
		{"Projects", fmt.Sprintf("%d", d.ProjectCount), "registered workspaces", productionPalette.Accent},
	}
	for i, metric := range metrics {
		rect := rectWH(contentX+i*(metricW+metricGap), metricsY, metricW, metricH)
		drawMetricCard(hdc, rect, metric.label, metric.value, metric.detail, metric.accent)
	}

	mainY := metricsY + metricH + 14
	bottomH := 250
	bottomY := contentY + contentH - bottomH
	mainH := bottomY - mainY - 14
	rightW := 250
	leftAvailable := contentW - rightW - 20
	recentW := (leftAvailable * 52) / 100
	activeW := leftAvailable - recentW - 10

	recentRect := rectWH(contentX, mainY, recentW, mainH)
	activeRect := rectWH(contentX+recentW+10, mainY, activeW, mainH)
	systemRect := rectWH(contentX+recentW+activeW+20, mainY, rightW, mainH)
	drawActivityPanel(hdc, recentRect, d.RecentActivity)
	drawActiveTasksPanel(hdc, activeRect, d.ActiveTasks)
	s.drawSystemPanel(hdc, systemRect, d)

	projectsW := (contentW * 55) / 100
	projectsRect := rectWH(contentX, bottomY, projectsW-6, bottomH)
	providersRect := rectWH(contentX+projectsW+6, bottomY, contentW-projectsW-6, bottomH)
	drawProjectsPanel(hdc, projectsRect, d.Projects)
	drawProvidersPanel(hdc, providersRect, d.Providers)

	healthText := "ChatGPT primary ready"
	healthColor := productionPalette.Green
	if s.mcpURL == "" {
		healthText = "ChatGPT bridge offline"
		healthColor = productionPalette.Amber
	}
	pill := rectWH(int(client.Right)-196, 78, 176, 30)
	roundedPanel(hdc, pill, productionPalette.Panel, healthColor, 12)
	drawTextStyled(hdc, "●  "+healthText, insetRect(pill, 10, 0), healthColor, 12, fwSemiBold, dtLeft|dtSingleLine|dtVCenter)
}

func drawMetricCard(hdc uintptr, rect nativeRect, label, value, detail string, accent uint32) {
	roundedPanel(hdc, rect, productionPalette.Panel, productionPalette.Border, 12)
	fillRectColor(hdc, nativeRect{Left: rect.Left, Top: rect.Top, Right: rect.Left + 4, Bottom: rect.Bottom}, accent)
	drawTextStyled(hdc, label, nativeRect{Left: rect.Left + 15, Top: rect.Top + 11, Right: rect.Right - 10, Bottom: rect.Top + 31}, productionPalette.Muted, 12, fwSemiBold, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis)
	drawTextStyled(hdc, value, nativeRect{Left: rect.Left + 15, Top: rect.Top + 31, Right: rect.Right - 10, Bottom: rect.Top + 66}, productionPalette.Text, 25, fwBold, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis)
	drawTextStyled(hdc, detail, nativeRect{Left: rect.Left + 15, Top: rect.Top + 67, Right: rect.Right - 10, Bottom: rect.Bottom - 7}, productionPalette.Muted, 11, fwNormal, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis)
}

func drawPanelFrame(hdc uintptr, rect nativeRect, title string) nativeRect {
	roundedPanel(hdc, rect, productionPalette.Panel, productionPalette.Border, 12)
	drawTextStyled(hdc, title, nativeRect{Left: rect.Left + 14, Top: rect.Top + 10, Right: rect.Right - 14, Bottom: rect.Top + 36}, productionPalette.Text, 15, fwSemiBold, dtLeft|dtSingleLine|dtVCenter)
	fillRectColor(hdc, nativeRect{Left: rect.Left + 1, Top: rect.Top + 42, Right: rect.Right - 1, Bottom: rect.Top + 43}, productionPalette.Border)
	return nativeRect{Left: rect.Left + 12, Top: rect.Top + 51, Right: rect.Right - 12, Bottom: rect.Bottom - 10}
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
	body := drawPanelFrame(hdc, rect, "Active tasks")
	if len(tasks) == 0 {
		drawTextStyled(hdc, "Nothing is currently running or waiting.", body, productionPalette.Muted, 12, fwNormal, dtLeft|dtWordBreak)
		return
	}
	rowH := 58
	for i, task := range tasks {
		if i >= 5 || body.Top+int32((i+1)*rowH) > body.Bottom {
			break
		}
		y := int(body.Top) + i*rowH
		color := statusColor(task.StatusLabel)
		drawTextStyled(hdc, task.Title, rectWH(int(body.Left), y, int(body.Right-body.Left)-90, 22), productionPalette.Text, 12, fwSemiBold, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis)
		drawTextStyled(hdc, task.StatusLabel, rectWH(int(body.Right)-86, y, 84, 22), color, 11, fwSemiBold, dtRight|dtSingleLine|dtVCenter|dtEndEllipsis)
		provider := strings.TrimSpace(task.Provider)
		if provider == "" {
			provider = "Router"
		}
		drawTextStyled(hdc, provider, rectWH(int(body.Left), y+23, 120, 18), productionPalette.Muted, 10, fwNormal, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis)
		next := task.NextAction
		if task.RetryAt != nil {
			next = "Retries " + task.RetryAt.Local().Format("15:04")
		}
		drawTextStyled(hdc, next, rectWH(int(body.Left)+126, y+23, int(body.Right-body.Left)-128, 18), productionPalette.Muted, 10, fwNormal, dtRight|dtSingleLine|dtVCenter|dtEndEllipsis)
		bar := rectWH(int(body.Left), y+47, int(body.Right-body.Left), 3)
		fillRectColor(hdc, bar, productionPalette.Border)
		fillRectColor(hdc, nativeRect{Left: bar.Left, Top: bar.Top, Right: bar.Right, Bottom: bar.Bottom}, color)
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
	body := drawPanelFrame(hdc, rect, "Projects")
	if len(projects) == 0 {
		drawTextStyled(hdc, "Add a repository from Work to start.", body, productionPalette.Muted, 12, fwNormal, dtLeft|dtWordBreak)
		return
	}
	rowH := 43
	for i, project := range projects {
		if i >= 5 || body.Top+int32((i+1)*rowH) > body.Bottom {
			break
		}
		y := int(body.Top) + i*rowH
		name := project.Name
		if project.Pinned {
			name = "★ " + name
		}
		drawTextStyled(hdc, name, rectWH(int(body.Left), y, int(body.Right-body.Left)-185, 22), productionPalette.Text, 12, fwSemiBold, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis)
		detail := fmt.Sprintf("%d active  ·  %d need you  ·  %d ready", project.Summary.Active, project.Summary.NeedsHuman, project.Summary.Completed)
		drawTextStyled(hdc, detail, rectWH(int(body.Right)-180, y, 178, 22), productionPalette.Muted, 10, fwNormal, dtRight|dtSingleLine|dtVCenter|dtEndEllipsis)
		drawTextStyled(hdc, project.Path, rectWH(int(body.Left), y+21, int(body.Right-body.Left), 18), productionPalette.Muted, 9, fwNormal, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis)
	}
}

func drawProvidersPanel(hdc uintptr, rect nativeRect, providers []DashboardProviderItem) {
	body := drawPanelFrame(hdc, rect, "Autonomous worker health")
	if len(providers) == 0 {
		drawTextStyled(hdc, "No autonomous workers detected yet.", body, productionPalette.Muted, 12, fwNormal, dtLeft|dtWordBreak)
		return
	}
	rowH := 36
	for i, provider := range providers {
		if i >= 6 || body.Top+int32((i+1)*rowH) > body.Bottom {
			break
		}
		y := int(body.Top) + i*rowH
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
		roundedPanel(hdc, rectWH(int(body.Left), y+10, 8, 8), color, color, 8)
		drawTextStyled(hdc, provider.Name, rectWH(int(body.Left)+17, y, 150, 27), productionPalette.Text, 11, fwNormal, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis)
		current := provider.CurrentTask
		if strings.TrimSpace(current) == "" {
			current = string(provider.Cost)
		}
		drawTextStyled(hdc, current, rectWH(int(body.Left)+170, y, int(body.Right-body.Left)-265, 27), productionPalette.Muted, 10, fwNormal, dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis)
		drawTextStyled(hdc, status, rectWH(int(body.Right)-82, y, 80, 27), color, 10, fwSemiBold, dtRight|dtSingleLine|dtVCenter)
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
