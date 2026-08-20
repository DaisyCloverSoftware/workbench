package core

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestWindowsUpdaterInstallCommitKeepsVerifiedExecutable(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source-updater.exe")
	target := filepath.Join(dir, WindowsUpdaterReleaseAsset)
	payload := minimalPE64AMD64()
	if err := os.WriteFile(source, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("old updater"), 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(payload)
	asset := VerifiedUpdateAsset{Name: WindowsUpdaterReleaseAsset, Path: source, SHA256: hex.EncodeToString(sum[:]), Size: int64(len(payload))}

	tx, err := BeginWindowsUpdaterInstall(asset, target)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	matches, err := WindowsAppMatchesVerifiedAsset(target, asset)
	if err != nil {
		t.Fatal(err)
	}
	if !matches {
		t.Fatal("committed Workbench-Updater.exe does not match verified asset")
	}
}

func TestWindowsUpdaterInstallRefusesWorkbenchAssetOrArbitraryName(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.exe")
	payload := minimalPE64AMD64()
	if err := os.WriteFile(source, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(payload)
	wrongAsset := VerifiedUpdateAsset{Name: WindowsReleaseAsset, Path: source, SHA256: hex.EncodeToString(sum[:]), Size: int64(len(payload))}
	if _, err := BeginWindowsUpdaterInstall(wrongAsset, filepath.Join(dir, WindowsUpdaterReleaseAsset)); err == nil {
		t.Fatal("expected Workbench.exe asset to be rejected for updater replacement")
	}
	updaterAsset := wrongAsset
	updaterAsset.Name = WindowsUpdaterReleaseAsset
	if _, err := BeginWindowsUpdaterInstall(updaterAsset, filepath.Join(dir, "anything.exe")); err == nil {
		t.Fatal("expected arbitrary updater target filename to be rejected")
	}
}
