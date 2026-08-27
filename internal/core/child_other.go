//go:build !windows

package core

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

const nonInteractiveChildWaitDelay = 2 * time.Second

func configureChildProcess(cmd *exec.Cmd, interactive bool) {
	if cmd == nil || interactive {
		return
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	// Non-interactive Workbench commands run in their own process group so a
	// context cancellation can terminate descendants such as ssh, git helpers,
	// or shell children instead of killing only the immediate process.
	cmd.SysProcAttr.Setpgid = true

	// exec.CommandContext installs a direct-process Kill callback. Replace it
	// with a process-group kill when a context-backed command is configured.
	// Plain exec.Command calls have no Cancel callback and retain their existing
	// lifecycle semantics.
	if cmd.Cancel == nil {
		return
	}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	// If an escaped descendant starts a new process group but keeps inherited
	// stdout/stderr descriptors open, never allow cmd.Wait to block forever.
	cmd.WaitDelay = nonInteractiveChildWaitDelay
}
