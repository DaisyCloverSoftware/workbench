# Deterministic changeset publishing

Workbench coding workers are allowed to edit and verify a repository, but they are not granted direct source-control publication authority. Publication is a separate deterministic capability owned by Workbench.

## Invariant

A coding worker may inspect, edit, build and test inside its authorised workspace. It must not push, publish, deploy or widen its own authority. Workbench may later turn a verified local result into a reviewable branch or pull request only after a deterministic policy gate succeeds.

This separation keeps model autonomy useful without making a model-generated shell command the authority boundary for source-control publication.

## Publication pipeline

1. **Capture baseline.** Record the repository HEAD that the task started from.
2. **Inspect changes.** Use Workbench's bounded changeset inspection to enumerate tracked and untracked files, reject protected paths, symlinks, binary/oversized content and probable secrets, and capture a bounded textual diff.
3. **Freeze a snapshot.** Fingerprint the baseline, file set, file modes and exact changed-file content. A moving working tree is retried rather than published.
4. **Prepare in isolation.** Reconstruct the inspected changes in a temporary git worktree based on the recorded baseline. The user's active branch, index and working tree are not switched or staged.
5. **Verify exactness.** Confirm the staged file set matches the inspected set exactly and run secret/binary checks again on the staged result.
6. **Verify quality.** Re-run the task's declared build/test checks against the prepared tree when policy requires it.
7. **Create a Workbench-owned local commit.** Use a generated `workbench/...` branch name and generic Workbench commit identity. The coding worker does not create this commit.
8. **Publish through policy.** A separate publisher may push only that Workbench-owned branch to the configured repository and optionally open a pull request. It never pushes the current branch and never performs a production deployment.
9. **Report provenance.** Record baseline revision, prepared commit, verification evidence and the Workbench task that produced the change.

## Fail-closed conditions

Automatic publication stops before remote mutation when any of these are true:

- the configured project is not the repository root;
- the baseline moved unexpectedly;
- the working changes mutate while being snapshotted;
- the prepared file set differs from the inspected file set;
- a protected credential path, probable secret, symlink, binary change or configured size ceiling is encountered;
- verification fails;
- the repository remote is missing or does not match the configured publication target;
- the requested action would push the user's current branch, deploy, publish a package/release, or otherwise exceed source-review authority.

A failed automatic publication gate does not retroactively fail a completed local coding task. Workbench should preserve the verified local result and surface only a genuine policy decision when no safe recovery path exists.

## Public/private boundary

The generic publisher belongs in the public Workbench source. Repository credentials, private remotes, task intent, local machine identity and private dogfood logs remain local/private configuration. Public pull-request text must be generated from public-safe change metadata rather than copied from private worker transcripts.

## Relationship to the Git relay

The Git relay is a control/result transport between Chat and a private Workbench installation. It is not the source-code publisher. The publisher operates on an authorised project repository; the relay remains a narrow message channel. Keeping those responsibilities separate prevents a private task transport from becoming a general remote shell or an accidental source-publication path.
