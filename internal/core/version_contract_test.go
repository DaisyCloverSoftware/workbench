package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseVersionContract(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	checks := map[string]string{
		filepath.Join(root, ".codex-plugin", "plugin.json"):          `"version": "` + Version + `"`,
		filepath.Join(root, "cmd", "workbench", "main_windows.go"):  `const appVersion = "` + Version + `"`,
		filepath.Join(root, "cmd", "workbench-runner", "main.go"):   `const runnerVersion = "` + Version + `"`,
		filepath.Join(root, "cmd", "workbench-server", "main.go"):   `const serverVersion = "` + Version + `"`,
		filepath.Join(root, "cmd", "workbench-relay", "main.go"):    `const relayVersion = "` + Version + `"`,
		filepath.Join(root, "internal", "mcp", "server.go"):          `"version": "` + Version + `"`,
		filepath.Join(root, "CHANGELOG.md"):                            "## " + Version,
	}
	for path, want := range checks {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if !strings.Contains(string(b), want) {
			t.Fatalf("%s does not advertise Workbench %s; expected %q", path, Version, want)
		}
	}
}
