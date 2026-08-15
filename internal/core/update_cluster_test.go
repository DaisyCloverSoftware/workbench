package core

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestVerifyLinuxAMD64ELF(t *testing.T) {
	file := filepath.Join(t.TempDir(), "binary")
	if err := os.WriteFile(file, minimalELF64AMD64(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := VerifyLinuxAMD64ELF(file); err != nil {
		t.Fatalf("valid synthetic ELF rejected: %v", err)
	}
	if err := os.WriteFile(file, []byte("MZ-not-elf"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := VerifyLinuxAMD64ELF(file); err == nil {
		t.Fatal("expected non-ELF file to be rejected")
	}
}

func TestExtractVerifiedClusterArchiveRequiresExactELFBinaries(t *testing.T) {
	archive := writeClusterArchive(t, nil)
	bundle, err := ExtractVerifiedClusterArchive(archive, "0.7.0", strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	defer bundle.Cleanup()
	if bundle.Version != "0.7.0" || len(bundle.Binaries) != 3 {
		t.Fatalf("unexpected bundle: %#v", bundle)
	}
	for _, name := range []string{"workbench-runner", "workbench-server", "workbench-relay"} {
		path := bundle.Binaries[name]
		if path == "" {
			t.Fatalf("bundle missing %s", name)
		}
		if err := VerifyLinuxAMD64ELF(path); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if filepath.Dir(path) != bundle.ScratchDir {
			t.Fatalf("%s escaped bundle scratch: %s", name, path)
		}
	}
}

func TestExtractVerifiedClusterArchiveRejectsUnexpectedEntry(t *testing.T) {
	archive := writeClusterArchive(t, func(zw *zip.Writer) {
		w, err := zw.Create("unexpected.txt")
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte("no"))
	})
	if _, err := ExtractVerifiedClusterArchive(archive, "0.7.0", ""); err == nil || !strings.Contains(err.Error(), "contains 4 entries") {
		t.Fatalf("expected unexpected-entry rejection, got %v", err)
	}
}

func TestExtractVerifiedClusterArchiveRejectsSymlinkEntry(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "cluster.zip")
	f, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for archiveName := range clusterArchiveFiles {
		h := &zip.FileHeader{Name: archiveName, Method: zip.Store}
		if archiveName == "workbench-runner-linux-amd64" {
			h.SetMode(os.ModeSymlink | 0o777)
		} else {
			h.SetMode(0o755)
		}
		w, err := zw.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write(minimalELF64AMD64())
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := ExtractVerifiedClusterArchive(archive, "0.7.0", ""); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("expected symlink rejection, got %v", err)
	}
}

func TestBinaryInstallTransactionRollbackRestoresAllTargets(t *testing.T) {
	dir := t.TempDir()
	var replacements []BinaryReplacement
	for _, name := range []string{"workbench-relay", "workbench-runner", "workbench-server"} {
		source := filepath.Join(dir, name+".source")
		target := filepath.Join(dir, name)
		payload := append(minimalELF64AMD64(), []byte("new-"+name)...)
		if err := os.WriteFile(source, payload, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte("old-"+name), 0o755); err != nil {
			t.Fatal(err)
		}
		replacements = append(replacements, BinaryReplacement{Name: name, SourcePath: source, TargetPath: target})
	}

	tx, err := BeginBinaryInstall(replacements)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range replacements {
		if err := VerifyLinuxAMD64ELF(r.TargetPath); err != nil {
			t.Fatalf("new target %s was not installed: %v", r.Name, err)
		}
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	for _, r := range replacements {
		b, err := os.ReadFile(r.TargetPath)
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != "old-"+r.Name {
			t.Fatalf("rollback did not restore %s: %q", r.Name, b)
		}
	}
}

func TestBinaryInstallTransactionCommitKeepsNewTargets(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	target := filepath.Join(dir, "workbench-runner")
	payload := append(minimalELF64AMD64(), []byte("new")...)
	if err := os.WriteFile(source, payload, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	tx, err := BeginBinaryInstall([]BinaryReplacement{{Name: "runner", SourcePath: source, TargetPath: target}})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := VerifyLinuxAMD64ELF(target); err != nil {
		t.Fatal(err)
	}
	matches, err := filepath.Glob(filepath.Join(dir, ".workbench-runner.workbench-old-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("committed transaction left backups: %v", matches)
	}
}

func TestBeginBinaryInstallRefusesNonRegularTarget(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	if err := os.WriteFile(source, minimalELF64AMD64(), 0o700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "workbench-runner")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := BeginBinaryInstall([]BinaryReplacement{{Name: "runner", SourcePath: source, TargetPath: target}}); err == nil || !strings.Contains(err.Error(), "non-regular") {
		t.Fatalf("expected non-regular target rejection, got %v", err)
	}
}

func writeClusterArchive(t *testing.T, extra func(*zip.Writer)) string {
	t.Helper()
	archive := filepath.Join(t.TempDir(), "cluster.zip")
	f, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for archiveName := range clusterArchiveFiles {
		h := &zip.FileHeader{Name: archiveName, Method: zip.Store}
		h.SetMode(0o755)
		w, err := zw.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(minimalELF64AMD64()); err != nil {
			t.Fatal(err)
		}
	}
	if extra != nil {
		extra(zw)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return archive
}

func minimalELF64AMD64() []byte {
	b := make([]byte, 64)
	copy(b[:4], []byte{0x7f, 'E', 'L', 'F'})
	b[4] = 2 // ELFCLASS64
	b[5] = 1 // little endian
	b[6] = 1 // ELF version
	binary.LittleEndian.PutUint16(b[16:18], 2)  // ET_EXEC
	binary.LittleEndian.PutUint16(b[18:20], 62) // EM_X86_64
	binary.LittleEndian.PutUint32(b[20:24], 1)  // version
	binary.LittleEndian.PutUint16(b[52:54], 64) // ELF header size
	return b
}

var _ = bytes.Equal
var _ = runtime.GOOS
