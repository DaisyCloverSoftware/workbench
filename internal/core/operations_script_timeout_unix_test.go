//go:build !windows

package core

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunOperationsScriptTimeoutStopsDescendantProcessGroup(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash unavailable")
	}

	repo := newOperationsScriptTestRepo(t)
	script := filepath.Join(repo, "scripts", "ops", "timeout-descendant.sh")
	if err := os.MkdirAll(filepath.Dir(script), 0o755); err != nil {
		t.Fatal(err)
	}
	body := `#!/usr/bin/env bash
set -euo pipefail
marker="$1"
(
  sleep 2
  printf 'descendant survived timeout\n' > "$marker"
) &
wait
`
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	gitTest(t, repo, "add", "scripts/ops/timeout-descendant.sh")
	gitTest(t, repo, "commit", "-m", "add timeout descendant operation")

	marker := filepath.Join(t.TempDir(), "descendant-marker")
	_, err := RunOperationsScript(context.Background(), repo, OperationsScriptRequest{
		Path:           "scripts/ops/timeout-descendant.sh",
		Args:           []string{marker},
		TimeoutSeconds: 1,
	})
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("operations script should time out, got %v", err)
	}

	// The historical failure killed only bash. Its background descendant kept
	// the operation stdout/stderr pipes open, survived the configured timeout,
	// and wrote this marker before cmd.Wait could return. Give that escaped
	// child enough time to prove it was terminated with the process group.
	time.Sleep(1200 * time.Millisecond)
	if _, statErr := os.Stat(marker); statErr == nil {
		t.Fatal("operations script descendant survived timeout and wrote its marker")
	} else if !errors.Is(statErr, os.ErrNotExist) {
		t.Fatal(statErr)
	}
}
