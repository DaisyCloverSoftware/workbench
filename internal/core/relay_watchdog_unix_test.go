//go:build !windows

package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRelayWatchdogRestartsExpiredCurrentPIDButNotStalePID(t *testing.T) {
	script := filepath.Clean(filepath.Join("..", "..", "scripts", "ops", "workbench-relay-watchdog.sh"))
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash unavailable")
	}

	fakeDir := t.TempDir()
	logPath := filepath.Join(fakeDir, "systemctl.log")
	fakeSystemctl := filepath.Join(fakeDir, "systemctl")
	body := `#!/bin/sh
case "$*" in
  "--user is-active --quiet workbench-github-relay.service") exit 0 ;;
  "--user show workbench-github-relay.service -p MainPID --value") printf '%s\n' "$FAKE_MAIN_PID" ;;
  "--user restart workbench-github-relay.service") printf 'restart\n' >> "$FAKE_SYSTEMCTL_LOG" ;;
  *) printf 'unexpected systemctl args: %s\n' "$*" >&2; exit 2 ;;
esac
`
	if err := os.WriteFile(fakeSystemctl, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}

	statePath := filepath.Join(fakeDir, "progress.json")
	run := func(statePID int) {
		t.Helper()
		state := fmt.Sprintf(`{"version":1,"pid":%d,"phase":"control-execute","updated_unix":%d,"deadline_unix":%d}`+"\n", statePID, time.Now().Add(-time.Minute).Unix(), time.Now().Add(-time.Second).Unix())
		if err := os.WriteFile(statePath, []byte(state), 0o600); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("bash", script)
		cmd.Env = append(os.Environ(),
			"PATH="+fakeDir+":/usr/bin:/bin",
			"FAKE_MAIN_PID=4242",
			"FAKE_SYSTEMCTL_LOG="+logPath,
			"WORKBENCH_RELAY_PROGRESS_FILE="+statePath,
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("watchdog failed: %v: %s", err, out)
		}
	}

	run(4242)
	b, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(b)) != "restart" {
		t.Fatalf("expired current PID did not restart exactly once: %q", b)
	}

	if err := os.WriteFile(logPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	run(4243)
	b, err = os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) != 0 {
		t.Fatalf("stale PID restarted current relay: %q", b)
	}
}
