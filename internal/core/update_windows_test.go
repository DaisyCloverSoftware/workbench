package core

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyWindowsAMD64PE(t *testing.T) {
	file := filepath.Join(t.TempDir(), "Workbench.exe")
	if err := os.WriteFile(file, minimalPE64AMD64(), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := VerifyWindowsAMD64PE(file); err != nil {
		t.Fatalf("valid synthetic PE rejected: %v", err)
	}

	bad := minimalPE64AMD64()
	binary.LittleEndian.PutUint16(bad[0x84:0x86], 0x014c) // I386
	if err := os.WriteFile(file, bad, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := VerifyWindowsAMD64PE(file); err == nil || !strings.Contains(err.Error(), "not AMD64") {
		t.Fatalf("expected non-AMD64 PE rejection, got %v", err)
	}
}

func TestWindowsAppInstallRollbackRestoresOldExecutable(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.exe")
	target := filepath.Join(dir, "Workbench.exe")
	payload := minimalPE64AMD64()
	if err := os.WriteFile(source, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("old Workbench"), 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(payload)
	asset := VerifiedUpdateAsset{Name: WindowsReleaseAsset, Path: source, SHA256: hex.EncodeToString(sum[:]), Size: int64(len(payload))}

	tx, err := BeginWindowsAppInstall(asset, target)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyWindowsAMD64PE(target); err != nil {
		t.Fatalf("new app was not installed: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "old Workbench" {
		t.Fatalf("rollback did not restore old app: %q", got)
	}
}

func TestWindowsAppInstallCommitKeepsVerifiedExecutable(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.exe")
	target := filepath.Join(dir, "Workbench.exe")
	payload := minimalPE64AMD64()
	if err := os.WriteFile(source, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(payload)
	asset := VerifiedUpdateAsset{Name: WindowsReleaseAsset, Path: source, SHA256: hex.EncodeToString(sum[:]), Size: int64(len(payload))}

	tx, err := BeginWindowsAppInstall(asset, target)
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
		t.Fatal("committed Workbench.exe does not match verified asset")
	}
}

func TestWindowsAppInstallRefusesArbitraryTargetName(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.exe")
	payload := minimalPE64AMD64()
	if err := os.WriteFile(source, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(payload)
	asset := VerifiedUpdateAsset{Name: WindowsReleaseAsset, Path: source, SHA256: hex.EncodeToString(sum[:]), Size: int64(len(payload))}
	if _, err := BeginWindowsAppInstall(asset, filepath.Join(dir, "anything.exe")); err == nil {
		t.Fatal("expected arbitrary target filename to be rejected")
	}
}

func TestWindowsAppMatchesVerifiedAssetHandlesMissingAndMismatch(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "Workbench.exe")
	payload := minimalPE64AMD64()
	sum := sha256.Sum256(payload)
	asset := VerifiedUpdateAsset{Name: WindowsReleaseAsset, SHA256: hex.EncodeToString(sum[:]), Size: int64(len(payload))}
	matches, err := WindowsAppMatchesVerifiedAsset(target, asset)
	if err != nil || matches {
		t.Fatalf("missing app: matches=%t err=%v", matches, err)
	}
	if err := os.WriteFile(target, []byte("other"), 0o600); err != nil {
		t.Fatal(err)
	}
	matches, err = WindowsAppMatchesVerifiedAsset(target, asset)
	if err != nil || matches {
		t.Fatalf("mismatched app: matches=%t err=%v", matches, err)
	}
}

func minimalPE64AMD64() []byte {
	const peOffset = 0x80
	const optionalSize = 240
	b := make([]byte, peOffset+4+20+optionalSize)
	b[0], b[1] = 'M', 'Z'
	binary.LittleEndian.PutUint32(b[0x3c:0x40], peOffset)
	copy(b[peOffset:peOffset+4], []byte{'P', 'E', 0, 0})
	coff := peOffset + 4
	binary.LittleEndian.PutUint16(b[coff:coff+2], 0x8664) // AMD64
	binary.LittleEndian.PutUint16(b[coff+2:coff+4], 0)    // sections
	binary.LittleEndian.PutUint16(b[coff+16:coff+18], optionalSize)
	binary.LittleEndian.PutUint16(b[coff+18:coff+20], 0x0002) // executable image
	opt := coff + 20
	binary.LittleEndian.PutUint16(b[opt:opt+2], 0x20b) // PE32+
	binary.LittleEndian.PutUint64(b[opt+24:opt+32], 0x400000)
	binary.LittleEndian.PutUint32(b[opt+32:opt+36], 0x1000)
	binary.LittleEndian.PutUint32(b[opt+36:opt+40], 0x200)
	binary.LittleEndian.PutUint16(b[opt+40:opt+42], 6)
	binary.LittleEndian.PutUint32(b[opt+56:opt+60], 0x1000)
	binary.LittleEndian.PutUint32(b[opt+60:opt+64], 0x200)
	binary.LittleEndian.PutUint16(b[opt+68:opt+70], 2) // Windows GUI subsystem
	binary.LittleEndian.PutUint64(b[opt+72:opt+80], 0x100000)
	binary.LittleEndian.PutUint64(b[opt+80:opt+88], 0x1000)
	binary.LittleEndian.PutUint64(b[opt+88:opt+96], 0x100000)
	binary.LittleEndian.PutUint64(b[opt+96:opt+104], 0x1000)
	binary.LittleEndian.PutUint32(b[opt+108:opt+112], 16)
	return b
}
