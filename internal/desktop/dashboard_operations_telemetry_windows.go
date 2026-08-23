//go:build windows

package desktop

import (
	"fmt"
	"strings"
	"time"

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
}

func operationsTelemetryListLine(item core.WorkItem, now time.Time) string {
	project := telemetryCompact(operationsProjectLabel(item), 18)
	title := telemetryCompact(item.Title, 32)
	state := strings.ToUpper(string(item.State))
	priority := strings.ToUpper(item.Priority.String())
	activity := operationsActivityAge(item.UpdatedAt, now)

	if item.State == core.TaskQueued {
		queue := priority
		if item.QueuePosition > 0 {
			queue = fmt.Sprintf("#%d %s", item.QueuePosition, priority)
		}
		return fmt.Sprintf("%s · %s — %s · %s · %s", queue, project, title, state, activity)
	}
	progress := operationsTelemetryProgress(item.Progress, item.State)
	runtime := operationsTelemetryElapsed(item, now)
	return fmt.Sprintf("%s · %s — %s · %s · %s · %s · %s", state, project, title, progress, runtime, activity, priority)
}

func operationsTelemetryExpandedLine(item core.WorkItem, now time.Time) string {
	line := operationsTelemetryListLine(item, now)
	worker := strings.TrimSpace(item.Provider)
	if worker != "" {
		line += " · worker " + worker
	}
	return line
}

func operationsTelemetryProgress(progress core.WorkProgress, status core.TaskStatus) string {
	phase := strings.TrimSpace(progress.Phase)
	if phase == "" {
		phase = string(status)
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
	writeOperationsDetailLine(&b, "State", strings.ToUpper(string(item.State)))
	writeOperationsDetailLine(&b, "Current stage", strings.TrimSpace(item.Progress.Phase))
	writeOperationsDetailLine(&b, "Progress", operationsTelemetryProgress(item.Progress, item.State))
	if item.StartedAt != nil && !item.StartedAt.IsZero() {
		writeOperationsDetailLine(&b, "Started", item.StartedAt.Local().Format("2 Jan 15:04:05"))
		writeOperationsDetailLine(&b, "Elapsed", telemetryDuration(now.Sub(*item.StartedAt)))
	}
	if !item.UpdatedAt.IsZero() {
		activity := strings.TrimSpace(item.Progress.Phase)
		if activity == "" {
			activity = strings.ToUpper(string(item.State))
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