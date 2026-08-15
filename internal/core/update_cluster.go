package core

import (
	"archive/zip"
	"context"
	"debug/elf"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	ClusterReleaseAsset         = "Workbench-Cluster-linux-amd64.zip"
	ClusterReleaseChecksumAsset = "Workbench-Cluster-linux-amd64.zip.sha256"
	maxClusterArchiveBytes      = 64 << 20
	maxClusterBinaryBytes       = 64 << 20
	maxClusterExtractedBytes    = 160 << 20
)

var clusterArchiveFiles = map[string]string{
	"workbench-runner-linux-amd64": "workbench-runner",
	"workbench-server-linux-amd64": "workbench-server",
	"workbench-relay-linux-amd64":  "workbench-relay",
}

type ClusterUpdateBundle struct {
	Version       string            `json:"version"`
	ArchiveSHA256 string            `json:"archive_sha256"`
	ScratchDir    string            `json:"scratch_dir"`
	Binaries      map[string]string `json:"binaries"`
}

type BinaryReplacement struct {
	Name       string
	SourcePath string
	TargetPath string
}

type binarySwap struct {
	replacement BinaryReplacement
	staged      string
	backup      string
	hadTarget   bool
	swapped     bool
}

// BinaryInstallTransaction keeps the old binaries available until the caller
// has verified the new runner and restarted any previously active Workbench
// services. Rollback restores the old inode set; Commit removes backups.
type BinaryInstallTransaction struct {
	swaps  []binarySwap
	closed bool
}

func PrepareOfficialClusterUpdate(ctx context.Context, currentVersion string) (UpdateCheck, ClusterUpdateBundle, error) {
	check, err := CheckOfficialUpdate(ctx, currentVersion)
	if err != nil {
		return UpdateCheck{}, ClusterUpdateBundle{}, err
	}
	if !check.UpdateAvailable {
		return check, ClusterUpdateBundle{}, nil
	}
	verified, err := DownloadVerifiedReleaseAsset(ctx, check.Release, ClusterReleaseAsset, ClusterReleaseChecksumAsset, maxClusterArchiveBytes)
	if err != nil {
		return check, ClusterUpdateBundle{}, err
	}
	downloadDir := filepath.Dir(verified.Path)
	defer os.RemoveAll(downloadDir)

	bundle, err := ExtractVerifiedClusterArchive(verified.Path, check.Release.Version, verified.SHA256)
	if err != nil {
		return check, ClusterUpdateBundle{}, err
	}
	return check, bundle, nil
}

func ExtractVerifiedClusterArchive(archivePath, version, archiveSHA256 string) (ClusterUpdateBundle, error) {
	if _, err := parseStableVersion(version); err != nil {
		return ClusterUpdateBundle{}, err
	}
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return ClusterUpdateBundle{}, fmt.Errorf("open Workbench cluster archive: %w", err)
	}
	defer zr.Close()
	if len(zr.File) != len(clusterArchiveFiles) {
		return ClusterUpdateBundle{}, fmt.Errorf("Workbench cluster archive contains %d entries; expected %d", len(zr.File), len(clusterArchiveFiles))
	}

	scratch, err := NewScratchDirectory("cluster-update-")
	if err != nil {
		return ClusterUpdateBundle{}, err
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.RemoveAll(scratch)
		}
	}()

	seen := map[string]bool{}
	binaries := map[string]string{}
	var total uint64
	for _, f := range zr.File {
		name := f.Name
		targetName, expected := clusterArchiveFiles[name]
		if !expected {
			return ClusterUpdateBundle{}, fmt.Errorf("unexpected file in Workbench cluster archive: %q", name)
		}
		if seen[name] {
			return ClusterUpdateBundle{}, fmt.Errorf("duplicate file in Workbench cluster archive: %q", name)
		}
		seen[name] = true
		if filepath.Base(name) != name || strings.ContainsAny(name, `/\\`) || name == "." || name == ".." {
			return ClusterUpdateBundle{}, fmt.Errorf("unsafe path in Workbench cluster archive: %q", name)
		}
		mode := f.Mode()
		if mode&os.ModeSymlink != 0 || mode.IsDir() || (!mode.IsRegular() && mode != 0) {
			return ClusterUpdateBundle{}, fmt.Errorf("Workbench cluster archive entry is not a regular file: %q", name)
		}
		if f.UncompressedSize64 == 0 || f.UncompressedSize64 > maxClusterBinaryBytes {
			return ClusterUpdateBundle{}, fmt.Errorf("Workbench cluster archive entry has invalid size: %q", name)
		}
		total += f.UncompressedSize64
		if total > maxClusterExtractedBytes {
			return ClusterUpdateBundle{}, errors.New("Workbench cluster archive expands beyond the allowed size")
		}

		dest := filepath.Join(scratch, targetName)
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o700)
		if err != nil {
			return ClusterUpdateBundle{}, err
		}
		rc, err := f.Open()
		if err != nil {
			_ = out.Close()
			return ClusterUpdateBundle{}, err
		}
		n, copyErr := io.Copy(out, io.LimitReader(rc, int64(f.UncompressedSize64)+1))
		closeReadErr := rc.Close()
		syncErr := out.Sync()
		closeWriteErr := out.Close()
		if copyErr != nil {
			return ClusterUpdateBundle{}, copyErr
		}
		if closeReadErr != nil {
			return ClusterUpdateBundle{}, closeReadErr
		}
		if syncErr != nil {
			return ClusterUpdateBundle{}, syncErr
		}
		if closeWriteErr != nil {
			return ClusterUpdateBundle{}, closeWriteErr
		}
		if uint64(n) != f.UncompressedSize64 {
			return ClusterUpdateBundle{}, fmt.Errorf("Workbench cluster archive size mismatch for %q", name)
		}
		if err := VerifyLinuxAMD64ELF(dest); err != nil {
			return ClusterUpdateBundle{}, fmt.Errorf("verify %s: %w", name, err)
		}
		binaries[targetName] = dest
	}
	for archiveName, targetName := range clusterArchiveFiles {
		if !seen[archiveName] || binaries[targetName] == "" {
			return ClusterUpdateBundle{}, fmt.Errorf("Workbench cluster archive is missing %q", archiveName)
		}
	}

	cleanup = false
	return ClusterUpdateBundle{
		Version:       version,
		ArchiveSHA256: archiveSHA256,
		ScratchDir:    scratch,
		Binaries:      binaries,
	}, nil
}

func (b ClusterUpdateBundle) Cleanup() error {
	if strings.TrimSpace(b.ScratchDir) == "" {
		return nil
	}
	return os.RemoveAll(b.ScratchDir)
}

func VerifyLinuxAMD64ELF(filePath string) error {
	info, err := os.Lstat(filePath)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return errors.New("binary is not a regular file")
	}
	f, err := elf.Open(filePath)
	if err != nil {
		return fmt.Errorf("not an ELF executable: %w", err)
	}
	defer f.Close()
	if f.Class != elf.ELFCLASS64 || f.Data != elf.ELFDATA2LSB || f.Machine != elf.EM_X86_64 {
		return fmt.Errorf("binary is not little-endian ELF64 x86-64 (class=%v data=%v machine=%v)", f.Class, f.Data, f.Machine)
	}
	if f.Type != elf.ET_EXEC && f.Type != elf.ET_DYN {
		return fmt.Errorf("ELF file type %v is not executable", f.Type)
	}
	return nil
}

func ClusterBinaryReplacements(bundle ClusterUpdateBundle, binDir string) ([]BinaryReplacement, error) {
	if _, err := parseStableVersion(bundle.Version); err != nil {
		return nil, err
	}
	absBin, err := filepath.Abs(binDir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(absBin, 0o755); err != nil {
		return nil, err
	}
	info, err := os.Stat(absBin)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("Workbench binary directory is not a directory: %s", absBin)
	}

	names := make([]string, 0, len(clusterArchiveFiles))
	for _, targetName := range clusterArchiveFiles {
		names = append(names, targetName)
	}
	sort.Strings(names)
	replacements := make([]BinaryReplacement, 0, len(names))
	for _, name := range names {
		source := bundle.Binaries[name]
		if source == "" {
			return nil, fmt.Errorf("cluster update bundle is missing %s", name)
		}
		if err := VerifyLinuxAMD64ELF(source); err != nil {
			return nil, fmt.Errorf("verify staged %s: %w", name, err)
		}
		replacements = append(replacements, BinaryReplacement{Name: name, SourcePath: source, TargetPath: filepath.Join(absBin, name)})
	}
	return replacements, nil
}

func BeginBinaryInstall(replacements []BinaryReplacement) (*BinaryInstallTransaction, error) {
	if len(replacements) == 0 {
		return nil, errors.New("no Workbench binaries were supplied for installation")
	}
	tx := &BinaryInstallTransaction{swaps: make([]binarySwap, len(replacements))}
	for i, replacement := range replacements {
		if strings.TrimSpace(replacement.Name) == "" || replacement.SourcePath == "" || replacement.TargetPath == "" {
			tx.discardStaged()
			return nil, errors.New("invalid Workbench binary replacement")
		}
		if err := VerifyLinuxAMD64ELF(replacement.SourcePath); err != nil {
			tx.discardStaged()
			return nil, fmt.Errorf("verify replacement %s: %w", replacement.Name, err)
		}
		targetDir := filepath.Dir(replacement.TargetPath)
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			tx.discardStaged()
			return nil, err
		}
		if info, err := os.Lstat(replacement.TargetPath); err == nil {
			if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
				tx.discardStaged()
				return nil, fmt.Errorf("refusing to replace non-regular Workbench binary %s", replacement.TargetPath)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			tx.discardStaged()
			return nil, err
		}

		stage, err := copyExecutableToDirectory(replacement.SourcePath, targetDir, "."+filepath.Base(replacement.TargetPath)+".workbench-new-*")
		if err != nil {
			tx.discardStaged()
			return nil, err
		}
		tx.swaps[i] = binarySwap{replacement: replacement, staged: stage}
	}

	for i := range tx.swaps {
		swap := &tx.swaps[i]
		if _, err := os.Lstat(swap.replacement.TargetPath); err == nil {
			backup, err := reserveSiblingPath(filepath.Dir(swap.replacement.TargetPath), "."+filepath.Base(swap.replacement.TargetPath)+".workbench-old-*")
			if err != nil {
				_ = tx.rollbackProcessed(i - 1)
				tx.discardStaged()
				return nil, err
			}
			if err := os.Rename(swap.replacement.TargetPath, backup); err != nil {
				_ = tx.rollbackProcessed(i - 1)
				tx.discardStaged()
				return nil, fmt.Errorf("backup %s: %w", swap.replacement.Name, err)
			}
			swap.backup = backup
			swap.hadTarget = true
		} else if !errors.Is(err, os.ErrNotExist) {
			_ = tx.rollbackProcessed(i - 1)
			tx.discardStaged()
			return nil, err
		}
		if err := os.Rename(swap.staged, swap.replacement.TargetPath); err != nil {
			if swap.hadTarget {
				_ = os.Rename(swap.backup, swap.replacement.TargetPath)
				swap.backup = ""
			}
			_ = tx.rollbackProcessed(i - 1)
			tx.discardStaged()
			return nil, fmt.Errorf("install %s: %w", swap.replacement.Name, err)
		}
		swap.staged = ""
		swap.swapped = true
		if err := syncDirectory(filepath.Dir(swap.replacement.TargetPath)); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}
	return tx, nil
}

func (tx *BinaryInstallTransaction) Rollback() error {
	if tx == nil || tx.closed {
		return nil
	}
	err := tx.rollbackProcessed(len(tx.swaps) - 1)
	tx.discardStaged()
	tx.closed = true
	return err
}

func (tx *BinaryInstallTransaction) Commit() error {
	if tx == nil || tx.closed {
		return nil
	}
	var errs []string
	for i := range tx.swaps {
		swap := &tx.swaps[i]
		if swap.backup != "" {
			if err := os.Remove(swap.backup); err != nil && !errors.Is(err, os.ErrNotExist) {
				errs = append(errs, err.Error())
			}
			swap.backup = ""
		}
	}
	tx.discardStaged()
	tx.closed = true
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func (tx *BinaryInstallTransaction) rollbackProcessed(last int) error {
	var errs []string
	for i := last; i >= 0; i-- {
		swap := &tx.swaps[i]
		if !swap.swapped {
			continue
		}
		if err := os.Remove(swap.replacement.TargetPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, fmt.Sprintf("remove new %s: %v", swap.replacement.Name, err))
			continue
		}
		if swap.hadTarget && swap.backup != "" {
			if err := os.Rename(swap.backup, swap.replacement.TargetPath); err != nil {
				errs = append(errs, fmt.Sprintf("restore %s: %v", swap.replacement.Name, err))
				continue
			}
			swap.backup = ""
		}
		swap.swapped = false
		if err := syncDirectory(filepath.Dir(swap.replacement.TargetPath)); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func (tx *BinaryInstallTransaction) discardStaged() {
	for i := range tx.swaps {
		if tx.swaps[i].staged != "" {
			_ = os.Remove(tx.swaps[i].staged)
			tx.swaps[i].staged = ""
		}
	}
}

func copyExecutableToDirectory(source, dir, pattern string) (string, error) {
	in, err := os.Open(source)
	if err != nil {
		return "", err
	}
	defer in.Close()
	out, err := os.CreateTemp(dir, pattern)
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
	if err := out.Chmod(0o755); err != nil {
		return "", err
	}
	if _, err := io.Copy(out, io.LimitReader(in, maxClusterBinaryBytes+1)); err != nil {
		return "", err
	}
	if err := out.Sync(); err != nil {
		return "", err
	}
	if err := out.Close(); err != nil {
		return "", err
	}
	if err := VerifyLinuxAMD64ELF(name); err != nil {
		return "", err
	}
	cleanup = false
	return name, nil
}

func reserveSiblingPath(dir, pattern string) (string, error) {
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", err
	}
	name := f.Name()
	if err := f.Close(); err != nil {
		_ = os.Remove(name)
		return "", err
	}
	if err := os.Remove(name); err != nil {
		return "", err
	}
	return name, nil
}

func syncDirectory(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := f.Sync(); err != nil {
		// Windows does not support fsync on directory handles. Atomic rename is
		// still used there for unit tests, while cluster application is Linux-only.
		if os.PathSeparator == '\\' {
			return nil
		}
		return err
	}
	return nil
}
