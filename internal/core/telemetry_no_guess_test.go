package core

import (
	"os"
	"strings"
	"testing"
)

func TestCoreProgressFallbackNeverCalculatesPercentageFromTimeOrStage(t *testing.T) {
	body, err := os.ReadFile("work_item.go")
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ToLower(string(body))
	if strings.Contains(text, "elapsed") && strings.Contains(text, "percent") {
		t.Fatal("core fallback must not infer percentage from elapsed time")
	}
}
