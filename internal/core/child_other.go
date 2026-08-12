//go:build !windows

package core

import "os/exec"

func configureChildProcess(cmd *exec.Cmd, interactive bool) {}
