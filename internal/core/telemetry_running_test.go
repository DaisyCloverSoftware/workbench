package core

import (
	"os"
	"strings"
	"testing"
)

func TestRunningTaskUsesExecutionStageTwo(t *testing.T) {
	body, err := os.ReadFile("engine.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `Stage: 2, StageTotal: 4`) {
		t.Fatal("running execution stage missing")
	}
}
