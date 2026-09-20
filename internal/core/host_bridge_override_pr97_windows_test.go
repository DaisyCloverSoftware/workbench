//go:build windows

package core

import (
	"os"
	"path/filepath"
	"strings"
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


func TestOverridePR97FixedToolDiscovery(t *testing.T) {
	root := t.TempDir()
	programFiles := filepath.Join(root, "Program Files")
	systemRoot := filepath.Join(root, "Windows")
	emptyPath := filepath.Join(root, "empty-path")
	if err := os.MkdirAll(emptyPath, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ProgramW6432", programFiles)
	t.Setenv("ProgramFiles", programFiles)
	t.Setenv("SystemRoot", systemRoot)
	t.Setenv("PATH", emptyPath)

	git := filepath.Join(programFiles, "Git", "cmd", "git.exe")
	pwsh := filepath.Join(programFiles, "PowerShell", "7", "pwsh.exe")
	for _, file := range []string{git, pwsh} {
		if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte("test"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	if got := findOverrideRinGitExecutable(); !strings.EqualFold(filepath.Clean(got), filepath.Clean(git)) {
		t.Fatalf("git=%q want %q", got, git)
	}
	if got := findOverridePR97PowerShellExecutable(); !strings.EqualFold(filepath.Clean(got), filepath.Clean(pwsh)) {
		t.Fatalf("powershell=%q want %q", got, pwsh)
	}
}

func TestOverridePR97PowerShellFallsBackToWindowsPowerShell(t *testing.T) {
	root := t.TempDir()
	programFiles := filepath.Join(root, "Program Files")
	systemRoot := filepath.Join(root, "Windows")
	emptyPath := filepath.Join(root, "empty-path")
	if err := os.MkdirAll(emptyPath, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ProgramW6432", programFiles)
	t.Setenv("ProgramFiles", programFiles)
	t.Setenv("SystemRoot", systemRoot)
	t.Setenv("PATH", emptyPath)

	fallback := filepath.Join(systemRoot, "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	if err := os.MkdirAll(filepath.Dir(fallback), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fallback, []byte("test"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := findOverridePR97PowerShellExecutable(); !strings.EqualFold(filepath.Clean(got), filepath.Clean(fallback)) {
		t.Fatalf("powershell=%q want fallback %q", got, fallback)
	}
}
