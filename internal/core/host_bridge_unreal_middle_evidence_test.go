package core

import (
	"strings"
	"testing"
)

func TestUnrealSmokeEvidenceRetainsSignalsInOmittedMiddle(t *testing.T) {
	padding := strings.Repeat("LogInit: ordinary startup progress\n", 600)
	text := padding + "LogDerivedDataCache: ZenLocal: Status: OK!\n" + padding
	legacy := newBoundedWorkerCapture(8 << 10)
	_, _ = legacy.Write([]byte(text))
	if !legacy.Truncated() || strings.Contains(legacy.String(), "ZenLocal") {
		t.Fatal("fixture must place the diagnostic outside the retained head and tail")
	}
	collector := &unrealSmokeCapture{}
	_, _ = collector.Write([]byte(text))
	if evidence := collector.finish(); !evidence.zenLocalOK || evidence.oversizedLine {
		t.Fatalf("complete middle record was lost: %s", evidence.summary())
	}
}
