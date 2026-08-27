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
	// Keep plain exec.Command and interactive processes on their historical
	// lifecycle. The process-group policy is specifically for bounded commands
	// created with exec.CommandContext, which install a Cancel callback.
	if cmd == nil || interactive || cmd.Cancel == nil {
		return
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	// Context-backed non-interactive Workbench commands run in their own
	// process group so cancellation terminates descendants such as ssh, git
	// helpers, or shell children instead of killing only the immediate process.
	cmd.SysProcAttr.Setpgid = true

	// exec.CommandContext defaults to killing only cmd.Process. Preserve that
	// hard-stop behavior while addressing the whole process group.
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
