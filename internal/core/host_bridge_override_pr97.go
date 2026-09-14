package core

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	HostBridgeOperationOverridePR97Proof = "override_pr97_native_proof_v1"
	overridePR97Capability               = "operation=" + HostBridgeOperationOverridePR97Proof
	overridePR97SourceSHA                = "bde855d7511ef45dbc9ec8aff67ebb92df25a579"
	overridePR97Repository               = "https://github.com/DaisyCloverSoftware/override.git"
	overridePR97Map                      = "/Game/Maps/ZeroDay"
	overridePR97Timeout                  = 90 * time.Minute
)

func validateOverridePR97Host(host HostBridgeHost, now time.Time) error {
	seen, err := time.Parse(time.RFC3339Nano, host.LastSeen)
	if err != nil || now.Sub(seen) < 0 || now.Sub(seen) > 2*time.Minute || !host.Online || host.Platform != HostBridgePlatformWindows || host.Arch != "amd64" {
		return errors.New("PR97 proof requires a fresh online Windows amd64 host")
	}
	capability := host.Capabilities[HostBridgeToolWorkbench]
	supported := false
	for _, field := range strings.Fields(capability.Version) {
		if field == overridePR97Capability {
			supported = true
		}
	}
	if !capability.Installed || !supported {
		return errors.New("PR97 proof NOT queued: installed Windows client lacks the sealed PR97 handler; reviewed client rollout is required")
	}
	engine := host.Capabilities[HostBridgeToolUnreal]
	if !engine.Installed || engine.Version != "Unreal Engine 5.8.1" {
		return errors.New("PR97 proof requires Unreal Engine 5.8.1")
	}
	return nil
}

// SubmitOverridePR97ProofJob can select only an authenticated, registered host.
// Repository, source SHA, recipe, engine association, workspace and map are sealed.
// No new generic relay control, command, script or filesystem parameter is added.
func SubmitOverridePR97ProofJob(hostID string) (HostJob, error) {
	hostID, err := validateHostBridgeID(hostID)
	if err != nil || !strings.HasPrefix(hostID, "windows_") {
		return HostJob{}, errors.New("PR97 proof requires a Windows host id")
	}
	var job HostJob
	err = withHostBridgeLock(func(root string) error {
		var host HostBridgeHost
		if err := readHostBridgeJSON(filepath.Join(root, "hosts", hostID+".json"), &host); err != nil {
			return err
		}
		if host.HostID != hostID {
			return errors.New("PR97 host identity mismatch")
		}
		if err := validateOverridePR97Host(host, time.Now().UTC()); err != nil {
			return err
		}
		entries, err := os.ReadDir(filepath.Join(root, "jobs"))
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			var existing HostJob
			if err := readHostBridgeJSON(filepath.Join(root, "jobs", entry.Name()), &existing); err != nil {
				return err
			}
			if existing.HostID == hostID && (existing.Status == "queued" || existing.Status == "claimed") {
				if existing.Spec.Tool == HostBridgeToolWorkbench && existing.Spec.Operation == HostBridgeOperationOverridePR97Proof {
					job = existing // Idempotent submission: do not launch a second cook.
					return nil
				}
				return errors.New("PR97 proof not queued: host has another unfinished job")
			}
		}
		id, err := newHostJobID()
		if err != nil {
			return err
		}
		now := time.Now().UTC().Format(time.RFC3339Nano)
		job = HostJob{ID: id, HostID: hostID, Spec: HostJobSpec{Tool: HostBridgeToolWorkbench, Operation: HostBridgeOperationOverridePR97Proof}, Status: "queued", CreatedAt: now, UpdatedAt: now}
		return writeHostBridgeJSON(filepath.Join(root, "jobs", id+".json"), job)
	})
	return job, err
}

type overridePR97NativeReport struct {
	Project    string `json:"project"`
	Map        string `json:"map"`
	UnrealRoot string `json:"unrealRoot"`
	Success    bool   `json:"success"`
	Error      string `json:"error"`
	Build      struct {
		Skipped  bool `json:"skipped"`
		ExitCode *int `json:"exitCode"`
	} `json:"build"`
	Automation struct {
		Skipped  bool `json:"skipped"`
		ExitCode *int `json:"exitCode"`
	} `json:"automation"`
}

type overridePR97PackageReport struct {
	Project    string `json:"project"`
	Map        string `json:"map"`
	UnrealRoot string `json:"unrealRoot"`
	Success    bool   `json:"success"`
	Error      string `json:"error"`
	Package    struct {
		ExitCode   *int   `json:"exitCode"`
		Executable string `json:"executable"`
	} `json:"package"`
	RuntimeProof struct {
		MapReachedPlay bool `json:"mapReachedPlay"`
	} `json:"runtimeProof"`
}

func decodeOverridePR97NativeReport(raw []byte) (overridePR97NativeReport, error) {
	var report overridePR97NativeReport
	if len(raw) == 0 || len(raw) > 1<<20 {
		return report, errors.New("PR97 native evidence missing or oversized")
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		return report, err
	}
	if !report.Success || report.Error != "" || report.Map != overridePR97Map || report.Build.Skipped || report.Automation.Skipped || report.Build.ExitCode == nil || *report.Build.ExitCode != 0 || report.Automation.ExitCode == nil || *report.Automation.ExitCode != 0 {
		return report, errors.New("PR97 native evidence does not prove Editor compile and automation")
	}
	return report, nil
}
func decodeOverridePR97PackageReport(raw []byte) (overridePR97PackageReport, error) {
	var report overridePR97PackageReport
	if len(raw) == 0 || len(raw) > 1<<20 {
		return report, errors.New("PR97 packaged evidence missing or oversized")
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		return report, err
	}
	if !report.Success || report.Error != "" || report.Map != overridePR97Map || report.Package.ExitCode == nil || *report.Package.ExitCode != 0 || !report.RuntimeProof.MapReachedPlay || report.Package.Executable == "" {
		return report, errors.New("PR97 packaged evidence does not prove Win64 packaging and ZeroDay reaching play")
	}
	return report, nil
}

// Validate every line without keeping the project's LFS inventory in memory.
// git lfs ls-files marks materialised content with '*', pointers with '-'.
type overridePR97LFSCapture struct {
	line  []byte
	count int
	bad   bool
}

func (c *overridePR97LFSCapture) Write(p []byte) (int, error) {
	for _, b := range p {
		if b == '\n' {
			c.checkLine()
			continue
		}
		if len(c.line) >= 4096 {
			c.bad = true
			continue
		}
		c.line = append(c.line, b)
	}
	return len(p), nil
}
func (c *overridePR97LFSCapture) checkLine() {
	line := strings.TrimSpace(string(c.line))
	c.line = c.line[:0]
	if line == "" {
		return
	}
	fields := strings.Fields(line)
	if len(fields) < 3 || fields[1] != "*" {
		c.bad = true
	}
	c.count++
}
func (c *overridePR97LFSCapture) complete() bool {
	if len(c.line) > 0 {
		c.checkLine()
	}
	return c.count > 0 && !c.bad
}
