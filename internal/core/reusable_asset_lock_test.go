package core

import (
	"os"
	"testing"
	"time"
)

func TestReusableAssetLockRecoversStaleLease(t *testing.T) {
	isolateKnowledgeConfig(t)
	path, err := ReusableAssetStatePath()
	if err != nil {
		t.Fatal(err)
	}
	lockPath := path + ".lock"
	if err := os.WriteFile(lockPath, []byte("stale\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-reusableAssetLockStale - time.Second)
	if err := os.Chtimes(lockPath, old, old); err != nil {
		t.Fatal(err)
	}
	release, err := lockReusableAssetWrite()
	if err != nil {
		t.Fatal(err)
	}
	release()
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("expected lock lease to be removed, got %v", err)
	}
}

func TestReusableAssetStateWriteLeavesNoTemporarySnapshot(t *testing.T) {
	isolateKnowledgeConfig(t)
	if _, _, err := SaveReusableAsset(KnowledgeItem{Scope: ScopeGlobal, Kind: KindRoutine, Title: "Checks", Content: "Run the focused test."}, ""); err != nil {
		t.Fatal(err)
	}
	path, err := ReusableAssetStatePath()
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepathDir(path))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if len(entry.Name()) >= len(".reusable-assets-") && entry.Name()[:len(".reusable-assets-")] == ".reusable-assets-" {
			t.Fatalf("temporary snapshot left behind: %s", entry.Name())
		}
	}
}

func filepathDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			if i == 0 {
				return path[:1]
			}
			return path[:i]
		}
	}
	return "."
}
