package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRelayInstallerRestartsExistingService(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Clean(filepath.Join(wd, "..", "..", "scripts", "install-github-relay.sh"))
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "systemctl --user restart workbench-github-relay.service") {
		t.Fatal("relay installer must restart an already-running service after rewriting its unit")
	}
}
