package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

var workbenchUpdateUnits = []string{
	"workbench-mcp.service",
	"workbench-github-relay.service",
}

type clusterUpdateReport struct {
	OK                bool     `json:"ok"`
	CurrentVersion    string   `json:"current_version"`
	LatestVersion     string   `json:"latest_version"`
	UpdateAvailable   bool     `json:"update_available"`
	Applied           bool     `json:"applied"`
	ArchiveSHA256     string   `json:"archive_sha256,omitempty"`
	RestartedServices []string `json:"restarted_services,omitempty"`
	Message           string   `json:"message,omitempty"`
}

func update() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: workbench-runner update <check|apply>")
		os.Exit(2)
	}
	action := strings.ToLower(strings.TrimSpace(os.Args[2]))
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	switch action {
	case "check":
		check, err := core.CheckOfficialUpdate(ctx, core.Version)
		if err != nil {
			write(map[string]any{"ok": false, "error": err.Error()})
			os.Exit(1)
		}
		write(map[string]any{"ok": true, "update": check})
	case "apply":
		report, err := applyClusterUpdate(ctx)
		if err != nil {
			report.OK = false
			write(map[string]any{"ok": false, "update": report, "error": err.Error()})
			os.Exit(1)
		}
		report.OK = true
		write(report)
	default:
		fmt.Fprintln(os.Stderr, "usage: workbench-runner update <check|apply>")
		os.Exit(2)
	}
}

func applyClusterUpdate(ctx context.Context) (clusterUpdateReport, error) {
	report := clusterUpdateReport{CurrentVersion: core.Version}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return report, errors.New("cluster self-update apply is supported on Linux amd64 hosts only")
	}
	lock, err := core.AcquireUpdateLock()
	if err != nil {
		return report, err
	}
	defer lock.Close()

	home, err := os.UserHomeDir()
	if err != nil {
		return report, err
	}
	binDir := filepath.Join(home, ".local", "bin")
	if err := verifyInstalledRunnerIdentity(filepath.Join(binDir, "workbench-runner")); err != nil {
		return report, err
	}
	if err := requireUserSystemd(ctx); err != nil {
		return report, err
	}
	activeUnits, err := activeWorkbenchUpdateUnits(ctx)
	if err != nil {
		return report, err
	}

	check, bundle, err := core.PrepareOfficialClusterUpdate(ctx, core.Version)
	if err != nil {
		return report, err
	}
	report.LatestVersion = check.LatestVersion
	report.UpdateAvailable = check.UpdateAvailable
	if !check.UpdateAvailable {
		report.Message = "Workbench is already at the latest stable release."
		return report, nil
	}
	defer bundle.Cleanup()
	report.ArchiveSHA256 = bundle.ArchiveSHA256

	replacements, err := core.ClusterBinaryReplacements(bundle, binDir)
	if err != nil {
		return report, err
	}
	tx, err := core.BeginBinaryInstall(replacements)
	if err != nil {
		return report, err
	}
	rollback := true
	defer func() {
		if rollback {
			_ = tx.Rollback()
		}
	}()

	runnerPath := filepath.Join(binDir, "workbench-runner")
	if err := verifyNewRunner(ctx, runnerPath, bundle.Version); err != nil {
		_ = tx.Rollback()
		rollback = false
		return report, err
	}

	restarted, restartErr := restartActiveWorkbenchUnits(ctx, activeUnits)
	if restartErr != nil {
		rollbackErr := tx.Rollback()
		rollback = false
		restoreErr := restartOldWorkbenchUnits(ctx, activeUnits)
		return report, errors.Join(restartErr, rollbackErr, restoreErr)
	}
	report.RestartedServices = restarted

	if err := tx.Commit(); err != nil {
		rollback = false
		report.Applied = true
		report.Message = "Workbench updated successfully, but old binary backup cleanup reported an error."
		return report, fmt.Errorf("update installed but backup cleanup failed: %w", err)
	}
	rollback = false
	report.Applied = true
	report.Message = "Workbench cluster binaries updated from the verified official release."
	return report, nil
}

func verifyInstalledRunnerIdentity(target string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exeInfo, err := os.Stat(exe)
	if err != nil {
		return err
	}
	targetInfo, err := os.Lstat(target)
	if err != nil {
		return fmt.Errorf("installed Workbench runner not found at %s: %w", target, err)
	}
	if targetInfo.Mode()&os.ModeSymlink != 0 || !targetInfo.Mode().IsRegular() {
		return fmt.Errorf("installed Workbench runner is not a regular file: %s", target)
	}
	targetStat, err := os.Stat(target)
	if err != nil {
		return err
	}
	if !os.SameFile(exeInfo, targetStat) {
		return fmt.Errorf("update apply must be run from the installed Workbench runner at %s", target)
	}
	return nil
}

func requireUserSystemd(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "systemctl", "--user", "show-environment")
	configureRunnerChild(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("Workbench cluster self-update requires a usable systemd user session; fallback-process installs are not updated automatically: %s", safeCommandFailure(out, err))
	}
	return nil
}

func activeWorkbenchUpdateUnits(ctx context.Context) ([]string, error) {
	var active []string
	for _, unit := range workbenchUpdateUnits {
		cmd := exec.CommandContext(ctx, "systemctl", "--user", "is-active", "--quiet", unit)
		configureRunnerChild(cmd)
		err := cmd.Run()
		if err == nil {
			active = append(active, unit)
			continue
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// systemctl returns a non-zero status for inactive and unknown units.
			// The updater never starts a unit that was not active before the swap.
			continue
		}
		return nil, fmt.Errorf("inspect Workbench service %s: %w", unit, err)
	}
	return active, nil
}

func verifyNewRunner(ctx context.Context, runnerPath, wantVersion string) error {
	versionCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(versionCtx, runnerPath, "version")
	configureRunnerChild(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("verify updated Workbench runner version: %s", safeCommandFailure(out, err))
	}
	if strings.TrimSpace(string(out)) != wantVersion {
		return fmt.Errorf("updated Workbench runner reports %q, expected %q", strings.TrimSpace(string(out)), wantVersion)
	}

	selfCtx, selfCancel := context.WithTimeout(ctx, 3*time.Minute)
	defer selfCancel()
	cmd = exec.CommandContext(selfCtx, runnerPath, "selftest")
	configureRunnerChild(cmd)
	out, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("updated Workbench runner selftest failed: %s", safeCommandFailure(out, err))
	}
	if !bytes.Contains(out, []byte("SELFTEST PASSED")) {
		return errors.New("updated Workbench runner selftest did not report success")
	}
	return nil
}

func restartActiveWorkbenchUnits(ctx context.Context, units []string) ([]string, error) {
	var restarted []string
	for _, unit := range units {
		if err := restartAndVerifyUnit(ctx, unit); err != nil {
			return restarted, err
		}
		restarted = append(restarted, unit)
	}
	return restarted, nil
}

func restartOldWorkbenchUnits(ctx context.Context, units []string) error {
	var errs []error
	for _, unit := range units {
		if err := restartAndVerifyUnit(ctx, unit); err != nil {
			errs = append(errs, fmt.Errorf("restore %s after binary rollback: %w", unit, err))
		}
	}
	return errors.Join(errs...)
}

func restartAndVerifyUnit(ctx context.Context, unit string) error {
	restartCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(restartCtx, "systemctl", "--user", "restart", unit)
	configureRunnerChild(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("restart %s: %s", unit, safeCommandFailure(out, err))
	}

	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		cmd = exec.CommandContext(restartCtx, "systemctl", "--user", "is-active", "--quiet", unit)
		configureRunnerChild(cmd)
		if cmd.Run() == nil {
			return nil
		}
		time.Sleep(250 * time.Millisecond)
	}
	return fmt.Errorf("Workbench service did not become active after restart: %s", unit)
}

func safeCommandFailure(out []byte, err error) string {
	text := strings.TrimSpace(string(out))
	if len(text) > 1024 {
		text = text[:1024] + "…"
	}
	if text != "" {
		return text
	}
	if err != nil {
		return err.Error()
	}
	return "unknown failure"
}

func configureRunnerChild(cmd *exec.Cmd) {
	// Keep updater commands non-interactive and predictable. The core process
	// helper is intentionally not exported; these commands do not invoke a shell.
	cmd.Stdin = nil
	cmd.Env = append(os.Environ(), "SYSTEMD_PAGER=cat", "PAGER=cat")
}
