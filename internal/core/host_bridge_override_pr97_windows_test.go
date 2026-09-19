//go:build windows

package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOverridePR97PathGuardsAllowAliasedParent(t *testing.T) {
	root := t.TempDir()
	realParent := filepath.Join(root, "real")
	cache := filepath.Join(realParent, "cache")
	bin := filepath.Join(realParent, "bin")
	if err := os.MkdirAll(cache, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	tool := filepath.Join(bin, "tool.exe")
	if err := os.WriteFile(tool, []byte("test"), 0o600); err != nil {
		t.Fatal(err)
	}

	aliasParent := filepath.Join(root, "alias-parent")
	if err := os.Symlink(realParent, aliasParent); err != nil {
		t.Skipf("Windows symlink privilege unavailable: %v", err)
	}

	if err := overridePR97OwnedDirectory(filepath.Join(aliasParent, "cache")); err != nil {
		t.Fatalf("safe directory beneath aliased parent was rejected: %v", err)
	}
	if err := overridePR97Regular(filepath.Join(aliasParent, "bin", "tool.exe")); err != nil {
		t.Fatalf("safe regular file beneath aliased parent was rejected: %v", err)
	}

	if err := overridePR97OwnedDirectory(aliasParent); err == nil {
		t.Fatal("leaf directory alias was accepted")
	}
	leafFileAlias := filepath.Join(root, "tool-link.exe")
	if err := os.Symlink(tool, leafFileAlias); err != nil {
		t.Skipf("Windows file symlink privilege unavailable: %v", err)
	}
	if err := overridePR97Regular(leafFileAlias); err == nil {
		t.Fatal("leaf file alias was accepted")
	}
}
