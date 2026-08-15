//go:build !windows

package core

import "testing"

func TestUpdateLockIsExclusiveAndReusable(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WORKBENCH_SCRATCH_ROOT", root)

	first, err := AcquireUpdateLock()
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	if _, err := AcquireUpdateLock(); err == nil {
		t.Fatal("expected second concurrent update lock to fail")
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	second, err := AcquireUpdateLock()
	if err != nil {
		t.Fatalf("update lock was not reusable after release: %v", err)
	}
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
}
