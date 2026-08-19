package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNamespaceHealthOperationIsReadOnlyAndBounded(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(here), "..", ".."))
	path := filepath.Join(root, "scripts", "ops", "namespace-health.sh")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)

	for _, want := range []string{
		"NAMESPACE_HEALTH_SELF_TEST_OK",
		"NAMESPACE_HEALTH",
		"get deployments",
		"get statefulsets",
		"get pods",
		"get jobs",
		"get persistentvolumeclaims",
		"get events",
		"tail -n 12",
		"snapshot=ok",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("namespace health operation missing %q", want)
		}
	}

	for _, forbidden := range []string{
		" get secret",
		" get secrets",
		" apply ",
		" delete ",
		" patch ",
		" exec ",
		" rollout ",
		" scale ",
		" set image",
		"helm ",
		"docker ",
		"bash -c",
		"sh -c",
	} {
		if strings.Contains(strings.ToLower(text), forbidden) {
			t.Fatalf("namespace health operation contains forbidden mutation/secret pattern %q", forbidden)
		}
	}

	if runtime.GOOS != "windows" {
		if _, err := exec.LookPath("bash"); err == nil {
			for _, args := range [][]string{{"-n", path}, {path, "--self-test"}} {
				cmd := exec.Command("bash", args...)
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("namespace health operation %v failed: %v: %s", args, err, out)
				}
			}
		}
	}
}
