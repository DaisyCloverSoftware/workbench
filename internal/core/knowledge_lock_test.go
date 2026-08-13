package core

import (
	"os"
	"testing"
	"time"
)

func TestKnowledgeFileLockSerializesIndependentWriters(t *testing.T) {
	isolateKnowledgeConfig(t)
	releaseFirst, err := acquireKnowledgeFileLock()
	if err != nil {
		t.Fatal(err)
	}
	defer releaseFirst()

	acquired := make(chan func(), 1)
	errs := make(chan error, 1)
	go func() {
		release, err := acquireKnowledgeFileLock()
		if err != nil {
			errs <- err
			return
		}
		acquired <- release
	}()

	select {
	case release := <-acquired:
		release()
		t.Fatal("second writer acquired knowledge lock before first released it")
	case err := <-errs:
		t.Fatal(err)
	case <-time.After(80 * time.Millisecond):
	}

	releaseFirst()
	select {
	case release := <-acquired:
		release()
	case err := <-errs:
		t.Fatal(err)
	case <-time.After(2 * time.Second):
		t.Fatal("second writer did not acquire knowledge lock after release")
	}
}

func TestKnowledgeFileLockRecoversStaleLease(t *testing.T) {
	isolateKnowledgeConfig(t)
	path, err := KnowledgeStatePath()
	if err != nil {
		t.Fatal(err)
	}
	lockPath := path + ".lock"
	if err := os.WriteFile(lockPath, []byte("stale\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-knowledgeLockStale - time.Minute)
	if err := os.Chtimes(lockPath, old, old); err != nil {
		t.Fatal(err)
	}
	release, err := acquireKnowledgeFileLock()
	if err != nil {
		t.Fatal(err)
	}
	release()
}
