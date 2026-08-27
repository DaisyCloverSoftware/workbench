package core

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRelayInstallerAddsAdaptiveProgressWatchdog(t *testing.T) {
	text := installerScript(t, "install-github-relay.sh")
	for _, want := range []string{
		"WORKBENCH_RELAY_PROGRESS_FILE",
		"WORKBENCH_RELAY_PROGRESS_INTERVAL",
		"workbench-github-relay-watchdog.service",
		"workbench-github-relay-watchdog.timer",
		"OnUnitActiveSec=30s",
		"[ -s \"$progress_file\" ]",
		"systemctl --user enable --now workbench-github-relay-watchdog.timer",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("relay installer watchdog contract missing %q", want)
		}
	}
	if !strings.Contains(text, "disable --now workbench-github-relay-watchdog.timer") {
		t.Fatal("relay installer must stop the old watchdog while replacing the supervised PID")
	}
}

func TestRelayWatchdogRestartsOnlyCurrentExpiredPID(t *testing.T) {
	text := installerScript(t, filepath.Join("ops", "workbench-relay-watchdog.sh"))
	for _, want := range []string{
		`"deadline_unix"`,
		`-p MainPID --value`,
		`[ "$main_pid" = "$pid" ]`,
		`[ "$now" -le "$deadline" ]`,
		`systemctl --user is-active --quiet "$unit"`,
		`exec systemctl --user restart "$unit"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("relay watchdog safety contract missing %q", want)
		}
	}
	for _, forbidden := range []string{"pkill ", "killall ", "eval ", "sudo ", "bash -c", "sh -c"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("relay watchdog contains unsafe broad primitive %q", forbidden)
		}
	}
}
