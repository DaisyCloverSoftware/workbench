package core

import (
	"context"
	"path/filepath"
	"strings"
)

// runnerProjectVisible keeps the runner inventory focused on canonical user
// repositories. Linked git worktrees are implementation/review artifacts of a
// canonical checkout, not separate Workbench projects, and legacy bootstrap
// backups are internal maintenance state rather than user workspaces.
func runnerProjectVisible(ctx context.Context, projectRoot, name string) bool {
	name = strings.TrimSpace(name)
	if name == "" || strings.HasPrefix(strings.ToLower(name), "workbench-pre-bootstrap-") {
		return false
	}

	commonDir, err := runGitLimited(ctx, projectRoot, 4096, "rev-parse", "--git-common-dir")
	if err != nil {
		return false
	}
	commonDir = strings.TrimSpace(commonDir)
	if commonDir == "" {
		return false
	}
	if !filepath.IsAbs(commonDir) {
		commonDir = filepath.Join(projectRoot, commonDir)
	}
	commonDir, err = filepath.Abs(commonDir)
	if err != nil {
		return false
	}
	if resolved, resolveErr := filepath.EvalSymlinks(commonDir); resolveErr == nil {
		commonDir = resolved
	}
	root, err := filepath.Abs(projectRoot)
	if err != nil {
		return false
	}
	if resolved, resolveErr := filepath.EvalSymlinks(root); resolveErr == nil {
		root = resolved
	}
	return withinRoot(root, commonDir)
}
