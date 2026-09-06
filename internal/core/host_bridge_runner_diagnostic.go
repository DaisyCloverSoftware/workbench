package core

import (
	"errors"
	"path/filepath"
	"time"
)

const HostBridgeToolRunnerDiagnostic = "actions_runner_diagnostic"
const HostBridgeOperationRunnerInspect = "inspect_v1"

// SubmitWindowsRunnerInspection is deliberately separate from version-only
// generic submissions. No executable, path, URL, service or arguments accepted.
func SubmitWindowsRunnerInspection(hostID string) (HostJob, error) {
	hostID, err := validateHostBridgeID(hostID)
	if err != nil {
		return HostJob{}, err
	}
	id, err := newHostJobID()
	if err != nil {
		return HostJob{}, err
	}
	now := time.Now().UTC()
	job := HostJob{ID: id, HostID: hostID, Spec: HostJobSpec{Tool: HostBridgeToolRunnerDiagnostic, Operation: HostBridgeOperationRunnerInspect}, Status: "queued", CreatedAt: now.Format(time.RFC3339Nano), UpdatedAt: now.Format(time.RFC3339Nano)}
	err = withHostBridgeLock(func(root string) error {
		var host HostBridgeHost
		if readHostBridgeJSON(filepath.Join(root, "hosts", hostID+".json"), &host) != nil {
			return errors.New("target host is not registered")
		}
		seen, e := time.Parse(time.RFC3339Nano, host.LastSeen)
		cap := host.Capabilities[HostBridgeToolRunnerDiagnostic]
		if e != nil || now.Sub(seen) < 0 || now.Sub(seen) > 2*time.Minute || host.Platform != HostBridgePlatformWindows || !cap.Installed || cap.Version != "1" {
			return errors.New("fresh Windows runner diagnostic v1 capability required; deploy the diagnostic client first")
		}
		return writeHostBridgeJSON(filepath.Join(root, "jobs", id+".json"), job)
	})
	return job, err
}
