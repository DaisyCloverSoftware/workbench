//go:build !windows

package core

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrepareChangesetPreservesExecutableModeChange(t *testing.T) {
	repo := initPrepareTestRepo(t)
	path := filepath.Join(repo, "tracked.txt")
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatal(err)
	}
	prepared, err := PrepareChangeset(context.Background(), repo, "task-mode")
	if err != nil {
		t.Fatal(err)
	}
	entry := prepareTestGit(t, repo, "ls-tree", prepared.Commit, "--", "tracked.txt")
	if !strings.HasPrefix(entry, "100755 blob ") {
		t.Fatalf("prepared executable mode was not preserved: %q", entry)
	}
}
