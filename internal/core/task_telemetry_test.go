package core

import (
	"strings"
	"testing"
)

func TestHarnessProgressParsesRealMeasuredAndStageTelemetry(t *testing.T) {
	measured, ok := parseHarnessProgressLine(`WORKBENCH_PROGRESS: {"kind":"measured","current":64,"total":100,"unit":"files","phase":"Verifying files"}`)
	if !ok {
		t.Fatal("measured telemetry was rejected")
	}
	if measured.Kind != ProgressMeasured || measured.Current != 64 || measured.Total != 100 || measured.Phase != "Verifying files" {
		t.Fatalf("measured=%#v", measured)
	}
	stage, ok := parseHarnessProgressLine(`WORKBENCH_PROGRESS: {"kind":"stages","stage":3,"stage_total":5,"phase":"Executing"}`)
	if !ok {
		t.Fatal("stage telemetry was rejected")
	}
	if stage.Kind != ProgressStages || stage.Stage != 3 || stage.StageTotal != 5 || stage.Phase != "Executing" {
		t.Fatalf("stage=%#v", stage)
	}
}

func TestHarnessProgressRejectsFabricatedOrInvalidMeasurements(t *testing.T) {
	bad := []string{
		`WORKBENCH_PROGRESS: {"kind":"measured","current":2,"total":0,"phase":"No denominator"}`,
		`WORKBENCH_PROGRESS: {"kind":"measured","current":11,"total":10,"phase":"Over total"}`,
		`WORKBENCH_PROGRESS: {"kind":"stages","stage":0,"stage_total":4,"phase":"Invalid stage"}`,
		`WORKBENCH_PROGRESS: {"kind":"stages","stage":5,"stage_total":4,"phase":"Past end"}`,
	}
	for _, line := range bad {
		if progress, ok := parseHarnessProgressLine(line); ok {
			t.Fatalf("invalid telemetry accepted: %#v", progress)
		}
	}
}

func TestTaskProgressUsesStagesWhenPercentageIsNotDefensible(t *testing.T) {
	development := Task{Status: TaskRunning, Mode: TaskModeDevelopment, Progress: WorkProgress{Kind: ProgressIndeterminate, Phase: "Running"}}
	got := TaskProgress(development)
	if got.Kind != ProgressStages || got.Stage != 2 || got.StageTotal != 4 || got.Phase != "Executing worker" {
		t.Fatalf("development progress=%#v", got)
	}
	operations := Task{Status: TaskRunning, Mode: TaskModeOperations, Progress: WorkProgress{Kind: ProgressIndeterminate, Phase: "Running"}}
	got = TaskProgress(operations)
	if got.Kind != ProgressStages || got.Stage != 2 || got.StageTotal != 4 || got.Phase != "Executing operation" {
		t.Fatalf("operations progress=%#v", got)
	}
	if strings.Contains(got.Phase, "%") {
		t.Fatalf("stage progress fabricated percent: %#v", got)
	}
}
