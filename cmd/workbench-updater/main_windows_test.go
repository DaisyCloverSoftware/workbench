//go:build windows

package main

import "testing"

func TestUpdaterKnowsProductionWorkbenchWindowClass(t *testing.T) {
	want := "DaisyCloverWorkbenchProductionDashboard"
	for _, className := range workbenchWindowClasses {
		if className == want {
			return
		}
	}
	t.Fatalf("updater handoff does not include production Workbench window class %q", want)
}

func TestUpdaterKeepsLegacyWorkbenchWindowClassForUpgradeCompatibility(t *testing.T) {
	want := "DaisyCloverWorkbenchProductionWindow"
	for _, className := range workbenchWindowClasses {
		if className == want {
			return
		}
	}
	t.Fatalf("updater handoff does not include legacy Workbench window class %q", want)
}
