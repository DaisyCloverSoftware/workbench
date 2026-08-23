package core

import (
	"os"
	"strings"
	"testing"
)

func TestCompletedTaskEndsAtExecutionLifecycleStageFour(t *testing.T) {
	body, err := os.ReadFile("engine.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `Phase: "Completed", Stage: 4, StageTotal: 4`) {
		t.Fatal("completed task must close the stage lifecycle")
	}
}
