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
}

func TestClusterInstallersDoNotPinServicesToLegacySrcRoot(t *testing.T) {
	for _, script := range []string{"install-cluster-mcp.sh", "install-github-relay.sh"} {
		text := installerScript(t, script)
		if strings.Contains(text, "Environment=WORKBENCH_RUNNER_ROOT=$HOME/src") {
			t.Fatalf("%s must not force systemd back to the legacy single ~/src root", script)
		}
	}
}
