package core

import (
	"os"
	"path/filepath"
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
