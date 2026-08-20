package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func installerScript(t *testing.T, name string) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Clean(filepath.Join(wd, "..", "..", "scripts", name))
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestRelayInstallerRestartsExistingService(t *testing.T) {
	text := installerScript(t, "install-github-relay.sh")
	if !strings.Contains(text, "systemctl --user restart workbench-github-relay.service") {
		t.Fatal("relay installer must restart an already-running service after rewriting its unit")
	}
	if !strings.Contains(text, "systemctl --user is-active --quiet workbench-github-relay.service") {
		t.Fatal("relay installer must verify the supervised service becomes active after restart")
	}
}

func TestRelayInstallerSmokeDoesNotConsumeLiveQueue(t *testing.T) {
	text := installerScript(t, "install-github-relay.sh")
	if !strings.Contains(text, `"$bin_dir/workbench-relay" "${relay_args[@]}" --help`) {
		t.Fatal("relay installer must smoke the built binary without polling live relay work")
	}
	if strings.Contains(text, `"$bin_dir/workbench-relay" "${relay_args[@]}" --once`) {
		t.Fatal("relay installer must not poll the live queue as an updater smoke test")
	}
}

func TestDelayedRelayRestartOperationIsDetachedAndFixedTarget(t *testing.T) {
	text := installerScript(t, filepath.Join("ops", "restart-workbench-relay-delayed.sh"))
	for _, want := range []string{"systemd-run", "--on-active=60s", "delay_seconds=60", "workbench-github-relay.service"} {
		if !strings.Contains(text, want) {
			t.Fatalf("delayed relay restart operation missing %q", want)
		}
	}
	for _, forbidden := range []string{"eval ", "bash -c", "sh -c", "curl ", "wget ", "sudo "} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("delayed relay restart operation contains forbidden primitive %q", forbidden)
		}
	}
}

func TestClusterInstallersDoNotPinServicesToLegacySrcRoot(t *testing.T) {
	for _, script := range []string{"install-cluster-mcp.sh", "install-github-relay.sh"} {
		text := installerScript(t, script)
		if strings.Contains(text, "Environment=WORKBENCH_RUNNER_ROOT=$HOME/src") {
			t.Fatalf("%s must not force systemd back to the legacy single ~/src root", script)
		}
	}
}
