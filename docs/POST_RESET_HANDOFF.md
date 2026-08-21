# Workbench development handoff — canonical reset baseline

**Activation status: NOT YET AUTHORISED.**

This handoff is intentionally based on the canonical repository record, not historical conversations. Ordinary feature development remains frozen until `docs/GOVERNANCE_RESET_2026-08-21.md` is changed to COMPLETE with evidence or an explicit residual-risk decision.

## Bootstrap rule

Always begin by reading current canonical `main`; do not treat the SHA in an old handoff as the permanent source of truth. Verify the current `main` HEAD at the start of a fresh development session.

The last pre-final-record cleanup checkpoint is:

- `main`: `ce4ecd5f0e47d764d6ad4221619390db1ea70af4` (PR #229);
- active public branch surface at that cleanup checkpoint: `main` only;
- open PRs at that cleanup checkpoint: zero;
- archival history tag: `archive/pre-governance-reset-20260821` (historical preservation only, never a development baseline).

The final governance-record PR is expected to advance `main` beyond that checkpoint; bootstrap from the then-current canonical head rather than this fixed checkpoint.

## What Workbench is

Workbench is a ChatGPT-first developer/operations control plane and durable execution bridge.

- ChatGPT is the primary reasoning/coding brain.
- Workbench provides durable project/task state, safe repository eyes/hands, scheduling/execution infrastructure, private transport, bounded machine controls and an outbound typed Windows bridge.
- OpenClaw/other autonomous harnesses are optional operator/worker capacity, not the default coder.
- The human is interrupted only for a genuine human-only decision/permission/authority boundary.

## Authoritative documents

Read in this order:

1. `docs/GOVERNANCE.md`
2. `docs/DECISIONS.md`
3. `docs/CURRENT_STATE.md`
4. `ARCHITECTURE.md`
5. `SECURITY.md`
6. `docs/operations-dashboard-contract.md`
7. `docs/UI_ACCEPTANCE_V0.9.md`
8. `ROADMAP.md`

Supporting reset evidence:

- `docs/GOVERNANCE_RESET_2026-08-21.md`
- `docs/REPOSITORY_CLEANUP_MANIFEST.md`
- `docs/CONVERSATION_PRUNING_MANIFEST.md`
- `docs/LOCAL_CHECKOUT_AUDIT_2026-08-21.md`

Use implementation/PR/chat/history/archive only as evidence. The archival tag and local audit ref are explicitly non-authoritative.

## Stable/runtime baseline inherited from reset

- stable release baseline: `v0.9.54`;
- freeze Windows desktop evidence: Workbench 0.9.54;
- cluster/private Workbench health: verified good at audit time;
- outbound Windows bridge: verified online at audit time;
- Blender detected: 5.1.2;
- Unreal Engine detected: 5.8.1;
- no conventional website-style Workbench DEV deployment.

Fresh end-to-end Blender render acceptance and fresh Unreal startup acceptance were not established by the reset.

## Repository hygiene baseline

The reset completed preservation-first repository cleanup:

- stale PR #110 closed unmerged;
- registered operational source checkout realigned to canonical GitHub source;
- unaccepted local relay-lock experiment documented before removal;
- one old local-only SEC-008 commit preserved behind a local audit ref;
- stale Workbench worktrees reduced from eight to one clean main worktree;
- fully merged remote refs deleted only after reachability proof;
- remaining public branch commit graphs preserved behind an archival tag whose checkpoint tree exactly matched canonical main;
- stale active public branch surface reduced to `main` only at cleanup checkpoint.

If a future local checkout differs from canonical `main`, do not treat local state as authority. Preserve/classify unexpected local-only material before realigning it.

## Current architecture/behaviour rules

### Scheduler-native work

- durable queued state;
- scheduler-owned queued → routing dispatch;
- server/CI/Windows/AI execution capacity plus waiting/needs-you state lanes;
- persisted Critical → High → Normal → Low priority then FIFO;
- historical persisted zero priority = Normal;
- truthful measured/stage/indeterminate progress.

### Operations semantics

Actual job execution, project/session presence and terminal operation history are distinct. A recent/active session does not mean each completed operation is still running.

### Remote/direct operations

Direct server/CI/Windows controls and private relay activity are not yet one authoritative scheduler-native job plane. Unknown/non-authoritative fields must remain unknown rather than synthesized.

### Configured execution target fidelity

If a project/task is configured for a runner/target and that target is unavailable, report blocked/unavailable truthfully. Do not silently run locally just to hide target unavailability. Worker location/capability/readiness must be accurate.

### Continuation

Authenticated private-relay durable continuation exists and is tested. Automatic dependency wake-up has live evidence. Full post-validator `wait → wake → useful resumed work → completed` remains unverified.

### Windows

Outbound typed/allowlisted operations only. Do not create generic Windows shell/inbound authority for convenience.

### Model-credit use

Do not spend external/scarce paid model credit for tests, probes or experiments without explicit user authorisation for that spend.

### Product completion language

Do not present a skeleton, prototype, partial UI/backend demo or engineering preview as a finished/production-ready coherent Workbench product.

### Release verification

Merged source is not released. A release is not accepted complete until the actual expected tag/release and required downloadable artifact exist. Release, deployment/installation and semantic verification are separate evidence states.

## Highest-priority known product defect after reset

**P0 — Operations semantic false-running projection.**

0.9.54 maps some individual relay operations whose state is `completed`/`failed` to `TaskRunning` while the surrounding project/ChatGPT session lease is active. This produced misleading live counts such as `Running 100`.

Canonical target:

1. real executing/queued/waiting jobs;
2. session/project presence;
3. recent terminal operation history;

must remain distinct.

Do **not** start the code correction until the governance reset activation gate passes.

Acceptance must prove, among other cases, `completed + session active != running job`. UI responsiveness alone is not semantic acceptance.

## Other post-reset priorities

After the reset is formally complete, follow the current `ROADMAP.md`/decision register. Recorded priorities include:

- authoritative cross-plane job model;
- full unattended-continuation live acceptance proof;
- reliable release publication without no-op retriggers;
- private relay retention/compaction policy;
- fresh Blender typed GPU-render acceptance;
- Unreal five-minute `zen` startup investigation;
- fresh decision on knowledge-graph/searchable-decisions capability;
- fresh decision/design if cross-process relay-state locking is actually required.

## Critical do-not-reintroduce rules

Do not:

- use historical conversations, archive tags or operational checkouts as requirements;
- map session presence/recency to individual running jobs;
- invent queue positions or progress percentages;
- make real remote work disappear because relay history is large;
- silently fall back locally when a configured runner/target is unavailable;
- report worker readiness/location/capability falsely;
- spend external model credit on unapproved testing/probing;
- present partial engineering/prototype work as a finished coherent product;
- add generic Windows shell/inbound authority;
- silently route direct ChatGPT development through OpenClaw;
- treat OpenClaw as the primary/default coder;
- call a release complete without verifying actual expected tag/release and artifact;
- conflate released, deployed/installed and verified;
- treat build/UI health as proof of semantic correctness;
- restore the 90-second Unreal smoke or the removed `TNotNull` crash as the current failure;
- assume Blender GUI preferences determine factory headless GPU rendering;
- normalise no-op main commits as the desired release protocol.

## Development workflow rule

Non-human waits do not terminate active work. Continue useful in-scope work through CI/build/runner/release/deployment waits. Stop only at evidence-backed completion, an inspectable result, or a genuine human-only blocker.

This does not override governance/change-control requirements.

## Exact remaining blocker / development-resume condition

Repository cleanup and accessible memory/context audit are complete. The remaining governance-reset blocker is historical conversation/pruning coverage.

Ordinary development may resume only after either:

1. the remaining historical Workbench conversations are fully enumerated/audited and pruning safety is proven; or
2. the user explicitly accepts the residual risk of inaccessible historical conversations and authorises the reset to close without proving each unseen conversation individually.

Then:

- update `docs/GOVERNANCE_RESET_2026-08-21.md` to COMPLETE with the evidence/risk decision;
- update this handoff to activated/current;
- verify current canonical `main` and clean operational checkout;
- start post-reset product work from canonical `main` only.

Until then: **feature development remains frozen**.
