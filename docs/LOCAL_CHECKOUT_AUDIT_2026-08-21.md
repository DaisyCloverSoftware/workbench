# Workbench local checkout audit — 2026-08-21

Status: **AUDITED, PRESERVED, CLEANED AND REALIGNED**

This record exists because the governance reset inspected the registered Workbench operational source checkout instead of assuming it matched canonical GitHub source. Private host paths/identifiers are deliberately omitted.

## Final verified state

At the post-PR #229 cleanup checkpoint the registered checkout was verified as:

- branch: `main`;
- HEAD: `ce4ecd5f0e47d764d6ad4221619390db1ea70af4`;
- working tree: clean;
- local Workbench worktree count: 1;
- old local-only SEC-008 commit preserved behind local audit ref `audit/pre-governance-reset-20260821`;
- audit ref target: `2defa97101447c04e8350bfae88414cbacafe237`.

The local audit ref is operational/private historical preservation only. It does not make that old line canonical source.

## Finding 1 — inherited uncommitted relay-state lock experiment

The initial bounded audit found:

- tracked modification: `internal/core/relay_state.go`;
- untracked `internal/core/relay_lock.go`;
- untracked `internal/core/relay_lock_test.go`;
- untracked `internal/core/relay_state_concurrency_test.go`.

The files formed one coherent cross-process relay-state concurrency experiment:

- lock-file lease across processes;
- five-second acquisition timeout;
- two-minute age-based stale-lock removal;
- fixed `<state>.tmp` path instead of current unique temp-file write;
- tests for lock serialization/stale recovery/concurrent saves.

The live safe-command policy rejected attempted Go-test command forms before execution, so the experiment was **untested by this audit**, not failed.

### Governance classification

**HISTORICAL / UNACCEPTED LOCAL IMPLEMENTATION EXPERIMENT.**

The rationale was preserved before cleanup. The implementation was not merged or promoted into current requirements.

Open technical question retained in `docs/DECISIONS.md`: whether supported production topology actually needs cross-process relay-state locking. If revisited, design must freshly cover serialization, reader completeness, crash/stale-lock safety, atomic replacement and multi-process tests.

## Finding 2 — stale local source lineage

After the dirty experiment was removed, the checkout was clean but still on an old v0.5-era source line rather than current GitHub `main`.

Exactly one local-only commit existed above its old remote-tracking main:

- `2defa97101447c04e8350bfae88414cbacafe237`
- subject: `sec: pin GitHub Actions to full commit SHAs (SEC-008)`
- changed only the build, release and runner workflow files.

Current public Workbench source already used full immutable action SHAs, so the security intent was preserved upstream.

### Cleanup action

A bounded tested realignment operation:

1. required a clean Workbench `main` checkout;
2. verified the exact known local-only commit subject/file set;
3. created/preserved local audit branch `audit/pre-governance-reset-20260821` at that exact commit;
4. fetched canonical public `main`;
5. verified the fetched head matched the exact reviewed governance operation commit;
6. realigned operational `main` to canonical source instead of merging the obsolete local line.

Post-action status/head/audit-ref were independently verified.

## Finding 3 — seven stale secondary worktrees

Privacy-safe worktree inventory then found eight Workbench worktrees total:

- primary `main` checkout;
- six detached worktrees at the old local-only `2defa97` line;
- one named worktree for `fix/preserve-changeset-file-modes`.

The six detached worktrees were clean. The named worktree was dirty in exactly:

- `internal/core/changeset_prepare.go`;
- `internal/core/changeset_prepare_test.go`.

A bounded diff/hash inspector proved the working blobs were not unique work. Their full Git object IDs were:

- `40900322cf4e61c4a65ce9b769958a3445812994`;
- `e90b7d17df6cdc7efa85f8c03739f756fd6bd260`.

Those exact blobs were already published by public commit `0b601caab1859e42767c5019ba61c01cf3af8c55` (file-mode preservation fix).

### Cleanup action

Only after that proof, a bounded tested operation:

- restored only those exact already-published duplicate edits;
- refused any other dirty detached/named worktree state;
- removed all seven secondary worktrees normally;
- deleted the already-merged old local named branch;
- pruned worktree metadata;
- fast-forwarded operational main to the exact reviewed governance checkpoint;
- preserved the local audit branch.

Final worktree inventory proved exactly one clean `main` worktree remained.

## Authority rule established by this audit

The canonical GitHub `main` repository is project/development authority. The registered cluster/local source checkout is an operational copy only.

Local commits, stale tracking refs, uncommitted files and worktrees MUST NOT silently redefine current Workbench. If local-only material is discovered again, preserve/classify it before deletion or realignment, then reconcile the operational copy to canonical source where appropriate.

## No lost work claim

No local material was removed by assumption:

- relay-lock experiment content/rationale was inspected and documented before deletion;
- local-only SEC-008 commit remains reachable behind an audit ref;
- stale file-mode dirty blobs were cryptographically matched to already-published public blobs before restoration/removal;
- clean detached worktrees were removed only after exact topology/head verification.

The registered checkout/worktree cleanup gate is therefore complete.
