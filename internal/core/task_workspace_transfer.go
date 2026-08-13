package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type TaskWorkspaceTransfer struct {
	Changed   bool              `json:"changed"`
	Prepared  PreparedChangeset `json:"prepared,omitempty"`
	ApplyInfo string            `json:"apply_info,omitempty"`
}

// TransferTaskWorkspaceChanges materializes one isolated worker workspace back
// into the user's source worktree without merging or switching branches. The
// source must still be clean and at the exact baseline from which the isolated
// workspace was created. Worker commits are refused: Workbench owns the review
// commit so coding workers never acquire source-control publication authority.
func TransferTaskWorkspaceChanges(ctx context.Context, ws TaskWorkspace) (TaskWorkspaceTransfer, error) {
	if !validTaskWorkspace(ctx, ws) {
		return TaskWorkspaceTransfer{}, errors.New("task workspace is no longer valid")
	}

	source, err := InspectChangeset(ctx, ws.Project)
	if err != nil {
		return TaskWorkspaceTransfer{}, err
	}
	if !source.Clean {
		return TaskWorkspaceTransfer{}, errors.New("source worktree changed while the isolated task was running; refusing to overwrite it")
	}
	if strings.TrimSpace(source.BaseRevision) != strings.TrimSpace(ws.BaseRevision) {
		return TaskWorkspaceTransfer{}, errors.New("source HEAD changed while the isolated task was running; refusing to transfer changes")
	}
	workspaceHead, err := runGitLimited(ctx, ws.Workspace, 16<<10, "rev-parse", "HEAD")
	if err != nil {
		return TaskWorkspaceTransfer{}, err
	}
	if strings.TrimSpace(workspaceHead) != strings.TrimSpace(ws.BaseRevision) {
		return TaskWorkspaceTransfer{}, errors.New("worker created a commit inside its isolated workspace; Workbench requires workers to leave review commits to the control plane")
	}

	inspection, err := InspectChangeset(ctx, ws.Workspace)
	if err != nil {
		return TaskWorkspaceTransfer{}, err
	}
	if inspection.Clean {
		return TaskWorkspaceTransfer{Changed: false}, nil
	}
	prepared, err := PrepareChangeset(ctx, ws.Workspace, ws.TaskID)
	if err != nil {
		return TaskWorkspaceTransfer{}, fmt.Errorf("prepare isolated task changes: %w", err)
	}
	patch, err := runGitLimited(ctx, ws.Workspace, maxChangesetDiffBytes,
		"diff", "--no-ext-diff", "--no-color", prepared.BaseRevision, prepared.Commit, "--")
	if err != nil {
		return TaskWorkspaceTransfer{}, fmt.Errorf("render isolated task patch: %w", err)
	}
	if strings.TrimSpace(patch) == "" {
		return TaskWorkspaceTransfer{}, errors.New("prepared task changes produced an empty patch")
	}
	applyInfo, err := ApplyPatch(ctx, ws.Project, patch)
	if err != nil {
		return TaskWorkspaceTransfer{}, fmt.Errorf("apply isolated task changes to source: %w", err)
	}

	transferred, err := SnapshotChangeset(ctx, ws.Project)
	if err != nil {
		return TaskWorkspaceTransfer{}, fmt.Errorf("verify transferred source changes: %w", err)
	}
	if transferred.Fingerprint != prepared.Fingerprint {
		return TaskWorkspaceTransfer{}, errors.New("transferred source changes do not match the isolated task fingerprint")
	}
	return TaskWorkspaceTransfer{Changed: true, Prepared: prepared, ApplyInfo: applyInfo}, nil
}
