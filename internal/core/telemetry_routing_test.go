package core

import (
	"os"
	"strings"
	"testing"
)

func TestExecutionStartsAtRoutingStageOne(t *testing.T) {
	body, err := os.ReadFile("engine.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `Phase: "Selecting executor", Stage: 1, StageTotal: 4`) {
		t.Fatal("routing stage one missing")
	}
}
