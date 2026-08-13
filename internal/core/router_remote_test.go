package core

import "testing"

func TestRouteCandidatesRequireRemoteRunnerConfiguration(t *testing.T) {
	providers := []Provider{{ID: "workbench-runner", Installed: true, Authenticated: true, CanWrite: true, Command: "transport", Cost: CostIncluded, Priority: 5}}
	if got := routeCandidates(providers, Preferences{}, Task{}); len(got) != 0 {
		t.Fatalf("unconfigured remote runner should be skipped: %#v", got)
	}

	got := routeCandidates(providers, Preferences{OpenClawSSHHost: "configured"}, Task{})
	found := false
	for _, p := range got {
		if p.ID == "workbench-runner" {
			found = true
		}
	}
	if !found {
		t.Fatalf("configured remote runner missing: %#v", got)
	}
}
