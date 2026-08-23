package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryAcceptanceMeasuredProgressHashesActualSourceFiles(t *testing.T) {
	body, err := os.ReadFile("../../.github/workflows/sprint1-operational-telemetry.yml")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, want := range []string{"filepath.WalkDir", "os.ReadFile(path)", "sha256.Sum256(data)", "int64(i+1)", "int64(len(files))"} {
		if !strings.Contains(text, want) {
			t.Fatalf("measured acceptance does not prove real work unit %q", want)
		}
	}
}
