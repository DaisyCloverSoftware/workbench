# Workbench local checkout audit — 2026-08-21

Status: **INHERITED UNCOMMITTED EXPERIMENT — NOT ACCEPTED IMPLEMENTATION**

This record exists because the governance reset inspected the registered `runner://workbench` checkout rather than assuming it was clean. Private host paths/identifiers are deliberately omitted.

## Verified checkout state

The bounded private relay confirmed:

- current branch: `main`;
- working tree: **dirty**;
- tracked modification: `internal/core/relay_state.go`;
- untracked files:
  - `internal/core/relay_lock.go`;
  - `internal/core/relay_lock_test.go`;
  - `internal/core/relay_state_concurrency_test.go`.

The safe-command policy rejected `git worktree list --porcelain`, so additional worktree enumeration remains unverified. The reset did not widen the command allowlist to obtain it.

## What the uncommitted work is

The files form one coherent cross-process relay-state concurrency experiment.

Current `main` uses an in-process `sync.RWMutex` around reads/writes of the local `github-relay.json` state file and preserves atomic replacement with a uniquely named temporary file created in the same directory.

The inherited local experiment instead:

- moves the in-process mutex into `relay_lock.go`;
- adds a lock-file lease intended to serialize writers across separate Workbench processes that share the same relay-state file;
- waits up to five seconds for the lock;
- considers a lock file older than two minutes stale and removes it;
- changes the write temporary file from a unique `os.CreateTemp(...)` file to a fixed `<state>.tmp` path;
- adds tests for lock serialization, stale-lock recovery and concurrent record creation/update preservation.

The presence of these files is evidence that cross-process relay-state writer safety had been investigated locally. It is **not** evidence that the proposed implementation was approved, completed, tested or intended to supersede current `main`.

## Validation status

Three bounded attempts were made through the existing private safe-command channel:

- narrowed relay Core tests;
- `go test ./internal/core`;
- standard `go test ./...`.

The live MCP safe-command policy rejected all three command forms before execution. Therefore the inherited experiment is **UNTESTED by this audit**, not failed.

No authority was widened merely to run the experiment.

## Governance classification

### Current requirement

The durable relay-state implementation must remain safe when used by the actual supported process topology. Any change to its persistence/locking semantics must preserve atomic durable writes and must be validated for the relevant multi-process failure modes.

### Experiment status

**HISTORICAL / UNACCEPTED LOCAL IMPLEMENTATION EXPERIMENT.**

Do not merge or revive these local files by assumption.

### Open technical question

Whether current production topology can produce concurrent cross-process relay-state writers, and therefore whether a process-shared lock is required, is an **UNVERIFIED IMPLEMENTATION RISK** to investigate after the governance reset. The local experiment is evidence of the question, not the answer.

If concurrency hardening is later required, a fresh design must explicitly cover:

- cross-process writer serialization;
- readers observing only complete old/new state;
- crash recovery;
- stale-lock ownership/recovery without breaking a legitimate writer;
- atomic same-filesystem replacement;
- preservation of the current unique-temporary-file safety or an equally strong alternative;
- concurrent create/update record preservation;
- Windows and Linux semantics where applicable;
- tests that exercise separate processes, not only goroutines, when the threat model is cross-process.

A fixed `.tmp` name and age-only stale-lock deletion must not become current design merely because they existed in this abandoned working tree.

## Cleanup decision

The useful knowledge/rationale is now preserved here. The four working-tree changes are not current product authority and should not remain indefinitely in the registered active checkout.

Governance-reset cleanup may therefore restore `internal/core/relay_state.go` exactly to current canonical `main` and remove the three untracked experiment files, provided the exact post-cleanup `git status --short` is verified clean.

This is repository hygiene, not feature development.

## Remaining blind spot

The bounded safe-command surface does not currently permit worktree enumeration, so a clean status of the registered checkout will not by itself prove that no other local worktree exists elsewhere. That limitation must remain explicit rather than being bypassed with a generic shell.