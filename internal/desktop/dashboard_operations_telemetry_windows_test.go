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
	for _, want := range []string{"RUNNING", "Workbench/Verify source", "64%", "1m30s elapsed", "8s ago", "HIGH"} {
		if !strings.Contains(line, want) {
			t.Fatalf("row %q missing %q", line, want)
		}
	}
	if !strings.Contains(line, "█") || !strings.Contains(line, "░") {
		t.Fatalf("row lacks visible measured progress bar: %q", line)
	}
	if strings.Contains(line, "WORKING") {
		t.Fatalf("row leaked internal working label: %q", line)
	}
	assertOperationsTelemetryBeforeTask(t, line, "64%", "elapsed", "ago", "HIGH")
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
	for _, want := range []string{"RUNNING", "Stage 3/5", "22s elapsed", "2s ago", "NORMAL", "●", "○"} {
		if !strings.Contains(line, want) {
			t.Fatalf("row %q missing %q", line, want)
		}
	}
	if strings.Contains(line, "%") || strings.Contains(strings.ToLower(line), "percentage unavailable") {
		t.Fatalf("stage row invented or exposed unavailable percentage: %q", line)
	}
	assertOperationsTelemetryBeforeTask(t, line, "Stage 3/5", "elapsed", "ago", "NORMAL")
}

func TestOperationsTelemetryQueuedRowMakesPriorityAndQueueImmediatelyVisible(t *testing.T) {
	now := time.Now()
	item := core.WorkItem{ProjectName: "Workbench", Title: "Queued task", State: core.TaskQueued, Priority: core.PriorityCritical, QueuePosition: 1, UpdatedAt: now}
	line := operationsTelemetryListLine(item, now)
	if !strings.Contains(line, "QUEUED #1") || !strings.Contains(line, "CRITICAL") {
		t.Fatalf("queued row=%q", line)
	}
	assertOperationsTelemetryBeforeTask(t, line, "CRITICAL", "ago")
}

func TestOperationsTelemetryUsesOwnerFacingStateLabels(t *testing.T) {
	if got := operationsTelemetryState(core.TaskRunning); got != "RUNNING" {
		t.Fatalf("running label=%q", got)
	}
	if got := operationsTelemetryState(core.TaskNeedsAttention); got != "NEEDS YOU" {
		t.Fatalf("needs-you label=%q", got)
	}
	if got := operationsTelemetryState(core.TaskWaitingDependency); got != "WAITING" {
		t.Fatalf("waiting label=%q", got)
	}
}

func assertOperationsTelemetryBeforeTask(t *testing.T, line string, fields ...string) {
	t.Helper()
	taskPos := strings.Index(line, "Workbench/")
	if taskPos < 0 {
		t.Fatalf("row %q missing task label", line)
	}
	for _, field := range fields {
		pos := strings.Index(line, field)
		if pos < 0 || pos > taskPos {
			t.Fatalf("row %q did not keep %q before task label", line, field)
		}
	}
}
