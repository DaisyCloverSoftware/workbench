package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestDetailExplicitlyLabelsStartedTimestamp(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil { t.Fatal(err) }
	if !strings.Contains(string(body), `"Started"`) {
		t.Fatal("started timestamp detail label missing")
	}
}
