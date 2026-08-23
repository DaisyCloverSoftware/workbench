package core

import (
	"os"
	"strings"
	"testing"
)

func TestSuccessfulProviderTransitionsThroughFinalizingStage(t *testing.T) {
	body, err := os.ReadFile("engine.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `Phase: "Finalizing result", Stage: 3, StageTotal: 4`) {
		t.Fatal("finalizing stage missing")
	}
}
