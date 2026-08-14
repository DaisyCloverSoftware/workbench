package core

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// ScratchBaseDir returns the private, disk-backed directory Workbench owns for
// transient Git worktrees and other potentially large scratch data. Linux
// defaults to the user's cache rather than /tmp so tmpfs quotas cannot break
// review preparation or relay publication. Operators may point the parent at a
// larger volume with WORKBENCH_SCRATCH_ROOT.
func ScratchBaseDir() (string, error) {
	root := strings.TrimSpace(os.Getenv("WORKBENCH_SCRATCH_ROOT"))
	if root == "" {
		cache, err := os.UserCacheDir()
		if err != nil {
			return "", err
		}
		root = cache
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	base := filepath.Join(abs, "Workbench", "scratch")
	if err := os.MkdirAll(base, 0o700); err != nil {
		return "", err
	}
	info, err := os.Stat(base)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("Workbench scratch path is not a directory")
	}
	// Workbench owns this leaf even when the operator supplied its parent.
	// Keep source-bearing transient work private from other local users.
	if err := os.Chmod(base, 0o700); err != nil {
		return "", err
	}
	return filepath.Clean(base), nil
}

// NewScratchDirectory creates one private transient directory below
// ScratchBaseDir. Callers remain responsible for removing it when finished.
func NewScratchDirectory(pattern string) (string, error) {
	base, err := ScratchBaseDir()
	if err != nil {
		return "", err
	}
	return os.MkdirTemp(base, pattern)
}
