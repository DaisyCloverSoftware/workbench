package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWorkbenchHealthOperationIsReadOnlyAndSecretFree(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(here), "..", ".."))
	path := filepath.Join(root, "scripts", "ops", "workbench-health.sh")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)

	for _, want := range []string{
		"WORKBENCH_HEALTH",
		"workbench-mcp.service",
		"workbench-github-relay.service",
		"workbench-runner",
		"workbench-server",
		"workbench-relay",
		"mcp_http=ok",
		"relay_checkout=clean",
		"overall=ok",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("health operation missing %q", want)
		}
	}

	for _, forbidden := range []string{
		"mcp-loopback-auth-value",
		"Authorization:",
		"systemctl --user restart",
		"systemctl --user stop",
		"systemctl --user start",
		"sudo ",
		"kubectl ",
		"helm ",
		"docker ",
		"rm -",
		"bash -c",
		"sh -c",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("health operation contains forbidden mutating/secret pattern %q", forbidden)
		}
	}

	if runtime.GOOS != "windows" {
		if _, err := exec.LookPath("bash"); err == nil {
			cmd := exec.Command("bash", "-n", path)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("health operation failed bash -n: %v: %s", err, out)
			}
		}
	}
}
