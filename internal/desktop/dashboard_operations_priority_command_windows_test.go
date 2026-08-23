//go:build windows

package desktop

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestSprint1OperationsPriorityCommandPathChangesVisibleCanonicalQueueOrder(t *testing.T) {
	now := time.Now().UTC()
	projectPath := t.TempDir()
	started := now.Add(-time.Minute)
	state := core.DefaultState()
	state.Projects = []core.Project{{ID: "project-1", Path: projectPath, Name: "Workbench", AddedAt: now, LastUsedAt: now}}
	state.ActiveProjectID = "project-1"
	state.ProjectPath = projectPath
	state.Tasks = []core.Task{
		{
			ID: "busy", ProjectPath: projectPath, Title: "Busy worker", Status: core.TaskRunning,
			Priority: core.PriorityNormal, ProviderID: "structured-harness", CreatedAt: now.Add(-3 * time.Minute), UpdatedAt: now,
			StartedAt: &started, Progress: core.WorkProgress{Kind: core.ProgressStages, Phase: "Executing worker", Stage: 2, StageTotal: 4},
		},
		{
			ID: "older", ProjectPath: projectPath, Title: "Older normal", Status: core.TaskQueued,
			Priority: core.PriorityNormal, CreatedAt: now.Add(-2 * time.Minute), UpdatedAt: now.Add(-2 * time.Minute),
		},
		{
			ID: "target", ProjectPath: projectPath, Title: "Telemetry queued priority target", Status: core.TaskQueued,
			Priority: core.PriorityNormal, CreatedAt: now.Add(-time.Minute), UpdatedAt: now.Add(-time.Minute),
		},
	}
	store, err := core.NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(state); err != nil {
		t.Fatal(err)
	}
	eng, err := core.NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}

	beforeSurface := buildDashboardOperationsSurface(eng, nil)
	before, ok := beforeSurface.Details["target"]
	if !ok {
		t.Fatal("target missing from canonical Operations surface")
	}
	if before.Item.QueuePosition != 2 || before.Item.Priority != core.PriorityNormal {
		t.Fatalf("before target priority/order = %s/#%d, want Normal/#2", before.Item.Priority.String(), before.Item.QueuePosition)
	}
	beforeRow := operationsTelemetryListLine(before.Item, now)
	if !strings.Contains(beforeRow, "QUEUED #2") || !strings.Contains(beforeRow, "NORMAL") {
		t.Fatalf("before visible row = %q", beforeRow)
	}

	oldUI := operationsDashboardUI
	defer func() { operationsDashboardUI = oldUI }()
	operationsDashboardUI = operationsDashboardUIState{
		Surface:    beforeSurface,
		SelectedID: "target",
		LaneIDs:    map[core.WorkLane][]string{},
	}
	s := &Shell{eng: eng, page: pageDashboard}
	if !s.handleOperationsDashboardCommand(idOpsPriorityUp, 0) {
		t.Fatal("Priority Up WM_COMMAND path was not handled")
	}

	persisted, ok := eng.Task("target")
	if !ok {
		t.Fatal("target disappeared after Priority Up")
	}
	if persisted.Priority != core.PriorityHigh {
		t.Fatalf("durable priority = %s, want High", persisted.Priority.String())
	}

	afterSurface := buildDashboardOperationsSurface(eng, nil)
	after := afterSurface.Details["target"]
	if after.Item.Priority != core.PriorityHigh || after.Item.QueuePosition != 1 {
		t.Fatalf("after target priority/order = %s/#%d, want High/#1", after.Item.Priority.String(), after.Item.QueuePosition)
	}
	afterRow := operationsTelemetryListLine(after.Item, now.Add(time.Second))
	if !strings.Contains(afterRow, "QUEUED #1") || !strings.Contains(afterRow, "HIGH") {
		t.Fatalf("after visible row = %q", afterRow)
	}
	if beforeRow == afterRow {
		t.Fatalf("visible priority row did not change: %q", afterRow)
	}
}
