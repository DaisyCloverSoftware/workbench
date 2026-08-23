package core

import (
	"os"
	"strings"
	"testing"
)

func TestStructuredHarnessProgressKeepsFinalStdoutContract(t *testing.T) {
	body, err := os.ReadFile("harness_protocol.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, `harnessProgressPrefix   = "WORKBENCH_PROGRESS:"`) {
		t.Fatal("reserved progress prefix missing")
	}
	if !strings.Contains(text, "cmd.StderrPipe()") || !strings.Contains(text, "reportTaskTelemetry(ctx, progress)") {
		t.Fatal("live progress is not streamed from adapter stderr into task telemetry")
	}
	if !strings.Contains(text, "decodeHarnessJobResult([]byte(stdout.String()))") {
		t.Fatal("final result must remain the bounded stdout JSON contract")
	}
}
