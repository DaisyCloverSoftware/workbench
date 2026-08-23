//go:build windows

package desktop

import (
	"fmt"
	"strings"
	"time"
	"unsafe"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

const wmAppOperationsClock = 0x8002

func (s *Shell) refreshOperationsTelemetryPresentation() {
	if s == nil || s.controls[idOpsDetails] == 0 || s.page != pageDashboard {
		return
	}
	now := time.Now()
	surface := operationsDashboardUI.Surface
	for _, control := range operationsLaneControls {
		items := surface.Live.ByLane[control.Lane]
		lines := make([]string, 0, len(items))
		selection := -1
		for i, item := range items {
			lines = append(lines, operationsTelemetryListLine(item, now))
			if item.ID == operationsDashboardUI.SelectedID {
				selection = i
			}
		}
		setOperationsList(s.controls[control.List], lines, selection)
	}
	if operationsDashboardUI.ExpandedLane != "" {
		items := surface.Live.ByLane[operationsDashboardUI.ExpandedLane]
		lines := make([]string, 0, len(items))
		selection := -1
		for i, item := range items {
			lines = append(lines, operationsTelemetryExpandedLine(item, now))
			if item.ID == operationsDashboardUI.SelectedID {
				selection = i
			}
		}
		setOperationsList(s.controls[idOpsFullList], lines, selection)
	}
	if detail, ok := surface.Details[operationsDashboardUI.SelectedID]; ok && operationsDashboardUI.SelectedID != "" {
		setWindowText(s.controls[idOpsDetails], operationsTelemetryDetailText(detail, now))
	}
	// The original Sprint 1 board used three narrow columns. That clipped the
	// operational telemetry before an owner could read it. Compact mode is now
	// two columns by three rows so each job exposes progress/runtime/activity at
	// normal production window width without changing the deeper detail layout.
	s.layoutOperationsTelemetryCompactBoard()
}

func operationsTelemetryListLine(item core.WorkItem, now time.Time) string {
	project := telemetryCompact(operationsProjectLabel(item), 10)
	title := telemetryCompact(item.Title, 28)
	state := operationsTelemetryState(item.State)
	priority := strings.ToUpper(item.Priority.String())
	task := project + "/" + title
	activity := operationsActivityAgeCompact(item.UpdatedAt, now)

	if item.State == core.TaskQueued {
		queue := ""
		if item.QueuePosition > 0 {
			queue = fmt.Sprintf(" #%d", item.QueuePosition)
		}
		return fmt.Sprintf("%s%s · %s · %s · %s", state, queue, priority, activity, task)
	}
	progress := operationsTelemetryCompactProgress(item.Progress, item.State)
	runtime := operationsTelemetryElapsedCompact(item, now)
	return fmt.Sprintf("%s · %s · %s · %s · %s · %s", state, progress, runtime, activity, priority, task)
}

func operationsTelemetryExpandedLine(item core.WorkItem, now time.Time) string {
	line := operationsTelemetryListLine(item, now)
	worker := strings.TrimSpace(item.Provider)
	if worker != "" {
		line += " · worker " + worker
	}
	return line
}

func operationsTelemetryState(status core.TaskStatus) string {
	switch status {
	case core.TaskRunning:
		return "RUNNING"
	case core.TaskRouting:
		return "STARTING"
	case core.TaskQueued:
		return "QUEUED"
	case core.TaskWaitingDependency, core.TaskWaitingRetry:
		return "WAITING"
	case core.TaskNeedsAttention:
		return "NEEDS YOU"
	case core.TaskCompleted:
		return "COMPLETED"
	case core.TaskFailed:
		return "FAILED"
	case core.TaskCancelled:
		return "CANCELLED"
	default:
		return strings.ToUpper(strings.ReplaceAll(string(status), "_", " "))
	}
}

func operationsTelemetryCompactProgress(progress core.WorkProgress, status core.TaskStatus) string {
	phase := telemetryCompact(strings.TrimSpace(progress.Phase), 12)
	if phase == "" {
		phase = telemetryCompact(operationsTelemetryState(status), 12)
	}
	switch progress.Kind {
	case core.ProgressMeasured:
		if progress.Total > 0 {
			current := progress.Current
			if current < 0 {
				current = 0
			}
			if current > progress.Total {
				current = progress.Total
			}
			percent := int((current * 100) / progress.Total)
			return fmt.Sprintf("%d%% %s", percent, telemetryBar(percent, 4))
		}
	case core.ProgressStages:
		if progress.StageTotal > 0 && progress.Stage > 0 {
			stage := progress.Stage
			if stage > progress.StageTotal {
				stage = progress.StageTotal
			}
			dots := strings.ReplaceAll(telemetryStageDots(stage, progress.StageTotal), " ", "")
			return fmt.Sprintf("Stage %d/%d %s", stage, progress.StageTotal, dots)
		}
	}
	return phase
}

func operationsTelemetryProgress(progress core.WorkProgress, status core.TaskStatus) string {
	phase := strings.TrimSpace(progress.Phase)
	if phase == "" {
		phase = operationsTelemetryState(status)
	}
	switch progress.Kind {
	case core.ProgressMeasured:
		if progress.Total > 0 {
			current := progress.Current
			if current < 0 {
				current = 0
			}
			if current > progress.Total {
				current = progress.Total
			}
			percent := int((current * 100) / progress.Total)
			unit := strings.TrimSpace(progress.Unit)
			measure := fmt.Sprintf("%d/%d", current, progress.Total)
			if unit != "" {
				measure += " " + unit
			}
			return fmt.Sprintf("%s %d%% · %s · %s", telemetryBar(percent, 12), percent, phase, measure)
		}
	case core.ProgressStages:
		if progress.StageTotal > 0 && progress.Stage > 0 {
			stage := progress.Stage
			if stage > progress.StageTotal {
				stage = progress.StageTotal
			}
			return fmt.Sprintf("%s Stage %d/%d — %s", telemetryStageDots(stage, progress.StageTotal), stage, progress.StageTotal, phase)
		}
	}
	return phase
}

func telemetryBar(percent, width int) string {
	if width < 1 {
		width = 1
	}
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	filled := percent * width / 100
	if percent == 100 {
		filled = width
	}
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", width-filled) + "]"
}

func telemetryStageDots(stage, total int) string {
	if total < 1 {
		return ""
	}
	if stage < 0 {
		stage = 0
	}
	if stage > total {
		stage = total
	}
	if total <= 8 {
		parts := make([]string, 0, total)
		for i := 1; i <= total; i++ {
			if i <= stage {
				parts = append(parts, "●")
			} else {
				parts = append(parts, "○")
			}
		}
		return strings.Join(parts, " ")
	}
	percent := stage * 100 / total
	return telemetryBar(percent, 8)
}

func operationsTelemetryElapsed(item core.WorkItem, now time.Time) string {
	if item.StartedAt == nil || item.StartedAt.IsZero() {
		return "elapsed —"
	}
	return "elapsed " + telemetryDuration(now.Sub(*item.StartedAt))
}

func operationsTelemetryElapsedCompact(item core.WorkItem, now time.Time) string {
	if item.StartedAt == nil || item.StartedAt.IsZero() {
		return "elapsed —"
	}
	return telemetryDuration(now.Sub(*item.StartedAt)) + " elapsed"
}

func operationsActivityAge(at, now time.Time) string {
	if at.IsZero() {
		return "activity —"
	}
	d := now.Sub(at)
	if d < 0 {
		d = 0
	}
	return "activity " + telemetryDuration(d) + " ago"
}

func operationsActivityAgeCompact(at, now time.Time) string {
	if at.IsZero() {
		return "activity —"
	}
	d := now.Sub(at)
	if d < 0 {
		d = 0
	}
	return telemetryDuration(d) + " ago"
}

func telemetryDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
}

func operationsTelemetryDetailText(detail DashboardOperationsDetail, now time.Time) string {
	var b strings.Builder
	item := detail.Item
	writeOperationsDetailLine(&b, "Project", detail.Project)
	writeOperationsDetailLine(&b, "Task / sprint", detail.Task)
	writeOperationsDetailLine(&b, "Assigned worker", detail.Worker)
	writeOperationsDetailLine(&b, "State", operationsTelemetryState(item.State))
	writeOperationsDetailLine(&b, "Current stage", strings.TrimSpace(item.Progress.Phase))
	writeOperationsDetailLine(&b, "Progress", operationsTelemetryProgress(item.Progress, item.State))
	if item.StartedAt != nil && !item.StartedAt.IsZero() {
		writeOperationsDetailLine(&b, "Started", item.StartedAt.Local().Format("2 Jan 15:04:05"))
		writeOperationsDetailLine(&b, "Elapsed", telemetryDuration(now.Sub(*item.StartedAt)))
	}
	if !item.UpdatedAt.IsZero() {
		activity := strings.TrimSpace(item.Progress.Phase)
		if activity == "" {
			activity = operationsTelemetryState(item.State)
		}
		writeOperationsDetailLine(&b, "Latest meaningful activity", fmt.Sprintf("%s · %s · %s", activity, item.UpdatedAt.Local().Format("2 Jan 15:04:05"), operationsActivityAge(item.UpdatedAt, now)))
	}
	priority := item.Priority.String()
	if item.State == core.TaskQueued && item.QueuePosition > 0 {
		priority += fmt.Sprintf(" · queue #%d", item.QueuePosition)
	}
	writeOperationsDetailLine(&b, "Priority", priority)
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

func (s *Shell) layoutOperationsTelemetryCompactBoard() {
	if s == nil || s.page != pageDashboard || operationsDashboardUI.SelectedID != "" || operationsDashboardUI.ExpandedLane != "" {
		return
	}
	var rect nativeRect
	procGetClientRect.Call(s.hwnd, uintptr(unsafe.Pointer(&rect)))
	clientWidth := int(rect.Right - rect.Left)
	clientHeight := int(rect.Bottom - rect.Top)
	contentX := productionSidebarWidth + 20
	contentY := productionHeaderHeight + 18
	contentW := clientWidth - contentX - 18
	contentH := clientHeight - contentY - 18
	if contentW < 720 || contentH < 560 {
		return
	}

	metricsY := contentY + 68
	metricH := 72
	boardY := metricsY + metricH + 12
	resourceH := 132
	resourceY := contentY + contentH - resourceH
	boardH := resourceY - boardY - 12
	if boardH < 240 {
		return
	}

	s.hideOperationsDashboardControls()
	colGap := 10
	rowGap := 8
	colW := (contentW - colGap) / 2
	rowH := (boardH - rowGap*2) / 3
	for i, control := range operationsLaneControls {
		col := i % 2
		row := i / 2
		x := contentX + col*(colW+colGap)
		y := boardY + row*(rowH+rowGap)
		showWindow(s.controls[control.Header], true)
		showWindow(s.controls[control.List], true)
		moveWindow(s.controls[control.Header], x+4, y+2, colW-8, 26)
		moveWindow(s.controls[control.List], x+6, y+32, colW-12, rowH-38)
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

func telemetryCompact(value string, limit int) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if limit <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	if limit == 1 {
		return "…"
	}
	return string(runes[:limit-1]) + "…"
}
