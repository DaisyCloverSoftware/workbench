package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const privateUpdateDelay = 8 * time.Second

// schedulePrivateWorkbenchUpdate deliberately accepts no remote command or URL
// from the control envelope. It reuses only the already-configured source tree
// and relay origin, then schedules the existing fail-closed bootstrap outside
// the relay process so restarting the relay cannot kill the updater itself.
func schedulePrivateWorkbenchUpdate(relayRepo string) (map[string]any, error) {
	relayRepo = strings.TrimSpace(relayRepo)
	if relayRepo == "" {
		return nil, errors.New("private update requires the configured relay repository")
	}

	sourceDir, err := privateUpdateSourceDir()
	if err != nil {
		return nil, err
	}
	script := filepath.Join(sourceDir, "scripts", "bootstrap-private-relay.sh")
	if st, statErr := os.Stat(script); statErr != nil || st.IsDir() {
		return nil, fmt.Errorf("Workbench update bootstrap is unavailable at configured source tree")
	}

	remoteOut, err := exec.Command("git", "-C", relayRepo, "remote", "get-url", "origin").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("read configured relay origin: %s", strings.TrimSpace(string(remoteOut)))
	}
	remote := strings.TrimSpace(string(remoteOut))
	if remote == "" {
		return nil, errors.New("configured relay origin is empty")
	}
	if _, err := exec.LookPath("systemd-run"); err != nil {
		return nil, errors.New("private self-update requires systemd-run on this host")
	}

	unit := fmt.Sprintf("workbench-private-update-%d", time.Now().UTC().UnixNano())
	cmd := exec.Command(
		"systemd-run", "--user", "--quiet", "--collect",
		"--unit", unit,
		"--on-active", privateUpdateDelay.String(),
		"/bin/bash", script, remote,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("schedule Workbench private update: %s", strings.TrimSpace(string(out)))
	}
	return map[string]any{
		"scheduled":     true,
		"delay_seconds": int(privateUpdateDelay / time.Second),
		"instruction":   "Workbench will fast-forward, test, rebuild and restart the private loop using its configured repositories.",
	}, nil
}

func privateUpdateSourceDir() (string, error) {
	if configured := strings.TrimSpace(os.Getenv("WORKBENCH_SOURCE_DIR")); configured != "" {
		return filepath.Abs(configured)
	}
	if root := strings.TrimSpace(os.Getenv("WORKBENCH_RUNNER_ROOT")); root != "" {
		return filepath.Abs(filepath.Join(root, "workbench"))
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "src", "workbench"), nil
}
