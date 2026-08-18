//go:build windows

package desktop

import (
	"sort"
	"strings"
)

const (
	dashboardProjectRowHeight    = 38
	dashboardProjectFooterHeight = 20
)

func sortDashboardProjectsForDisplay(projects []DashboardProjectItem) {
	sort.SliceStable(projects, func(i, j int) bool {
		leftNeeds, rightNeeds := projects[i].Summary.NeedsHuman > 0, projects[j].Summary.NeedsHuman > 0
		if leftNeeds != rightNeeds {
			return leftNeeds
		}
		leftActive, rightActive := projects[i].Summary.Active > 0, projects[j].Summary.Active > 0
		if leftActive != rightActive {
			return leftActive
		}
		if projects[i].Summary.Active != projects[j].Summary.Active {
			return projects[i].Summary.Active > projects[j].Summary.Active
		}
		if projects[i].Pinned != projects[j].Pinned {
			return projects[i].Pinned
		}
		return strings.ToLower(projects[i].Name) < strings.ToLower(projects[j].Name)
	})
}

func dashboardProjectWindow(projectCount, availableHeight int) (visible, hidden int) {
	if projectCount <= 0 || availableHeight <= 0 {
		return 0, 0
	}
	visible = availableHeight / dashboardProjectRowHeight
	if visible >= projectCount {
		return projectCount, 0
	}
	usable := availableHeight - dashboardProjectFooterHeight
	visible = usable / dashboardProjectRowHeight
	if visible < 1 {
		visible = 1
	}
	if visible > projectCount {
		visible = projectCount
	}
	return visible, projectCount - visible
}
