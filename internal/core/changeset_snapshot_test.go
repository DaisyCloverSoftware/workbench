package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFingerprintChangesetChangesWithFileContent(t *testing.T) {
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
	if first == "" || second == "" || first == second {
		t.Fatalf("fingerprint did not track content change: %q %q", first, second)
	}
}

func TestFingerprintChangesetRepresentsDeletedPath(t *testing.T) {
	root := t.TempDir()
	in := ChangesetInspection{Project: root, BaseRevision: "base", Files: []string{"deleted.txt"}, Safe: true}
	got, err := fingerprintChangeset(in)
	if err != nil {
		t.Fatal(err)
	}
	if got == "" {
		t.Fatal("expected deleted path to contribute to a fingerprint")
	}
}

func TestSameChangesetShapeIncludesUntrackedSet(t *testing.T) {
	a := ChangesetInspection{Project: "repo", BaseRevision: "base", Files: []string{"a"}, Untracked: []string{"a"}, Diff: "diff", Safe: true}
	b := a
	b.Untracked = []string{"b"}
	if sameChangesetShape(a, b) {
		t.Fatal("different untracked sets must not compare equal")
	}
}
