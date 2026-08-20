package core

import (
	"crypto/sha256"
	"debug/pe"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	WindowsReleaseAsset                  = "Workbench.exe"
	WindowsReleaseChecksumAsset          = "Workbench.exe.sha256"
	WindowsUpdaterReleaseAsset           = "Workbench-Updater.exe"
	WindowsUpdaterReleaseChecksumAsset   = "Workbench-Updater.exe.sha256"
	maxWindowsAppBytes                   = 128 << 20
)

type WindowsAppInstallTransaction struct {
	target    string
	backup    string
	hadTarget bool
	closed    bool
}

func VerifyWindowsAMD64PE(filePath string) error {
	info, err := os.Lstat(filePath)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return errors.New("Windows application is not a regular file")
	}
	if info.Size() <= 0 || info.Size() > maxWindowsAppBytes {
		return errors.New("Windows application size is outside the allowed range")
	}
	f, err := pe.Open(filePath)
	if err != nil {
		return fmt.Errorf("not a PE executable: %w", err)
	}
	defer f.Close()
	if f.FileHeader.Machine != pe.IMAGE_FILE_MACHINE_AMD64 {
		return fmt.Errorf("Windows application is not AMD64 (machine=%#x)", f.FileHeader.Machine)
	}
	if f.FileHeader.Characteristics&pe.IMAGE_FILE_EXECUTABLE_IMAGE == 0 {
		return errors.New("PE file is not marked executable")
	}
	if _, ok := f.OptionalHeader.(*pe.OptionalHeader64); !ok {
		return errors.New("Windows application does not contain a PE32+ optional header")
	}
	return nil
}

func FileSHA256(filePath string, maxBytes int64) (string, int64, error) {
	if maxBytes <= 0 {
		return "", 0, errors.New("hash size limit must be positive")
	}
	info, err := os.Lstat(filePath)
	if err != nil {
		return "", 0, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", 0, errors.New("file to hash is not regular")
	}
	if info.Size() < 0 || info.Size() > maxBytes {
		return "", 0, errors.New("file to hash exceeds the allowed size")
	}
	f, err := os.Open(filePath)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, maxBytes+1))
	if err != nil {
		return "", 0, err
	}
	if n > maxBytes || n != info.Size() {
		return "", n, errors.New("file changed while Workbench was hashing it")
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

func WindowsAppMatchesVerifiedAsset(target string, asset VerifiedUpdateAsset) (bool, error) {
	if _, err := os.Stat(target); errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	hashValue, size, err := FileSHA256(target, maxWindowsAppBytes)
	if err != nil {
		return false, err
	}
	if asset.Size > 0 && size != asset.Size {
		return false, nil
	}
	return strings.EqualFold(hashValue, asset.SHA256), nil
}

// BeginWindowsAppInstall atomically swaps a verified official Workbench.exe
// into place but retains the old file until Commit. If relaunching the new app
// fails, the updater can Rollback and restore the previous executable.
func BeginWindowsAppInstall(asset VerifiedUpdateAsset, target string) (*WindowsAppInstallTransaction, error) {
	return beginWindowsExecutableInstall(asset, target, WindowsReleaseAsset)
}

// BeginWindowsUpdaterInstall applies the same verified transactional replacement
// boundary to Workbench-Updater.exe. Workbench uses this only while the updater
// is not needed for the current operation, allowing installed copies to receive
// updater reliability fixes without trusting an unverified replacement binary.
func BeginWindowsUpdaterInstall(asset VerifiedUpdateAsset, target string) (*WindowsAppInstallTransaction, error) {
	return beginWindowsExecutableInstall(asset, target, WindowsUpdaterReleaseAsset)
}

func beginWindowsExecutableInstall(asset VerifiedUpdateAsset, target, expectedAsset string) (*WindowsAppInstallTransaction, error) {
	if asset.Name != expectedAsset {
		return nil, fmt.Errorf("unexpected Windows update asset %q", asset.Name)
	}
	if len(asset.SHA256) != sha256.Size*2 {
		return nil, errors.New("verified Windows update asset has no valid SHA-256")
	}
	if filepath.Base(target) != expectedAsset {
		return nil, fmt.Errorf("Workbench Windows updater may only install %s", expectedAsset)
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return nil, err
	}
	if err := VerifyWindowsAMD64PE(asset.Path); err != nil {
		return nil, fmt.Errorf("verify staged %s: %w", expectedAsset, err)
	}
	stagedHash, stagedSize, err := FileSHA256(asset.Path, maxWindowsAppBytes)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(stagedHash, asset.SHA256) || (asset.Size > 0 && stagedSize != asset.Size) {
		return nil, fmt.Errorf("staged %s no longer matches its verified release asset", expectedAsset)
	}

	targetDir := filepath.Dir(absTarget)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return nil, err
	}
	if info, err := os.Lstat(absTarget); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("refusing to replace non-regular %s: %s", expectedAsset, absTarget)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	stage, err := copyWindowsExecutable(asset.Path, targetDir)
	if err != nil {
		return nil, err
	}
	cleanupStage := true
	defer func() {
		if cleanupStage {
			_ = os.Remove(stage)
		}
	}()

	tx := &WindowsAppInstallTransaction{target: absTarget}
	if _, err := os.Lstat(absTarget); err == nil {
		backup, err := reserveSiblingPath(targetDir, "."+expectedAsset+".workbench-old-*")
		if err != nil {
			return nil, err
		}
		if err := os.Rename(absTarget, backup); err != nil {
			return nil, fmt.Errorf("replace %s: close the running executable and retry (%w)", expectedAsset, err)
		}
		tx.backup = backup
		tx.hadTarget = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	if err := os.Rename(stage, absTarget); err != nil {
		if tx.hadTarget {
			_ = os.Rename(tx.backup, absTarget)
			tx.backup = ""
		}
		return nil, fmt.Errorf("install %s: %w", expectedAsset, err)
	}
	cleanupStage = false
	if err := syncDirectory(targetDir); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := VerifyWindowsAMD64PE(absTarget); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	installedHash, installedSize, err := FileSHA256(absTarget, maxWindowsAppBytes)
	if err != nil || !strings.EqualFold(installedHash, asset.SHA256) || (asset.Size > 0 && installedSize != asset.Size) {
		_ = tx.Rollback()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("installed %s failed post-swap checksum verification", expectedAsset)
	}
	return tx, nil
}

func (tx *WindowsAppInstallTransaction) Rollback() error {
	if tx == nil || tx.closed {
		return nil
	}
	var errs []error
	if err := os.Remove(tx.target); err != nil && !errors.Is(err, os.ErrNotExist) {
		errs = append(errs, err)
	}
	if tx.hadTarget && tx.backup != "" {
		if err := os.Rename(tx.backup, tx.target); err != nil {
			errs = append(errs, err)
		} else {
			tx.backup = ""
		}
	}
	if err := syncDirectory(filepath.Dir(tx.target)); err != nil {
		errs = append(errs, err)
	}
	tx.closed = true
	return errors.Join(errs...)
}

func (tx *WindowsAppInstallTransaction) Commit() error {
	if tx == nil || tx.closed {
		return nil
	}
	var err error
	if tx.backup != "" {
		err = os.Remove(tx.backup)
		if errors.Is(err, os.ErrNotExist) {
			err = nil
		}
		tx.backup = ""
	}
	tx.closed = true
	return err
}

func copyWindowsExecutable(source, targetDir string) (string, error) {
	in, err := os.Open(source)
	if err != nil {
		return "", err
	}
	defer in.Close()
	out, err := os.CreateTemp(targetDir, ".Workbench.exe.workbench-new-*")
	if err != nil {
		return "", err
	}
	name := out.Name()
	cleanup := true
	defer func() {
		_ = out.Close()
		if cleanup {
			_ = os.Remove(name)
		}
	}()
	n, err := io.Copy(out, io.LimitReader(in, maxWindowsAppBytes+1))
	if err != nil {
		return "", err
	}
	if n <= 0 || n > maxWindowsAppBytes {
		return "", errors.New("staged Workbench.exe exceeds the allowed size")
	}
	if err := out.Sync(); err != nil {
		return "", err
	}
	if err := out.Close(); err != nil {
		return "", err
	}
	if err := VerifyWindowsAMD64PE(name); err != nil {
		return "", err
	}
	cleanup = false
	return name, nil
}
