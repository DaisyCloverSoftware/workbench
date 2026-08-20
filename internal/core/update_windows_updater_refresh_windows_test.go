//go:build windows

package core

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestWindowsUpdaterStateRequiresCurrentWorkbenchVersionAndMatchingHash(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, WindowsUpdaterReleaseAsset)
	payload := minimalPE64AMD64()
	if err := os.WriteFile(target, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(payload)
	hashValue := hex.EncodeToString(sum[:])
	if err := writeWindowsUpdaterState(dir, Version, hashValue); err != nil {
		t.Fatal(err)
	}
	if !windowsUpdaterStateMatches(dir, target) {
		t.Fatal("fresh verified updater state did not match installed updater")
	}
	if err := os.WriteFile(target, append(payload, 0), 0o600); err != nil {
		t.Fatal(err)
	}
	if windowsUpdaterStateMatches(dir, target) {
		t.Fatal("tampered updater incorrectly matched recorded verified state")
	}
}
