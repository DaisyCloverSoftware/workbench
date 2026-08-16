package desktop

import (
	"strings"
	"testing"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestWaitingDependencyIsAutoSelectedAsActiveWork(t *testing.T) {
	tasks := []TaskItem{
		{ID: "done", Status: core.TaskCompleted},
		{ID: "dependency", Status: core.TaskWaitingDependency},
	}
	if got := chooseSelectedTask(tasks, ""); got != "dependency" {
		t.Fatalf("selected task=%q want waiting dependency", got)
	}
}

func TestWindowsCancelControlIncludesWaitingDependency(t *testing.T) {
	text := desktopSource(t, "shell_windows.go")
	needle := "item.Status == core.TaskWaitingDependency"
	if !strings.Contains(text, needle) {
		t.Fatalf("Windows Cancel state omits durable dependency wait: missing %q", needle)
	}
}
