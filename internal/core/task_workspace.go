package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type TaskWorkspace struct {
	Version      int       `json:"version"`
	TaskID       string    `json:"task_id"`
	Project      string    `json:"project"`
	Workspace    string    `json:"workspace"`
	BaseRevision string    `json:"base_revision"`
	CreatedAt    time.Time `json:"created_at"`
}

const taskWorkspaceMetadataVersion = 1

// CreateTaskWorkspace creates a detached Git worktree for one autonomous task.
// It deliberately requires the user's source worktree to be clean: silently
// omitting unrelated uncommitted edits would give a worker a misleading view,
// while copying them into an autonomous workspace would blur ownership of the
// eventual changeset. Dirty repositories can continue through legacy in-place
// routing until the engine explicitly adopts isolation policy.
func CreateTaskWorkspace(ctx context.Context, project, taskID string) (TaskWorkspace, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return TaskWorkspace{}, errors.New("task id is required")
	}
	root, err := canonicalTaskWorkspaceProject(ctx, project)
	if err != nil {
		return TaskWorkspace{}, err
	}
	inspection, err := InspectChangeset(ctx, root)
	if err != nil {
		return TaskWorkspace{}, err
	}
	if !inspection.Clean {
		return TaskWorkspace{}, errors.New("source worktree has local changes; isolated task workspace requires a clean repository")
	}

	base := strings.TrimSpace(inspection.BaseRevision)
	path, metadataPath, err := taskWorkspacePaths(root, taskID)
	if err != nil {
		return TaskWorkspace{}, err
	}
	if existing, ok, err := loadTaskWorkspace(metadataPath); err != nil {
		return TaskWorkspace{}, err
	} else if ok {
		if existing.Project != root || existing.TaskID != taskID || existing.BaseRevision != base || existing.Workspace != path {
			return TaskWorkspace{}, errors.New("existing task workspace metadata does not match this task")
		}
		if validTaskWorkspace(ctx, existing) {
			return existing, nil
		}
		return TaskWorkspace{}, errors.New("recorded task workspace is no longer valid")
	}
	if _, statErr := os.Stat(path); statErr == nil {
		return TaskWorkspace{}, errors.New("task workspace path already exists without trusted metadata")
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return TaskWorkspace{}, statErr
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return TaskWorkspace{}, err
	}

	cmd := exec.CommandContext(ctx, "git", "-C", root, "worktree", "add", "--detach", "--quiet", path, base)
	configureChildProcess(cmd, false)
	if out, runErr := cmd.CombinedOutput(); runErr != nil {
		return TaskWorkspace{}, fmt.Errorf("create isolated task workspace: %s", strings.TrimSpace(string(out)))
	}
	created := TaskWorkspace{
		Version:      taskWorkspaceMetadataVersion,
		TaskID:       taskID,
		Project:      root,
		Workspace:    path,
		BaseRevision: base,
		CreatedAt:    time.Now().UTC(),
	}
	if err := saveTaskWorkspace(metadataPath, created); err != nil {
		_ = exec.Command("git", "-C", root, "worktree", "remove", "--force", path).Run()
		_ = os.RemoveAll(path)
		return TaskWorkspace{}, err
	}
	return created, nil
}

func OpenTaskWorkspace(project, taskID string) (TaskWorkspace, bool, error) {
	root, err := canonicalTaskWorkspaceProject(context.Background(), project)
	if err != nil {
		return TaskWorkspace{}, false, err
	}
	_, metadataPath, err := taskWorkspacePaths(root, strings.TrimSpace(taskID))
	if err != nil {
		return TaskWorkspace{}, false, err
	}
	ws, ok, err := loadTaskWorkspace(metadataPath)
	if err != nil || !ok {
		return ws, ok, err
	}
	if ws.Project != root || ws.TaskID != strings.TrimSpace(taskID) || !validTaskWorkspace(context.Background(), ws) {
		return TaskWorkspace{}, false, errors.New("recorded task workspace is invalid")
	}
	return ws, true, nil
}

// RemoveTaskWorkspace removes only a Workbench-owned worktree whose metadata
// still matches the requested task. It never removes the user's source tree.
func RemoveTaskWorkspace(ctx context.Context, project, taskID string) error {
	root, err := canonicalTaskWorkspaceProject(ctx, project)
	if err != nil {
		return err
	}
	path, metadataPath, err := taskWorkspacePaths(root, strings.TrimSpace(taskID))
	if err != nil {
		return err
	}
	ws, ok, err := loadTaskWorkspace(metadataPath)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	if ws.Project != root || ws.TaskID != strings.TrimSpace(taskID) || ws.Workspace != path {
		return errors.New("task workspace metadata mismatch; refusing cleanup")
	}
	cmd := exec.CommandContext(ctx, "git", "-C", root, "worktree", "remove", "--force", path)
	configureChildProcess(cmd, false)
	if out, runErr := cmd.CombinedOutput(); runErr != nil {
		return fmt.Errorf("remove task workspace: %s", strings.TrimSpace(string(out)))
	}
	if err := os.Remove(metadataPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func canonicalTaskWorkspaceProject(ctx context.Context, project string) (string, error) {
	root, err := projectRoot(project)
	if err != nil {
		return "", err
	}
	gitRoot, err := runGitLimited(ctx, root, 16<<10, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", errors.New("task workspace requires a Git repository")
	}
	resolved, err := filepath.EvalSymlinks(strings.TrimSpace(gitRoot))
	if err != nil {
		return "", err
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return "", err
	}
	if filepath.Clean(resolved) != filepath.Clean(root) {
		return "", errors.New("task workspace project must be the Git repository root")
	}
	return root, nil
}

func taskWorkspacePaths(project, taskID string) (string, string, error) {
	if taskID == "" {
		return "", "", errors.New("task id is required")
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256([]byte(project + "\x00" + taskID))
	key := hex.EncodeToString(sum[:12])
	base := filepath.Join(cache, "Workbench", "task-workspaces")
	return filepath.Join(base, key), filepath.Join(base, key+".json"), nil
}

func validTaskWorkspace(ctx context.Context, ws TaskWorkspace) bool {
	if ws.Version != taskWorkspaceMetadataVersion || ws.Workspace == "" || ws.Project == "" || ws.BaseRevision == "" {
		return false
	}
	info, err := os.Stat(ws.Workspace)
	if err != nil || !info.IsDir() {
		return false
	}
	gitRoot, err := runGitLimited(ctx, ws.Workspace, 16<<10, "rev-parse", "--show-toplevel")
	if err != nil {
		return false
	}
	resolved, err := filepath.EvalSymlinks(strings.TrimSpace(gitRoot))
	if err != nil {
		return false
	}
	resolved, err = filepath.Abs(resolved)
	return err == nil && filepath.Clean(resolved) == filepath.Clean(ws.Workspace)
}

func loadTaskWorkspace(path string) (TaskWorkspace, bool, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return TaskWorkspace{}, false, nil
	}
	if err != nil {
		return TaskWorkspace{}, false, err
	}
	if len(b) > 64<<10 {
		return TaskWorkspace{}, false, errors.New("task workspace metadata is unexpectedly large")
	}
	var ws TaskWorkspace
	if err := json.Unmarshal(b, &ws); err != nil {
		return TaskWorkspace{}, false, err
	}
	return ws, true, nil
}

func saveTaskWorkspace(path string, ws TaskWorkspace) error {
	b, err := json.MarshalIndent(ws, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".task-workspace-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		return err
	}
	if _, err := f.Write(b); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
