package core

import (
	"os"
	"strings"
	"testing"
)

func TestExecutionLifecycleHasMeaningfulStageNames(t *testing.T) {
	body, err := os.ReadFile("engine.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, want := range []string{"Selecting executor", "Executing worker", "Executing operation", "Finalizing result", "Completed"} {
		if !strings.Contains(text, want) {
			t.Fatalf("execution stage %q missing", want)
		}
	}
}
