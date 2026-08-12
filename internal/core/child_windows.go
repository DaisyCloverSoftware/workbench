//go:build windows

package core

import (
	"os/exec"
	"syscall"
)

func configureChildProcess(cmd *exec.Cmd, interactive bool) {
	if interactive {
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000010} // CREATE_NEW_CONSOLE
	} else {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	}
}
