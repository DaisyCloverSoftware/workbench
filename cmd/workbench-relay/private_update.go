package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const privateUpdateDelay = 8 * time.Second

const (
	privateUpdateScheduled = "scheduled"
	privateUpdateRunning   = "running"
	privateUpdateSucceeded = "succeeded"
	privateUpdateFailed    = "failed"
	privateUpdateNeverRun  = "never_run"
)

type privateUpdateStatus struct {
	State     string `json:"state"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// schedulePrivateWorkbenchUpdate deliberately accepts no remote command or URL
// from the control envelope. It reuses only the already-configured source tree
// and relay origin, then schedules the fixed status-writing update helper outside
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
	helper := filepath.Join(sourceDir, "scripts", "run-private-update.sh")
	if st, statErr := os.Stat(helper); statErr != nil || st.IsDir() {
		return nil, errors.New("Workbench private update helper is unavailable at configured source tree")
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

	if err := writePrivateUpdateStatus(privateUpdateScheduled); err != nil {
		return nil, errors.New("could not persist private update schedule state")
	}
	unit := fmt.Sprintf("workbench-private-update-%d", time.Now().UTC().UnixNano())
	cmd := exec.Command(
		"systemd-run", "--user", "--quiet", "--collect",
		"--unit", unit,
		"--on-active", privateUpdateDelay.String(),
		"/bin/bash", helper, remote,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = writePrivateUpdateStatus(privateUpdateFailed)
		return nil, fmt.Errorf("schedule Workbench private update: %s", strings.TrimSpace(string(out)))
	}
	return map[string]any{
		"scheduled":     true,
		"delay_seconds": int(privateUpdateDelay / time.Second),
		"state":         privateUpdateScheduled,
		"instruction":   "Workbench will refresh from a disposable maintenance checkout, test, rebuild and restart the private loop without modifying the developer checkout.",
	}, nil
}

func privateUpdateStatusPath() (string, error) {
	if configured := strings.TrimSpace(os.Getenv("WORKBENCH_PRIVATE_UPDATE_STATUS_FILE")); configured != "" {
		return filepath.Abs(configured)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state", "workbench", "private-update-status.json"), nil
}

func writePrivateUpdateStatus(state string) error {
	if !validPrivateUpdateState(state) || state == privateUpdateNeverRun {
		return errors.New("invalid private update state")
	}
	path, err := privateUpdateStatusPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(privateUpdateStatus{State: state, UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano)}, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	f, err := os.CreateTemp(filepath.Dir(path), ".private-update-status-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		return err
	}
	if _, err := f.Write(b); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func readPrivateUpdateStatus() (map[string]any, error) {
	path, err := privateUpdateStatusPath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]any{"state": privateUpdateNeverRun}, nil
	}
	if err != nil {
		return nil, errors.New("private update status is unavailable")
	}
	if len(b) > 4096 {
		return nil, errors.New("private update status is invalid")
	}
	var status privateUpdateStatus
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&status); err != nil || !validPrivateUpdateState(status.State) || status.State == privateUpdateNeverRun {
		return nil, errors.New("private update status is invalid")
	}
	result := map[string]any{"state": status.State}
	if strings.TrimSpace(status.UpdatedAt) != "" {
		if _, err := time.Parse(time.RFC3339Nano, status.UpdatedAt); err != nil {
			return nil, errors.New("private update status timestamp is invalid")
		}
		result["updated_at"] = status.UpdatedAt
	}
	return result, nil
}

func validPrivateUpdateState(state string) bool {
	switch state {
	case privateUpdateScheduled, privateUpdateRunning, privateUpdateSucceeded, privateUpdateFailed, privateUpdateNeverRun:
		return true
	default:
		return false
	}
}

func privateUpdateSourceDir() (string, error) {
	// An explicit operator override wins. Otherwise, once the non-destructive
	// maintenance checkout exists, use its fixed helper for all future updates;
	// the developer checkout may intentionally remain dirty or on another branch.
	if configured := strings.TrimSpace(os.Getenv("WORKBENCH_SOURCE_DIR")); configured != "" {
		return filepath.Abs(configured)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	maintenance := filepath.Join(home, ".local", "share", "workbench", "update-source")
	if st, statErr := os.Stat(filepath.Join(maintenance, "scripts", "run-private-update.sh")); statErr == nil && !st.IsDir() {
		return maintenance, nil
	}
	if root := strings.TrimSpace(os.Getenv("WORKBENCH_RUNNER_ROOT")); root != "" {
		return filepath.Abs(filepath.Join(root, "workbench"))
	}
	return filepath.Join(home, "src", "workbench"), nil
}
