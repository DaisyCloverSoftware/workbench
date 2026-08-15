package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func withHarnessCommandFakes(t *testing.T) {
	t.Helper()
	oldSave := saveRunnerHarnessAdapter
	oldDelete := deleteRunnerHarnessAdapter
	oldStatus := runnerHarnessStatus
	t.Cleanup(func() {
		saveRunnerHarnessAdapter = oldSave
		deleteRunnerHarnessAdapter = oldDelete
		runnerHarnessStatus = oldStatus
	})
}

func TestApplyHarnessCommandSetGetDeleteExposeOnlySafeStatus(t *testing.T) {
	withHarnessCommandFakes(t)
	const privatePath = "/private/runner/adapter"
	configured := false
	saveRunnerHarnessAdapter = func(path string) (core.RunnerHarnessConfig, error) {
		if path != privatePath {
			t.Fatalf("set path=%q want private fixture", path)
		}
		configured = true
		return core.RunnerHarnessConfig{Version: 1, AdapterPath: privatePath}, nil
	}
	deleteRunnerHarnessAdapter = func() error {
		configured = false
		return nil
	}
	runnerHarnessStatus = func() core.RunnerHarnessStatus {
		if configured {
			return core.RunnerHarnessStatus{Configured: true, Available: true, AdapterName: "adapter", Status: "configured · structured protocol v1"}
		}
		return core.RunnerHarnessStatus{Status: "not configured"}
	}

	set, err := applyHarnessCommand([]string{"set", privatePath})
	if err != nil {
		t.Fatal(err)
	}
	if !set.OK || !set.Configured || !set.Available || set.AdapterName != "adapter" {
		t.Fatalf("unexpected set response: %#v", set)
	}
	if strings.Contains(strings.Join([]string{set.AdapterName, set.Status, set.Error}, " "), privatePath) {
		t.Fatalf("set response leaked private adapter path: %#v", set)
	}

	get, err := applyHarnessCommand([]string{"get"})
	if err != nil || !get.Configured || get.AdapterName != "adapter" {
		t.Fatalf("unexpected get response=%#v err=%v", get, err)
	}

	deleted, err := applyHarnessCommand([]string{"delete"})
	if err != nil {
		t.Fatal(err)
	}
	if !deleted.OK || !deleted.Deleted || deleted.Configured || deleted.Status != "not configured" {
		t.Fatalf("unexpected delete response: %#v", deleted)
	}
}

func TestApplyHarnessCommandRejectsInvalidShapesAndPropagatesMutationFailure(t *testing.T) {
	withHarnessCommandFakes(t)
	for _, args := range [][]string{nil, {"get", "extra"}, {"set"}, {"set", ""}, {"delete", "extra"}, {"unknown"}} {
		if _, err := applyHarnessCommand(args); err == nil {
			t.Fatalf("invalid harness command was accepted: %#v", args)
		}
	}

	saveRunnerHarnessAdapter = func(string) (core.RunnerHarnessConfig, error) {
		return core.RunnerHarnessConfig{}, errors.New("invalid adapter")
	}
	if _, err := applyHarnessCommand([]string{"set", "/bad"}); err == nil || !strings.Contains(err.Error(), "invalid adapter") {
		t.Fatalf("set failure not propagated: %v", err)
	}
}
