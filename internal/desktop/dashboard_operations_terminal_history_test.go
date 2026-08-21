package desktop

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestOperationsDashboardHundredTerminalEventsDoNotBecomeRunningJobs(t *testing.T) {
	store, err := core.NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	eng, err := core.NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	remote := make([]core.RunnerChatActivityInfo, 0, 101)
	for i := 0; i < 100; i++ {
		state := "completed"
		if i%2 == 1 {
			state = "failed"
		}
		remote = append(remote, core.RunnerChatActivityInfo{
			ID:          fmt.Sprintf("terminal_%03d_12345678", i),
			ProjectRef:  "runner://workbench",
			Action:      "run_safe_command",
			State:       state,
			UpdatedAt:   now.Add(-time.Duration(i) * time.Second),
			Active:      true,
			ActiveKnown: true,
		})
	}
	remote = append(remote, core.RunnerChatActivityInfo{
		ID:          "actual_running_12345678",
		ProjectRef:  "runner://workbench",
		Action:      "run_safe_command",
		State:       "running",
		UpdatedAt:   now,
		Active:      true,
		ActiveKnown: true,
	})

	got := buildDashboardOperationsSnapshot(eng, remote)
	if got.Running != 1 || got.Queued != 0 || got.Waiting != 0 || got.NeedsHuman != 0 {
		t.Fatalf("terminal history inflated live counters: %#v", got)
	}
	if len(got.Items) != 1 || got.Items[0].ID != "actual_running_12345678" || got.Items[0].State != core.TaskRunning {
		t.Fatalf("live board should contain only the real running operation: %#v", got.Items)
	}
}
