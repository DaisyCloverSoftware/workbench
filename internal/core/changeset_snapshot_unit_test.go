package core

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestChangesetFingerprintTracksContent(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "file.txt")
	if err := os.WriteFile(path, []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	in := ChangesetInspection{Project: root, BaseRevision: "base", Files: []string{"file.txt"}, Safe: true}
	first, err := fingerprintChangeset(in)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("two\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := fingerprintChangeset(in)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("fingerprint did not change")
	}
}

func TestChangesetFingerprintIgnoresNonGitPermissionBits(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not expose POSIX group/other permission bits")
	}
	root := t.TempDir()
	path := filepath.Join(root, "file.txt")
	if err := os.WriteFile(path, []byte("same\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	in := ChangesetInspection{Project: root, BaseRevision: "base", Files: []string{"file.txt"}, Safe: true}
	first, err := fingerprintChangeset(in)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o664); err != nil {
		t.Fatal(err)
	}
	second, err := fingerprintChangeset(in)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("non-Git permission bits changed fingerprint: %s != %s", first, second)
	}
}

func TestChangesetFingerprintTracksExecutableBit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not expose POSIX executable bits through os.FileMode")
	}
	root := t.TempDir()
	path := filepath.Join(root, "file.txt")
	if err := os.WriteFile(path, []byte("same\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	in := ChangesetInspection{Project: root, BaseRevision: "base", Files: []string{"file.txt"}, Safe: true}
	first, err := fingerprintChangeset(in)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatal(err)
	}
	second, err := fingerprintChangeset(in)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("Git executable-bit change did not change fingerprint")
	}
}
