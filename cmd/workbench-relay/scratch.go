package main

import (
	"fmt"
	"os"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

// The relay can create full Git worktrees while publishing results. Keep every
// process-level temporary file/worktree under Workbench's private cache instead
// of the system temp directory, which may be a small tmpfs on Linux hosts.
func init() {
	base, err := core.ScratchBaseDir()
	if err != nil {
		fatal(fmt.Errorf("prepare Workbench scratch storage: %w", err))
	}
	for _, name := range []string{"TMPDIR", "TMP", "TEMP"} {
		if err := os.Setenv(name, base); err != nil {
			fatal(fmt.Errorf("configure relay scratch storage: %w", err))
		}
	}
}
