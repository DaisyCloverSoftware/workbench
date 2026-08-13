package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInspectChangedPathRejectsCredentialPath(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("ordinary text\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := inspectChangedPath(root, ".env"); err == nil {
		t.Fatal("expected credential path to be rejected")
	}
}

func TestInspectChangedPathRejectsCredentialLikeContent(t *testing.T) {
	root := t.TempDir()
	marker := "token" + "=" + strings.Repeat("x", 20)
	if err := os.WriteFile(filepath.Join(root, "file.txt"), []byte(marker+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := inspectChangedPath(root, "file.txt"); err == nil {
		t.Fatal("expected credential-like content to be rejected")
	}
}

func TestInspectChangedPathRejectsBinaryContent(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "file.dat"), []byte{'a', 0, 'b'}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := inspectChangedPath(root, "file.dat"); err == nil {
		t.Fatal("expected binary content to be rejected")
	}
}
