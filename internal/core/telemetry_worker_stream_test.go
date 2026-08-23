package core

import (
	"os"
	"strings"
	"testing"
)

func TestHarnessWaitsForProgressStreamBeforeDecodingFinalResult(t *testing.T) {
	body, err := os.ReadFile("harness_protocol.go")
	if err != nil { t.Fatal(err) }
	text := string(body)
	wait := strings.Index(text, "runErr := cmd.Wait()")
	join := strings.Index(text, "wg.Wait()")
	decode := strings.Index(text, "decodeHarnessJobResult")
	if wait < 0 || join < wait || decode < join {
		t.Fatal("structured harness progress stream/final result order is incorrect")
	}
}
