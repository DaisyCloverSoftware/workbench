package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CreateOperationsTaskWorkspace creates a disposable detached worktree at the
// repository's committed HEAD without requiring the user's source worktree to
// be clean. Machine-side operations must not be blocked by unrelated local
// edits, and those edits must never be copied into OpenClaw's workspace.
//
// Unlike coding workspaces, an existing valid operations workspace is reused
// even if the source checkout has moved since the first attempt. That keeps a
// supervised/retried operation pinned to the same committed snapshot while its
// actual host/runtime state may continue to evolve.
func CreateOperationsTaskWorkspace(ctx context.Context, project, taskID string) (TaskWorkspace, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return TaskWorkspace{}, errors.New("task id is required")
	}
	root, err := canonicalTaskWorkspaceProject(ctx, project)
	if err != nil {
		return TaskWorkspace{}, err
	}
	path, metadataPath, err := taskWorkspacePaths(root, taskID)
	if err != nil {
		return TaskWorkspace{}, err
	}
	if existing, ok, err := loadTaskWorkspace(metadataPath); err != nil {
		return TaskWorkspace{}, err
	} else if ok {
		if existing.TaskID != taskID || !sameDirectoryIdentity(existing.Project, root) || !sameDirectoryIdentity(existing.Workspace, path) {
			return TaskWorkspace{}, errors.New("existing operations workspace metadata does not match this task")
		}
		if validTaskWorkspace(ctx, existing) {
			return existing, nil
		}
		return TaskWorkspace{}, errors.New("recorded operations workspace is no longer valid")
	}

	base, err := runGitLimited(ctx, root, 16<<10, "rev-parse", "HEAD")
	if err != nil || strings.TrimSpace(base) == "" {
		return TaskWorkspace{}, errors.New("operations workspace could not resolve committed repository HEAD")
	}
	base = strings.TrimSpace(base)
	baseBranch := currentTaskBaseBranch(ctx, root)

	if _, statErr := os.Stat(path); statErr == nil {
		return TaskWorkspace{}, errors.New("operations workspace path already exists without trusted metadata")
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return TaskWorkspace{}, statErr
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return TaskWorkspace{}, err
	}

	cmd := exec.CommandContext(ctx, "git", "-C", root, "worktree", "add", "--detach", "--quiet", path, base)
	configureChildProcess(cmd, false)
	if out, runErr := cmd.CombinedOutput(); runErr != nil {
		return TaskWorkspace{}, fmt.Errorf("create isolated operations workspace: %s", strings.TrimSpace(string(out)))
	}
	created := TaskWorkspace{
		Version:      taskWorkspaceMetadataVersion,
		TaskID:       taskID,
		Project:      root,
		Workspace:    path,
		BaseRevision: base,
		BaseBranch:   baseBranch,
		CreatedAt:    time.Now().UTC(),
	}
	if err := saveTaskWorkspace(metadataPath, created); err != nil {
		_ = exec.Command("git", "-C", root, "worktree", "remove", "--force", path).Run()
		_ = os.RemoveAll(path)
		return TaskWorkspace{}, err
	}
	return created, nil
}
