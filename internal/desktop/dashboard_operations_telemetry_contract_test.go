package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestSprint1OperationsTelemetryPresentationContract(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, want := range []string{"ProgressMeasured", "telemetryBar", "ProgressStages", "telemetryStageDots", "operationsActivityAge", "operationsTelemetryElapsed", "Priority"} {
		if !strings.Contains(text, want) {
			t.Fatalf("telemetry presentation missing %q", want)
		}
	}
	if strings.Contains(strings.ToLower(text), "deterministic percentage unavailable") {
		t.Fatal("telemetry presentation must not expose percentage-unavailable copy")
	}
}

func TestSprint1OperationsCanonicalLayoutKeepsCompactLanesWide(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_controls_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, want := range []string{
		"colW := (contentW - colGap) / 2",
		"rowH := (boardH - rowGap*2) / 3",
		"col := i % 2",
		"row := i / 2",
		"operationsTelemetryListLine(item, time.Now())",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("canonical Operations layout missing %q", want)
		}
	}
	if strings.Contains(text, "colW := (contentW - colGap*2) / 3") {
		t.Fatal("canonical Operations layout regressed to three narrow columns")
	}
}

func TestSprint1OperationsListboxesAreOwnerDrawnAtCreation(t *testing.T) {
	creator, err := os.ReadFile("dashboard_operations_controls_ownerdraw_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	creatorText := string(creator)
	for _, want := range []string{
		"lbsNotify | lbsOwnerDrawFixed | lbsHasStrings",
		"wsVScroll | lbsOwnerDrawFixed | lbsHasStrings",
		"prepareOperationsOwnerDrawList",
		"lbSetItemHeight",
	} {
		if !strings.Contains(creatorText, want) {
			t.Fatalf("Operations creation-time owner draw missing %q", want)
		}
	}

	ownerDraw, err := os.ReadFile("dashboard_operations_listbox_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	ownerText := string(ownerDraw)
	for _, want := range []string{"wmMeasureItem", "measureOperationsListItem", "ItemHeight = 22", "drawOperationsListItem", "dtEndEllipsis"} {
		if !strings.Contains(ownerText, want) {
			t.Fatalf("Operations owner-draw implementation missing %q", want)
		}
	}

	shell, err := os.ReadFile("production_shell_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	shellText := string(shell)
	for _, want := range []string{
		"s.createOperationsDashboardControlsOwnerDraw()",
		"case wmMeasureItem:",
		"s.measureOperationsListItem(lParam)",
		"s.drawOperationsListItem(lParam)",
	} {
		if !strings.Contains(shellText, want) {
			t.Fatalf("production owner-draw route missing %q", want)
		}
	}
}
