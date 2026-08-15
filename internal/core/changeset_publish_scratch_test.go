package core

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsolatedPublishGitDirUsesWorkbenchScratch(t *testing.T) {
	repo := initPrepareTestRepo(t)
	root := t.TempDir()
	t.Setenv("WORKBENCH_SCRATCH_ROOT", root)
	dir, err := isolatedPublishGitDir(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	base := filepath.Join(root, "Workbench", "scratch")
	rel, err := filepath.Rel(base, dir)
	if err != nil {
		t.Fatal(err)
	}
	if rel == "." || rel == ".." || filepath.IsAbs(rel) || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		t.Fatalf("publication scratch escaped Workbench cache: base=%q dir=%q rel=%q", base, dir, rel)
	}
	if !strings.HasPrefix(filepath.Base(dir), "publish-git-") {
		t.Fatalf("unexpected publication scratch name: %s", dir)
	}
}
