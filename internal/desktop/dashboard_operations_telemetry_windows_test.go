//go:build windows

package desktop

import (
	"strings"
	"testing"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestOperationsTelemetryRowShowsMeasuredProgressRuntimeActivityAndPriority(t *testing.T) {
	now := time.Date(2026, 8, 23, 3, 30, 30, 0, time.UTC)
	started := now.Add(-90 * time.Second)
	item := core.WorkItem{
		ProjectName: "Workbench", Title: "Verify source", State: core.TaskRunning,
		Priority: core.PriorityHigh, Provider: "structured-harness", StartedAt: &started,
		UpdatedAt: now.Add(-8 * time.Second),
		Progress: core.WorkProgress{Kind: core.ProgressMeasured, Current: 64, Total: 100, Unit: "files", Phase: "Verifying files"},
	}
	line := operationsTelemetryListLine(item, now)
	for _, want := range []string{"RUNNING", "64%", "Verifying files", "elapsed 1m30s", "activity 8s ago", "HIGH"} {
		if !strings.Contains(line, want) {
			t.Fatalf("row %q missing %q", line, want)
		}
	}
	if !strings.Contains(line, "█") || !strings.Contains(line, "░") {
		t.Fatalf("row lacks visible measured progress bar: %q", line)
	}
}

func TestOperationsTelemetryRowShowsStageProgressWithoutInventingPercent(t *testing.T) {
	now := time.Date(2026, 8, 23, 3, 30, 30, 0, time.UTC)
	started := now.Add(-22 * time.Second)
	item := core.WorkItem{
		ProjectName: "Workbench", Title: "Deploy operation", State: core.TaskRunning,
		Priority: core.PriorityNormal, StartedAt: &started, UpdatedAt: now.Add(-2 * time.Second),
		Progress: core.WorkProgress{Kind: core.ProgressStages, Stage: 3, StageTotal: 5, Phase: "Executing"},
	}
	line := operationsTelemetryListLine(item, now)
	for _, want := range []string{"Stage 3/5", "Executing", "elapsed 22s", "activity 2s ago", "NORMAL", "●", "○"} {
		if !strings.Contains(line, want) {
			t.Fatalf("row %q missing %q", line, want)
		}
	}
	if strings.Contains(line, "%") || strings.Contains(strings.ToLower(line), "percentage unavailable") {
		t.Fatalf("stage row invented or exposed unavailable percentage: %q", line)
	}
}

func TestOperationsTelemetryQueuedRowMakesPriorityAndQueueImmediatelyVisible(t *testing.T) {
	now := time.Now()
	item := core.WorkItem{ProjectName: "Workbench", Title: "Queued task", State: core.TaskQueued, Priority: core.PriorityCritical, QueuePosition: 1, UpdatedAt: now}
	line := operationsTelemetryListLine(item, now)
	if !strings.Contains(line, "#1 CRITICAL") || !strings.Contains(line, "QUEUED") {
		t.Fatalf("queued row=%q", line)
	}
}
