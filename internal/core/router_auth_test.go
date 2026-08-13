package core

import "testing"

func TestRouteCandidatesSkipKnownUnauthenticatedWorkers(t *testing.T) {
	providers := []Provider{
		{ID: "not-ready", Installed: true, Authenticated: false, CanWrite: true, Command: "worker", Cost: CostIncluded, Priority: 10},
		{ID: "ready", Installed: true, Authenticated: true, CanWrite: true, Command: "worker", Cost: CostIncluded, Priority: 20},
	}
	got := routeCandidates(providers, Preferences{}, Task{})
	if len(got) != 1 || got[0].ID != "ready" {
		t.Fatalf("unexpected candidates: %#v", got)
	}
}

func TestCloneStateCopiesAttemptSlices(t *testing.T) {
	state := DefaultState()
	state.Tasks = []Task{{ID: "task-one", Attempts: []string{"first"}}}
	clone := cloneState(state)
	clone.Tasks[0].Attempts[0] = "changed"
	if state.Tasks[0].Attempts[0] != "first" {
		t.Fatalf("clone mutated original attempts: %#v", state.Tasks[0].Attempts)
	}
}
