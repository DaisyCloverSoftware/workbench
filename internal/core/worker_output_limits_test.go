package core

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestBoundedWorkerCaptureKeepsPrefixAndFinalAttentionMarker(t *testing.T) {
	capture := newBoundedWorkerCapture(256)
	prefix := "BEGIN-WORKER-OUTPUT\n"
	noise := strings.Repeat("x", 4096)
	tail := "\nATTENTION_REQUIRED: choose the production-safe option\nEND-WORKER-OUTPUT"
	for _, part := range []string{prefix, noise, tail} {
		if n, err := capture.Write([]byte(part)); err != nil || n != len(part) {
			t.Fatalf("Write(%d)=(%d,%v)", len(part), n, err)
		}
	}
	if !capture.Truncated() {
		t.Fatal("large worker stream was not marked truncated")
	}
	out := capture.String()
	if !strings.Contains(out, "BEGIN-WORKER-OUTPUT") {
		t.Fatalf("capture lost worker prefix: %q", out)
	}
	if !strings.Contains(out, "Workbench truncated") {
		t.Fatalf("capture did not make truncation explicit: %q", out)
	}
	if !strings.Contains(out, "END-WORKER-OUTPUT") {
		t.Fatalf("capture lost worker tail: %q", out)
	}
	result := classifyRunOutput(out)
	if result.Attention != "choose the production-safe option" {
		t.Fatalf("final attention marker was lost after stream truncation: %#v", result)
	}
}

func TestBoundedWorkerCaptureNeverGrowsWithInputVolume(t *testing.T) {
	capture := newBoundedWorkerCapture(1024)
	chunk := []byte(strings.Repeat("z", 64<<10))
	for i := 0; i < 200; i++ {
		if _, err := capture.Write(chunk); err != nil {
			t.Fatal(err)
		}
	}
	if got := len(capture.prefix) + len(capture.tail); got > 1024 {
		t.Fatalf("capture retained %d bytes, want <= 1024", got)
	}
	if capture.total != int64(len(chunk)*200) {
		t.Fatalf("capture total=%d want %d", capture.total, len(chunk)*200)
	}
}

func TestPersistedWorkerReportIsBoundedAndKeepsBothEnds(t *testing.T) {
	text := "REPORT-BEGIN\n" + strings.Repeat("0123456789", maxPersistedWorkerReportBytes) + "\nREPORT-END"
	bounded := boundPersistedWorkerText(text)
	if len(bounded) > maxPersistedWorkerReportBytes {
		t.Fatalf("persisted worker report=%d bytes, want <= %d", len(bounded), maxPersistedWorkerReportBytes)
	}
	if !strings.Contains(bounded, "REPORT-BEGIN") || !strings.Contains(bounded, "REPORT-END") {
		t.Fatalf("persisted report did not preserve both ends")
	}
	if !strings.Contains(bounded, "truncated") {
		t.Fatal("persisted report truncation was not explicit")
	}
}

func TestWorkerControlTextCannotBecomeHugeTaskField(t *testing.T) {
	question := strings.Repeat("decision ", maxWorkerControlTextBytes)
	bounded := boundWorkerControlText(question)
	if len(bounded) > maxWorkerControlTextBytes {
		t.Fatalf("control field=%d bytes, want <= %d", len(bounded), maxWorkerControlTextBytes)
	}
	if !strings.Contains(bounded, "truncated by Workbench") {
		t.Fatal("control-field truncation was not explicit")
	}
}

func TestWorkerTextBoundsDoNotSplitUTF8(t *testing.T) {
	text := strings.Repeat("🧰", maxPersistedWorkerReportBytes) + "TAIL-✓"
	bounded := boundPersistedWorkerText(text)
	if !utf8.ValidString(bounded) {
		t.Fatal("persisted worker report contains invalid UTF-8 after truncation")
	}
	control := boundWorkerControlText(strings.Repeat("é", maxWorkerControlTextBytes))
	if !utf8.ValidString(control) {
		t.Fatal("worker control text contains invalid UTF-8 after truncation")
	}
}

func TestBoundRunResultForPersistenceBoundsAllWorkerFacingText(t *testing.T) {
	res := boundRunResultForPersistence(RunResult{
		Output:            strings.Repeat("o", maxPersistedWorkerReportBytes*4),
		Attention:         strings.Repeat("a", maxWorkerControlTextBytes*4),
		WorkerUnavailable: strings.Repeat("u", maxWorkerControlTextBytes*4),
	})
	if len(res.Output) > maxPersistedWorkerReportBytes {
		t.Fatalf("output=%d", len(res.Output))
	}
	if len(res.Attention) > maxWorkerControlTextBytes || len(res.WorkerUnavailable) > maxWorkerControlTextBytes {
		t.Fatalf("control fields exceed bounds: attention=%d unavailable=%d", len(res.Attention), len(res.WorkerUnavailable))
	}
}
