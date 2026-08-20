package core

import (
	"strings"
	"time"
)

// WorkItemFromHostJob projects one durable Windows host-bridge job without
// granting capabilities the host queue does not yet implement. In particular,
// reprioritise/move/cancel remain false until those mutations exist safely.
func WorkItemFromHostJob(job HostJob, host HostBridgeHost) WorkItem {
	created := parseHostWorkTime(job.CreatedAt)
	updated := parseHostWorkTime(job.UpdatedAt)
	if updated.IsZero() {
		updated = created
	}
	var started *time.Time
	if value := parseHostWorkTime(job.ClaimedAt); !value.IsZero() {
		started = &value
	}

	machine := strings.TrimSpace(host.Label)
	if machine == "" {
		machine = strings.TrimSpace(job.HostID)
	}
	tool := hostToolLabel(job.Spec.Tool)
	operation := strings.TrimSpace(job.Spec.Operation)
	title := tool
	if operation != "" {
		title += " · " + hostOperationLabel(operation)
	}

	item := WorkItem{
		ID:        job.ID,
		Title:     title,
		State:     hostJobWorkItemState(job.Status),
		Priority:  WorkPriorityNormal,
		Location:  WorkLocation{Lane: WorkLaneWindowsHost, Executor: machine, Machine: machine, Provider: "Windows host bridge", Tool: tool},
		Progress:  WorkProgress{Kind: WorkProgressNone},
		CreatedAt: created,
		StartedAt: started,
		UpdatedAt: updated,
	}
	if item.State == WorkItemRunning {
		item.Progress = WorkProgress{Kind: WorkProgressIndeterminate, StageName: "Running on " + machine}
	}
	if item.State == WorkItemQueued && !host.Online {
		item.Blocker = "Target Windows host is offline"
	}
	if item.State == WorkItemFailed {
		item.Blocker = strings.TrimSpace(job.Error)
	}
	return item
}

// ActiveWorkItemsFromHostJobs returns only queued/claimed host work for the live
// control-room view. Completed/failed jobs remain available through the durable
// host inventory for a later history/details panel.
func ActiveWorkItemsFromHostJobs(jobs []HostJob, hosts []HostBridgeHost) []WorkItem {
	byID := make(map[string]HostBridgeHost, len(hosts))
	for _, host := range hosts {
		byID[host.HostID] = host
	}
	items := make([]WorkItem, 0, len(jobs))
	for _, job := range jobs {
		state := hostJobWorkItemState(job.Status)
		if state != WorkItemQueued && state != WorkItemRunning {
			continue
		}
		items = append(items, WorkItemFromHostJob(job, byID[job.HostID]))
	}
	return items
}

func hostJobWorkItemState(status string) WorkItemState {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "queued":
		return WorkItemQueued
	case "claimed", "running":
		return WorkItemRunning
	case "completed":
		return WorkItemCompleted
	case "failed":
		return WorkItemFailed
	case "cancelled":
		return WorkItemCancelled
	default:
		return WorkItemWaiting
	}
}

func parseHostWorkTime(raw string) time.Time {
	value, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}
	}
	return value
}

func hostToolLabel(tool string) string {
	switch strings.ToLower(strings.TrimSpace(tool)) {
	case HostBridgeToolBlender:
		return "Blender"
	case HostBridgeToolUnreal:
		return "Unreal"
	default:
		if value := strings.TrimSpace(tool); value != "" {
			return value
		}
		return "Windows job"
	}
}

func hostOperationLabel(operation string) string {
	operation = strings.TrimSpace(strings.ReplaceAll(operation, "_", " "))
	if operation == "" {
		return "job"
	}
	return operation
}
