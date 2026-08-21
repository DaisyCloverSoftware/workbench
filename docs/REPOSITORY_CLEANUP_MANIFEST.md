# Workbench repository cleanup manifest — 2026-08-21

Status: **REVIEWED, NOT FULLY EXECUTED**

This manifest separates cleanup decisions from destructive actions. A branch/file is not deleted merely because it is old.

## Baseline

The audit enumerated **288 remote branches**. They include long-lived feature/fix branches, release-request branches, proof/test branches, diagnostics and the open PR #110 branch.

Full local worktree/checkout status was not available through the current bounded private control interface. Therefore repository cleanup cannot be declared globally complete.

## Explicit branch dispositions

### SAFE TO DELETE AFTER GOVERNANCE RECORD IS MERGED

`fix/operations-active-session-state-20260821`

- Source of PR #221.
- Comparison against current main: ahead 0, behind 5 at audit time.
- No unique commits remain outside main.
- Its behaviour is now explicitly classified as rejected in `docs/DECISIONS.md` (the merged code remains an implementation discrepancy until later product correction).

`release-request/v0.9.54`

- Comparison against current main: ahead 0, behind 2 at audit time.
- Its validated release commit is already in main history through PR #222.
- No unique branch commit remains.

`diag/dashboard-activity-live-20260821`

- One unique 12-line diagnostic script, seven commits behind main at audit time.
- The script only invokes the runner `chat_activity` tool and prints bounded live activity.
- Its durable useful knowledge — the fact that the 0.9.53/0.9.54 runner feed was non-empty and activity/session data existed — is preserved in `docs/CURRENT_STATE.md` and the governance audit record.
- The script is not required for the current canonical product contract.
- Retain until the governance branch is merged; then the branch may be deleted while Git history remains available.

### RETAIN / DO NOT DELETE DURING THIS RESET

`main`

- Default branch and canonical source.

`governance/reset-20260821`

- Active governance-reset branch until merged/closed.

Open PR #110 branch (`feat/project-knowledge-graph` or the exact current PR head ref)

- The PR is intentionally not merged, but it remains an open review/audit object.
- Its implementation basis is stale (0.9.10), yet deleting the branch while the PR is open would destroy the active review object.
- Resolve/close the PR explicitly before branch deletion.

### REQUIRES BATCH AUDIT BEFORE DELETE

All other release-request, release-proof, test-proof, old feature/fix and no-op branches.

The branch-count scale means pattern-based deletion without checking unique commits would violate the governance reset rule. For each candidate, establish at least:

1. whether it has commits not reachable from `main`;
2. whether an open PR/issue refers to it;
3. whether unique operational/requirements evidence has been canonicalised;
4. whether it is needed for rollback/migration/audit;
5. whether its removal would lose anything not already retained in Git history/release artifacts.

Only then delete the remote ref.

## Active-tree material reviewed

### Current root/docs

No duplicate canonical governance files existed before this reset. New documents have distinct roles:

- `docs/GOVERNANCE.md` — permanent governance rules;
- `docs/DECISIONS.md` — decision/supersession/do-not-reintroduce register;
- `docs/CURRENT_STATE.md` — current verified implementation/runtime state;
- `docs/GOVERNANCE_RESET_2026-08-21.md` — audit coverage/completion gate;
- `docs/CONVERSATION_PRUNING_MANIFEST.md` — conversation disposition;
- `docs/REPOSITORY_CLEANUP_MANIFEST.md` — cleanup disposition;
- `docs/POST_RESET_HANDOFF.md` — fresh development bootstrap.

Existing architecture/security/roadmap/Operations/UI-acceptance documents are updated rather than duplicated.

### Temporary diagnostics

The diagnostic branch script is not copied into the active tree because its evidence has been preserved and it is not a canonical operational requirement.

### Superseded tests

Current 0.9.54 source contains a regression test that expects `completed + Active=true` remote session activity to be projected as Running. This test now conflicts with the canonical Operations contract. It is **not modified during the feature freeze**; it is recorded as a P0 post-reset implementation discrepancy.

### Release no-op commit

The current `main` HEAD is an identical-tree publication retrigger. Do not rewrite public history to remove it. Preserve it as audit evidence and fix the release process later rather than hiding the symptom.

## Local/private cleanup blind spots

Not proven by this reset:

- every developer checkout `git status`;
- every local worktree;
- untracked/build artifacts outside GitHub;
- private relay repository retention/compaction safety;
- every remote branch's unique-commit reachability.

The bounded audit interface was not widened into a generic shell merely to obtain these facts.

## Completion rule

Repository cleanup is **not complete** until:

- all remote branches have an explicit disposition or are intentionally retained;
- local checkouts/worktrees can be inspected safely;
- branch deletions approved above are executed after governance merge;
- open PR #110 is resolved explicitly;
- private relay retention policy is decided and safely applied if cleanup is desired;
- final clean-status evidence is recorded.

Until then, this manifest is the authoritative cleanup queue and development remains subject to the governance reset gate.
