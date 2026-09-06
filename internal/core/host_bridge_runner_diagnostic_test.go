package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunnerDiagnosticSubmissionRequiresAdvertisedCapability(t *testing.T) {
	t.Setenv("WORKBENCH_HOST_BRIDGE_STATE_DIR", t.TempDir())
	hostID := "windows_diagnostic_fixture"
	hb := HostBridgeHeartbeat{HostID: hostID, Label: "fixture", Platform: "windows", Arch: "amd64", Capabilities: map[string]HostCapability{}}
	if _, e := RecordHostBridgeHeartbeat(hb); e != nil {
		t.Fatal(e)
	}
	if _, e := SubmitWindowsRunnerInspection(hostID); e == nil {
		t.Fatal("old client was queued a new operation")
	}
	hb.Capabilities[HostBridgeToolRunnerDiagnostic] = HostCapability{Installed: true, Version: "1"}
	if _, e := RecordHostBridgeHeartbeat(hb); e != nil {
		t.Fatal(e)
	}
	job, e := SubmitWindowsRunnerInspection(hostID)
	if e != nil {
		t.Fatal(e)
	}
	if job.Spec.Tool != HostBridgeToolRunnerDiagnostic || job.Spec.Operation != HostBridgeOperationRunnerInspect || job.Status != "queued" {
		t.Fatal("unexpected diagnostic")
	}
	if _, e := SubmitHostBridgeJob(hostID, job.Spec); e == nil {
		t.Fatal("generic version-only submitter was widened")
	}
}
func TestRunnerDiagnosticStaleAndUnknownHostFailClosed(t *testing.T) {
	t.Setenv("WORKBENCH_HOST_BRIDGE_STATE_DIR", t.TempDir())
	if _, e := SubmitWindowsRunnerInspection("windows_unknown_fixture"); e == nil {
		t.Fatal("unknown host accepted")
	}
	hostID := "windows_diagnostic_fixture"
	hb := HostBridgeHeartbeat{HostID: hostID, Label: "fixture", Platform: "windows", Arch: "amd64", Capabilities: map[string]HostCapability{HostBridgeToolRunnerDiagnostic: {Installed: true, Version: "1"}}}
	host, e := RecordHostBridgeHeartbeat(hb)
	if e != nil {
		t.Fatal(e)
	}
	host.LastSeen = time.Now().Add(-3 * time.Minute).UTC().Format(time.RFC3339Nano)
	if e := withHostBridgeLock(func(root string) error {
		return writeHostBridgeJSON(filepath.Join(root, "hosts", hostID+".json"), host)
	}); e != nil {
		t.Fatal(e)
	}
	if _, e := SubmitWindowsRunnerInspection(hostID); e == nil {
		t.Fatal("stale capability accepted")
	}
	root, _ := hostBridgeRoot()
	entries, _ := os.ReadDir(filepath.Join(root, "jobs"))
	if len(entries) != 0 {
		t.Fatal("failure queued a job")
	}
}
