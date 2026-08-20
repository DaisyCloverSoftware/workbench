//go:build windows

package core

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const windowsUpdaterStateFile = ".workbench-updater-verified.json"

type windowsUpdaterVerifiedState struct {
	CheckedByWorkbenchVersion string `json:"checked_by_workbench_version"`
	ReleaseVersion            string `json:"release_version"`
	SHA256                    string `json:"sha256"`
}

// RefreshVerifiedWindowsUpdater keeps the separately shipped updater binary on
// the same verified release channel as Workbench.exe. This is intentionally
// performed by Workbench rather than by the updater process itself: a running
// executable cannot reliably replace its own file on Windows.
//
// A small local state record avoids downloading the updater on every launch.
// The record is accepted only for the exact Workbench version that performed the
// verified check and only while the installed updater still matches the recorded
// SHA-256. Every Workbench version therefore revalidates/refreshed its updater at
// least once, and local tampering forces another official-release verification.
func RefreshVerifiedWindowsUpdater(ctx context.Context, installDir string) (bool, error) {
	installDir = filepath.Clean(strings.TrimSpace(installDir))
	if installDir == "" || installDir == "." {
		return false, errors.New("Windows updater install directory is required")
	}
	target := filepath.Join(installDir, WindowsUpdaterReleaseAsset)
	if windowsUpdaterStateMatches(installDir, target) {
		return false, nil
	}

	release, err := FetchOfficialLatestReleaseResilient(ctx)
	if err != nil {
		return false, err
	}
	asset, err := DownloadVerifiedReleaseAsset(ctx, release, WindowsUpdaterReleaseAsset, WindowsUpdaterReleaseChecksumAsset, maxWindowsAppBytes)
	if err != nil {
		return false, err
	}
	defer os.RemoveAll(filepath.Dir(asset.Path))
	if err := VerifyWindowsAMD64PE(asset.Path); err != nil {
		return false, err
	}

	matches, matchErr := WindowsAppMatchesVerifiedAsset(target, asset)
	if matchErr == nil && matches {
		if err := writeWindowsUpdaterState(installDir, release.Version, asset.SHA256); err != nil {
			return false, err
		}
		return false, nil
	}
	if matchErr != nil && !errors.Is(matchErr, os.ErrNotExist) {
		if info, statErr := os.Lstat(target); statErr != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return false, matchErr
		}
	}

	var lastErr error
	for {
		tx, installErr := BeginWindowsUpdaterInstall(asset, target)
		if installErr == nil {
			if commitErr := tx.Commit(); commitErr != nil {
				return true, commitErr
			}
			if err := writeWindowsUpdaterState(installDir, release.Version, asset.SHA256); err != nil {
				return true, err
			}
			return true, nil
		}
		lastErr = installErr
		if !isTransientWindowsUpdaterReplacementError(installErr) {
			return false, installErr
		}
		select {
		case <-ctx.Done():
			return false, errors.Join(ctx.Err(), lastErr)
		case <-time.After(2 * time.Second):
		}
	}
}

func windowsUpdaterStateMatches(installDir, target string) bool {
	statePath := filepath.Join(installDir, windowsUpdaterStateFile)
	raw, err := os.ReadFile(statePath)
	if err != nil || len(raw) == 0 || len(raw) > 4096 {
		return false
	}
	var state windowsUpdaterVerifiedState
	if json.Unmarshal(raw, &state) != nil {
		return false
	}
	if strings.TrimSpace(state.CheckedByWorkbenchVersion) != Version || len(strings.TrimSpace(state.SHA256)) != 64 {
		return false
	}
	hashValue, _, err := FileSHA256(target, maxWindowsAppBytes)
	return err == nil && strings.EqualFold(hashValue, strings.TrimSpace(state.SHA256))
}

func writeWindowsUpdaterState(installDir, releaseVersion, hashValue string) error {
	state := windowsUpdaterVerifiedState{
		CheckedByWorkbenchVersion: Version,
		ReleaseVersion:            strings.TrimSpace(releaseVersion),
		SHA256:                    strings.ToLower(strings.TrimSpace(hashValue)),
	}
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(installDir, ".workbench-updater-state-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	cleanup := true
	defer func() {
		_ = tmp.Close()
		if cleanup {
			_ = os.Remove(name)
		}
	}()
	if _, err := tmp.Write(append(raw, '\n')); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	statePath := filepath.Join(installDir, windowsUpdaterStateFile)
	_ = os.Remove(statePath)
	if err := os.Rename(name, statePath); err != nil {
		return err
	}
	cleanup = false
	return syncDirectory(installDir)
}

func isTransientWindowsUpdaterReplacementError(err error) bool {
	return errors.Is(err, syscall.Errno(5)) || errors.Is(err, syscall.Errno(32))
}
