package desktop

import (
	"fmt"
	"strings"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

type DashboardWorkCard struct {
	ID               string
	ProjectName      string
	Title            string
	StateLabel       string
	PriorityLabel    string
	QueueLabel       string
	LocationLabel    string
	ProgressLabel    string
	ProgressPercent  int
	HasPercent       bool
	ElapsedLabel     string
	Blocker          string
	CanReprioritize  bool
	CanCancel        bool
	CanMove          bool
}

func BuildDashboardWorkCards(items []core.WorkItem, now time.Time) []DashboardWorkCard {
	cards := make([]DashboardWorkCard, 0, len(items))
	for _, item := range items {
		percent, hasPercent := item.Progress.Percent()
		cards = append(cards, DashboardWorkCard{
			ID:              item.ID,
			ProjectName:     strings.TrimSpace(item.ProjectName),
			Title:           strings.TrimSpace(item.Title),
			StateLabel:      workItemStateLabel(item.State),
			PriorityLabel:   workPriorityLabel(item.Priority),
			QueueLabel:      workQueueLabel(item),
			LocationLabel:   workLocationLabel(item.Location),
			ProgressLabel:   workProgressLabel(item.Progress),
			ProgressPercent: percent,
			HasPercent:      hasPercent,
			ElapsedLabel:    workElapsedLabel(item, now),
			Blocker:         strings.TrimSpace(item.Blocker),
			CanReprioritize: item.CanReprioritize,
			CanCancel:       item.CanCancel,
			CanMove:         item.CanMove,
		})
	}
	return cards
}

func workItemStateLabel(state core.WorkItemState) string {
	switch state {
	case core.WorkItemQueued:
		return "Queued"
	case core.WorkItemRouting:
		return "Routing"
	case core.WorkItemRunning:
		return "Running"
	case core.WorkItemWaiting:
		return "Waiting"
	case core.WorkItemNeedsAttention:
		return "Needs you"
	case core.WorkItemCompleted:
		return "Completed"
	case core.WorkItemFailed:
		return "Failed"
	case core.WorkItemCancelled:
		return "Cancelled"
	default:
		return "Unknown"
	}
}

func workPriorityLabel(priority core.WorkPriority) string {
	switch core.NormalizeWorkPriority(priority) {
	case core.WorkPriorityCritical:
		return "CRITICAL"
	case core.WorkPriorityHigh:
		return "HIGH"
	case core.WorkPriorityLow:
		return "LOW"
	default:
		return "NORMAL"
	}
}

func workQueueLabel(item core.WorkItem) string {
	if item.State != core.WorkItemQueued {
		return ""
	}
	if item.QueuePosition > 0 {
		return fmt.Sprintf("Queued #%d", item.QueuePosition)
	}
	return "Queued"
}

func workLocationLabel(location core.WorkLocation) string {
	parts := make([]string, 0, 3)
	if machine := strings.TrimSpace(location.Machine); machine != "" {
		parts = append(parts, machine)
	}
	if tool := strings.TrimSpace(location.Tool); tool != "" {
		parts = append(parts, tool)
	}
	if executor := strings.TrimSpace(location.Executor); executor != "" && executor != strings.TrimSpace(location.Machine) {
		parts = append(parts, executor)
	}
	if len(parts) == 0 {
		parts = append(parts, dashboardLaneLabel(location.Lane))
	}
	return strings.Join(parts, " · ")
}

func workProgressLabel(progress core.WorkProgress) string {
	switch progress.Kind {
	case core.WorkProgressMeasured:
		unit := strings.TrimSpace(progress.Unit)
		if unit == "" {
			unit = "items"
		}
		return fmt.Sprintf("%d / %d %s", progress.Current, progress.Total, unit)
	case core.WorkProgressStages:
		name := strings.TrimSpace(progress.StageName)
		if name == "" {
			name = "Stage"
		}
		return fmt.Sprintf("%s · %d / %d", name, progress.Stage, progress.StageTotal)
	case core.WorkProgressIndeterminate:
		if name := strings.TrimSpace(progress.StageName); name != "" {
			return name
		}
		return "Working"
	default:
		return ""
	}
}

func workElapsedLabel(item core.WorkItem, now time.Time) string {
	if item.StartedAt == nil || item.StartedAt.IsZero() || now.Before(*item.StartedAt) {
		return ""
	}
	elapsed := now.Sub(*item.StartedAt).Round(time.Second)
	if elapsed < time.Minute {
		return fmt.Sprintf("%ds elapsed", int(elapsed.Seconds()))
	}
	if elapsed < time.Hour {
		return fmt.Sprintf("%dm %02ds elapsed", int(elapsed.Minutes()), int(elapsed.Seconds())%60)
	}
	return fmt.Sprintf("%dh %02dm elapsed", int(elapsed.Hours()), int(elapsed.Minutes())%60)
}

func dashboardLaneLabel(lane core.WorkLane) string {
	switch lane {
	case core.WorkLaneServerOperations:
		return "Server operations"
	case core.WorkLaneCIBuilds:
		return "CI / builds"
	case core.WorkLaneWindowsHost:
		return "Windows workstation"
	case core.WorkLaneAIWorkers:
		return "AI workers"
	case core.WorkLaneNeedsHuman:
		return "Needs you"
	default:
		return "Waiting"
	}
}
