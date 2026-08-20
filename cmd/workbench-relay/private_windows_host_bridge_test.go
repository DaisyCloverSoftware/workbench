package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestPrivateWindowsHostBridgeActionsAreNarrowAndProjectless(t *testing.T) {
	t.Setenv("WORKBENCH_HOST_BRIDGE_STATE_DIR", t.TempDir())
	hostID := "windows_relayhost"
	if _, err := core.RecordHostBridgeHeartbeat(core.HostBridgeHeartbeat{
		HostID:   hostID,
		Label:    "Relay Windows host",
		Platform: core.HostBridgePlatformWindows,
		Arch:     "amd64",
		Capabilities: map[string]core.HostCapability{
			core.HostBridgeToolBlender: {Installed: true, Version: "Blender 4.5.0"},
		},
	}); err != nil {
		t.Fatal(err)
	}

	for _, action := range []string{"list_windows_hosts", "run_windows_blender_version", "get_windows_host_job"} {
		if !isPrivateSafeHandsAction(action) {
			t.Fatalf("%s is not registered as a private safe-hands action", action)
		}
	}

	list, err := executePrivateSafeHands(context.Background(), privateControlEnvelope{Action: "list_windows_hosts", Args: json.RawMessage(`{}`)}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if list["count"] != 1 {
		t.Fatalf("unexpected host list: %#v", list)
	}

	submitted, err := executePrivateSafeHands(context.Background(), privateControlEnvelope{
		Action: "run_windows_blender_version",
		Args:   json.RawMessage(`{"host_id":"windows_relayhost"}`),
	}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	job, ok := submitted["host_job"].(core.HostJob)
	if !ok || job.Spec.Tool != core.HostBridgeToolBlender || job.Spec.Operation != core.HostBridgeOperationVersion {
		t.Fatalf("unexpected submitted job: %#v", submitted)
	}

	fetched, err := executePrivateSafeHands(context.Background(), privateControlEnvelope{
		Action: "get_windows_host_job",
		Args:   json.RawMessage(`{"job_id":"` + job.ID + `"}`),
	}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	got, ok := fetched["host_job"].(core.HostJob)
	if !ok || got.ID != job.ID || got.Status != "queued" {
		t.Fatalf("unexpected fetched job: %#v", fetched)
	}

	if _, err := executePrivateSafeHands(context.Background(), privateControlEnvelope{
		Action:  "run_windows_blender_version",
		Project: "runner://workbench",
		Args:    json.RawMessage(`{"host_id":"windows_relayhost"}`),
	}, "", ""); err == nil {
		t.Fatal("Windows host action accepted a project scope")
	}
}
