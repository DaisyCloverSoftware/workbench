package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func pr97ReadyHost() HostBridgeHost {
	return HostBridgeHost{HostID: "windows_pr97_testhost", Platform: "windows", Arch: "amd64", Online: true, LastSeen: time.Now().UTC().Format(time.RFC3339Nano), Capabilities: map[string]HostCapability{
		HostBridgeToolWorkbench: {Installed: true, Version: "Workbench test " + overridePR97Capability},
		HostBridgeToolUnreal:    {Installed: true, Version: "Unreal Engine 5.8.1"},
	}}
}
func TestOverridePR97HostBoundary(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*HostBridgeHost)
	}{
		{"old client", func(h *HostBridgeHost) {
			h.Capabilities[HostBridgeToolWorkbench] = HostCapability{Installed: true, Version: "Workbench 0.9.63"}
		}},
		{"marker substring", func(h *HostBridgeHost) {
			h.Capabilities[HostBridgeToolWorkbench] = HostCapability{Installed: true, Version: overridePR97Capability + "_other"}
		}},
		{"wrong engine", func(h *HostBridgeHost) {
			h.Capabilities[HostBridgeToolUnreal] = HostCapability{Installed: true, Version: "Unreal Engine 5.8.2"}
		}},
		{"stale host", func(h *HostBridgeHost) { h.LastSeen = time.Now().Add(-3 * time.Minute).UTC().Format(time.RFC3339Nano) }},
		{"future host", func(h *HostBridgeHost) { h.LastSeen = time.Now().Add(time.Minute).UTC().Format(time.RFC3339Nano) }},
		{"offline", func(h *HostBridgeHost) { h.Online = false }},
		{"wrong OS", func(h *HostBridgeHost) { h.Platform = "linux" }},
		{"wrong arch", func(h *HostBridgeHost) { h.Arch = "arm64" }},
	}
	if err := validateOverridePR97Host(pr97ReadyHost(), time.Now()); err != nil {
		t.Fatal(err)
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			host := pr97ReadyHost()
			test.mutate(&host)
			if validateOverridePR97Host(host, time.Now()) == nil {
				t.Fatal("unsafe host accepted")
			}
		})
	}
}
func TestOverridePR97Submission(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WORKBENCH_HOST_BRIDGE_STATE_DIR", root)
	host := pr97ReadyHost()
	err := withHostBridgeLock(func(root string) error {
		return writeHostBridgeJSON(filepath.Join(root, "hosts", host.HostID+".json"), host)
	})
	if err != nil {
		t.Fatal(err)
	}
	first, err := SubmitOverridePR97ProofJob(host.HostID)
	if err != nil {
		t.Fatal(err)
	}
	second, err := SubmitOverridePR97ProofJob(host.HostID)
	if err != nil || second.ID != first.ID {
		t.Fatal("submission is not idempotent", err)
	}
	raw, _ := json.Marshal(first.Spec)
	if string(raw) != `{"tool":"workbench","operation":"override_pr97_native_proof_v1"}` {
		t.Fatal("operation carries variable build inputs", string(raw))
	}
	claimed, err := ClaimHostBridgeJob(host.HostID)
	if err != nil || claimed == nil {
		t.Fatal(err)
	}
	expiry, _ := time.Parse(time.RFC3339Nano, claimed.ClaimExpiresAt)
	if time.Until(expiry) < overridePR97Timeout {
		t.Fatal("claim expires before the bounded build")
	}
	if _, err := SubmitOverridePR97ProofJob("../escape"); err == nil {
		t.Fatal("path-shaped host accepted")
	}
	entries, _ := os.ReadDir(filepath.Join(root, "jobs"))
	if len(entries) != 1 {
		t.Fatal("duplicate jobs written")
	}
}
func TestOverridePR97EvidenceBoundary(t *testing.T) {
	native := `{"success":true,"map":"/Game/Maps/ZeroDay","build":{"skipped":false,"exitCode":0},"automation":{"skipped":false,"exitCode":0}}`
	packaged := `{"success":true,"map":"/Game/Maps/ZeroDay","package":{"exitCode":0,"executable":"OverrideZeroDay.exe"},"runtimeProof":{"mapReachedPlay":true}}`
	if _, err := decodeOverridePR97NativeReport([]byte(native)); err != nil {
		t.Fatal(err)
	}
	if _, err := decodeOverridePR97PackageReport([]byte(packaged)); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{"{}", strings.Replace(native, `"success":true`, `"success":false`, 1), strings.Replace(native, `"skipped":false`, `"skipped":true`, 1), strings.Replace(native, `"exitCode":0`, `"exitCode":null`, 1), strings.Replace(native, `"exitCode":0`, `"exitCode":1`, 1), strings.Replace(native, "ZeroDay", "OtherMap", 1)} {
		if _, err := decodeOverridePR97NativeReport([]byte(raw)); err == nil {
			t.Fatal("incomplete native evidence accepted", raw)
		}
	}
	for _, raw := range []string{"{}", strings.Replace(packaged, `"mapReachedPlay":true`, `"mapReachedPlay":false`, 1), strings.Replace(packaged, `"exitCode":0`, `"exitCode":null`, 1), strings.Replace(packaged, "ZeroDay", "OtherMap", 1), strings.Replace(packaged, `"executable":"OverrideZeroDay.exe"`, `"executable":""`, 1)} {
		if _, err := decodeOverridePR97PackageReport([]byte(raw)); err == nil {
			t.Fatal("incomplete packaged evidence accepted", raw)
		}
	}
}
func TestOverridePR97LFSBoundary(t *testing.T) {
	for _, test := range []struct {
		data string
		good bool
	}{
		{"0123456789 * Content/A asset.uasset\n", true},
		{"0123456789 * Content/A.uasset\r\nabcdef0123 * Content/B.uasset", true},
		{"0123456789 - Content/A.uasset\n", false}, {"", false}, {"malformed\n", false},
		{strings.Repeat("x", 4097) + " * Content/A.uasset\n", false},
	} {
		capture := &overridePR97LFSCapture{}
		for _, b := range []byte(test.data) {
			_, _ = capture.Write([]byte{b})
		}
		if capture.complete() != test.good {
			t.Fatal("incorrect LFS materialisation decision")
		}
	}
}
