//go:build linux

package core

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestTransferTaskWorkspaceChangesMatchesFingerprintWithCollaborativeUmask(t *testing.T) {
	isolateKnowledgeConfig(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	previousUmask := syscall.Umask(0o002)
	defer syscall.Umask(previousUmask)

	repo := initPrepareTestRepo(t)
	ctx := context.Background()
	ws, err := CreateTaskWorkspace(ctx, repo, "task-transfer-umask")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws.Workspace, "tracked.txt"), []byte("after\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws.Workspace, "added.txt"), []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	workspaceSnapshot, err := SnapshotChangeset(ctx, ws.Workspace)
	if err != nil {
		t.Fatal(err)
	}

	result, err := TransferTaskWorkspaceChanges(ctx, ws)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed || result.Prepared.Fingerprint != workspaceSnapshot.Fingerprint {
		t.Fatalf("unexpected transfer result: %#v", result)
	}
	transferred, err := SnapshotChangeset(ctx, repo)
	if err != nil {
		t.Fatal(err)
	}
	if transferred.Fingerprint != workspaceSnapshot.Fingerprint {
		t.Fatalf("umask changed transferred fingerprint: %s != %s", transferred.Fingerprint, workspaceSnapshot.Fingerprint)
	}
}
