//go:build !windows

package core

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestTerminateRunnerJobProcessStopsWholeDetachedGroup(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "orphan-marker")
	cmd := exec.Command("sh", "-c", `(sleep 1; printf orphan > "$1") & wait`, "sh", marker)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	t.Cleanup(func() { _ = syscall.Kill(-pid, syscall.SIGKILL) })

	time.Sleep(75 * time.Millisecond)
	if err := terminateRunnerJobProcess(pid); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	time.Sleep(1100 * time.Millisecond)
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("provider child survived durable job cancellation and wrote its marker")
	} else if !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
}
