package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestQueuedRowCombinesQueuePositionAndPriorityName(t *testing.T) {
	body, err := os.ReadFile("dashboard_operations_telemetry_windows.go")
	if err != nil { t.Fatal(err) }
	if !strings.Contains(string(body), `queue = fmt.Sprintf("#%d %s", item.QueuePosition, priority)`) {
		t.Fatal("queued row does not combine queue position and priority")
	}
}
