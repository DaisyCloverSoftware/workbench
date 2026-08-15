//go:build !windows

package core

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

func spawnDetachedRunnerJob(jobID string) (int, error) {
	exe, err := os.Executable()
	if err != nil {
		return 0, err
	}
	null, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return 0, err
	}
	defer null.Close()
	cmd := exec.Command(exe, "job-execute", jobID)
	cmd.Stdin = null
	cmd.Stdout = null
	cmd.Stderr = null
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	pid := cmd.Process.Pid
	if err := cmd.Process.Release(); err != nil {
		return 0, err
	}
	return pid, nil
}

func runnerJobProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

func terminateRunnerJobProcess(pid int) error {
	if pid <= 0 {
		return nil
	}
	err := syscall.Kill(pid, syscall.SIGTERM)
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}
