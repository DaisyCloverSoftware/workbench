package core

import "os/exec"

// ConfigureBoundedChildProcess applies Workbench's platform-specific bounded
// child-process policy to a context-backed, non-interactive command. Internal
// executables such as workbench-relay use this wrapper so cancellation semantics
// stay consistent without duplicating Unix process-group handling.
func ConfigureBoundedChildProcess(cmd *exec.Cmd) {
	configureChildProcess(cmd, false)
}
