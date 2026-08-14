package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScratchBaseDirUsesConfiguredParent(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WORKBENCH_SCRATCH_ROOT", root)

	base, err := ScratchBaseDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "Workbench", "scratch")
	assertSameDirectory(t, base, want)

	info, err := os.Stat(base)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Fatalf("scratch base is not a directory: %s", base)
	}
}

func TestNewScratchDirectoryStaysBelowWorkbenchCache(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WORKBENCH_SCRATCH_ROOT", root)

	dir, err := NewScratchDirectory("prepare-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	base := filepath.Join(root, "Workbench", "scratch")
	rel, err := filepath.Rel(base, dir)
	if err != nil {
		t.Fatal(err)
	}
	if rel == "." || rel == ".." || filepath.IsAbs(rel) || len(rel) >= 3 && rel[:3] == ".."+string(os.PathSeparator) {
		t.Fatalf("scratch directory escaped Workbench cache: base=%q dir=%q rel=%q", base, dir, rel)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Fatalf("scratch path is not a directory: %s", dir)
	}
}

func assertSameDirectory(t *testing.T, got, want string) {
	t.Helper()
	gotInfo, err := os.Stat(got)
	if err != nil {
		t.Fatalf("stat %q: %v", got, err)
	}
	wantInfo, err := os.Stat(want)
	if err != nil {
		t.Fatalf("stat %q: %v", want, err)
	}
	if !os.SameFile(gotInfo, wantInfo) {
		t.Fatalf("directories differ: got=%q want=%q", got, want)
	}
}
