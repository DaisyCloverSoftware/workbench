package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseWorkflowPackagesMaintenanceAssets(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	workflow := filepath.Join(root, ".github", "workflows", "release.yml")
	b, err := os.ReadFile(workflow)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	for _, want := range []string{
		"Workbench.exe",
		"Workbench.exe.sha256",
		"Workbench-Updater.exe",
		"Workbench-Updater.exe.sha256",
		"Workbench-Cluster-linux-amd64.zip",
		"Workbench-Cluster-linux-amd64.zip.sha256",
		"./cmd/workbench-updater",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release workflow is missing maintenance asset/build contract %q", want)
		}
	}
}
