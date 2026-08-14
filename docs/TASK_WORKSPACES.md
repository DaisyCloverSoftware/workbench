# Isolated task workspaces

Workbench must not make an autonomous coding worker the owner of the user's active Git branch, index or publication authority.

The task-workspace contract separates those responsibilities:

1. Workbench resolves the requested project to the canonical Git repository root.
2. Isolation is eligible only when the user's source worktree is clean. A dirty tree is never silently omitted from the worker's view and is never copied into an autonomous workspace without an explicit higher-level policy.
3. Workbench records the exact source `HEAD` and creates a detached task worktree under private local cache state.
4. The worker may inspect/edit/test inside that task worktree, but it must not commit, push or deploy.
5. Durable project memory remains associated with the logical source repository, not the transient worktree path.
6. On successful worker completion, Workbench first proves the user's source tree is still clean and at the same baseline.
7. The isolated changes are frozen through the normal changeset snapshot/preparation pipeline, producing a Workbench-owned review commit.
8. Workbench applies the verified text changes back to the unchanged source worktree and then compares fingerprints. A mismatch fails closed.
9. The prepared Workbench-owned commit can separately enter the publication pipeline. Coding workers never receive the publication target or push credentials.

## Failure rules

Workbench refuses automatic transfer when:

- the source worktree changed while the task was running;
- source `HEAD` moved;
- the worker created a commit;
- workspace metadata no longer matches the task/project;
- changeset inspection or preparation detects a protected path, symlink, binary content, oversized content, or probable secret material;
- the applied source result does not reproduce the exact isolated changeset fingerprint.

A refusal preserves the user's source changes and keeps the isolated workspace available for deterministic diagnosis/recovery. It is not a reason to grant the worker broader Git authority.

## Restart behaviour

Task-workspace metadata is private local state keyed by canonical project + durable task ID. Reopening the same task recovers the same detached workspace when its metadata and Git worktree remain valid. Workbench does not create a second workspace for the same durable task.

This primitive is intentionally independent of any model provider or agent harness. The engine/runner can adopt it without changing the worker protocol, and publication remains a separate control-plane capability.
