package core

import (
	"os"
	"strings"
	"testing"
)

func TestTelemetryImplementationDocumentsNoGuessedPercentage(t *testing.T) {
	body, err := os.ReadFile("work_item.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "rather than a guessed percentage") {
		t.Fatal("active fallback must stay explicitly stage based")
	}
}
