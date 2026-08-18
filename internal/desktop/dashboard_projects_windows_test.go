//go:build windows

package desktop

import (
	"testing"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestSortDashboardProjectsForDisplaySurfacesActionableWork(t *testing.T) {
	projects := []DashboardProjectItem{
		{Name: "inactive-pinned", Pinned: true},
		{Name: "active-two", Summary: core.TaskDashboardSummary{Active: 2}},
		{Name: "needs-you", Summary: core.TaskDashboardSummary{NeedsHuman: 1}},
		{Name: "active-one", Summary: core.TaskDashboardSummary{Active: 1}},
		{Name: "inactive"},
	}
	sortDashboardProjectsForDisplay(projects)
	want := []string{"needs-you", "active-two", "active-one", "inactive-pinned", "inactive"}
	for i, name := range want {
		if projects[i].Name != name {
			t.Fatalf("order[%d]=%q want %q: %+v", i, projects[i].Name, name, projects)
		}
	}
}

func TestDashboardProjectWindowAccountsForHiddenProjects(t *testing.T) {
	visible, hidden := dashboardProjectWindow(22, dashboardProjectRowHeight*4+dashboardProjectFooterHeight)
	if visible != 4 || hidden != 18 {
		t.Fatalf("visible=%d hidden=%d want 4/18", visible, hidden)
	}
}
