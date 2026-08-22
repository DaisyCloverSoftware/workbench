package core

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkbenchExecutableIdentityIsBoundedAndPathFree(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Workbench.exe")
	content := []byte("synthetic workbench executable bytes")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}

	identity, err := workbenchExecutableIdentity(path, "0.9.56-test")
	if err != nil {
		t.Fatal(err)
	}
	wantHash := sha256.Sum256(content)
	want := fmt.Sprintf("Workbench 0.9.56-test sha256=%x bytes=%d", wantHash, len(content))
	if identity != want {
		t.Fatalf("identity=%q want=%q", identity, want)
	}
	if strings.Contains(identity, path) || strings.Contains(identity, filepath.Dir(path)) {
		t.Fatalf("identity leaked executable path: %q", identity)
	}
}

func TestWorkbenchIdentityHeartbeatIsReadOnlyAndAllowlisted(t *testing.T) {
	heartbeat, err := sanitizeHostHeartbeat(HostBridgeHeartbeat{
		HostID:   "windows_identitytest",
		Label:    "Identity test host",
		Platform: HostBridgePlatformWindows,
		Arch:     "amd64",
		Capabilities: map[string]HostCapability{
			HostBridgeToolWorkbench: {Installed: true, Version: "Workbench 0.9.56 sha256=0123456789abcdef bytes=123"},
			"powershell":            {Installed: true, Version: "7.5"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := heartbeat.Capabilities[HostBridgeToolWorkbench]; !ok || !got.Installed || !strings.HasPrefix(got.Version, "Workbench ") {
		t.Fatalf("Workbench identity capability was not preserved: %#v", heartbeat.Capabilities)
	}
	if _, ok := heartbeat.Capabilities["powershell"]; ok {
		t.Fatalf("unknown capability escaped allowlist: %#v", heartbeat.Capabilities)
	}
	if _, err := validateHostJobSpec(HostJobSpec{Tool: HostBridgeToolWorkbench, Operation: HostBridgeOperationVersion}); err == nil {
		t.Fatal("Workbench heartbeat capability unexpectedly gained executable job authority")
	}
}

func TestWorkbenchExecutableIdentityRejectsSymlink(t *testing.T) {
	target := filepath.Join(t.TempDir(), "Workbench.exe")
	if err := os.WriteFile(target, []byte("synthetic executable"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(filepath.Dir(target), "Workbench-link.exe")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink not supported in test environment: %v", err)
	}
	if _, err := workbenchExecutableIdentity(link, "0.9.56-test"); err == nil {
		t.Fatal("symlink executable identity was accepted")
	}
}
