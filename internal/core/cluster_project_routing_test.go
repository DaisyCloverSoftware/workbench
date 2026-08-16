package core

import (
	"context"
	"testing"
)

func TestClusterProjectRefusalDoesNotCoolLocalProvider(t *testing.T) {
	res, err := RunProviderIsolated(context.Background(), Provider{ID: "claude", Name: "Claude"}, Task{ID: "task-cluster", ProjectPath: "runner://garage"}, Preferences{})
	if err == nil {
		t.Fatal("expected local provider to refuse a cluster-only project")
	}
	if res.Retryable {
		t.Fatal("project-location eligibility is not a retryable provider failure")
	}
}

func TestWaitingDependencyBlocksProjectRemoval(t *testing.T) {
	if !taskBlocksProjectRemoval(TaskWaitingDependency) {
		t.Fatal("waiting_dependency must keep its project registered")
	}
}
