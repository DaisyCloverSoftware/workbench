# Workbench development handoff — canonical post-reset baseline

**Activation status: AUTHORISED — GOVERNANCE RESET CLOSED 2026-08-21.**

This handoff is based on the canonical repository record, not historical conversations. At 2026-08-21 12:33 BST the user explicitly accepted the residual risk that not every historical Workbench conversation can be enumerated/read in full and authorised development to move forward.

Historical conversations are therefore **not required bootstrap material**. They are non-authoritative evidence only and cannot silently override current canonical repository state.

## Bootstrap rule

Always begin by reading current canonical `main`; do not treat the SHA in an old handoff as permanent truth. Verify the current `main` HEAD at the start of a fresh development session.

The reset's final pre-closure canonical checkpoint was:

- `main`: `25e8c106acefc54adea5df39ea53b5a6f4d1336b` (PR #230);
- active public branch surface: `main` only;
- open PRs/issues at that checkpoint: zero;
- archival tag: `archive/pre-governance-reset-20260821` — historical preservation only, never a development baseline.

The reset-closure PR advances `main` beyond that checkpoint. Always bootstrap from then-current canonical `main`.

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

Reset evidence:

- `docs/GOVERNANCE_RESET_2026-08-21.md`
- `docs/REPOSITORY_CLEANUP_MANIFEST.md`
- `docs/CONVERSATION_PRUNING_MANIFEST.md`
- `docs/LOCAL_CHECKOUT_AUDIT_2026-08-21.md`

Implementation, PRs, chats, memory and archives are evidence only. If historical evidence appears to conflict with current canonical documents, the canonical record wins until a new conscious decision updates it.

## What Workbench is

Workbench is a ChatGPT-first developer/operations control plane and durable execution bridge.

- ChatGPT is the primary reasoning/coding brain.
- Workbench provides durable project/task state, safe repository eyes/hands, scheduling/execution infrastructure, private transport, bounded machine controls and an outbound typed Windows bridge.
- OpenClaw/other autonomous harnesses are optional operator/worker capacity, not the default coder.
- The human is interrupted only for a genuine human-only decision, permission or authority boundary.

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

The reset completed preservation-first cleanup:

- stale PR #110 closed unmerged;
- registered operational source checkout realigned to canonical GitHub source;
- unaccepted local relay-lock experiment documented before removal;
- one old local-only SEC-008 commit preserved behind a local audit ref;
- stale Workbench worktrees reduced from eight to one clean main worktree;
- fully merged remote refs deleted only after reachability proof;
- remaining public branch commit graphs preserved behind an archival tag whose checkpoint tree matched canonical main;
- active public branch surface reduced to `main` only at the cleanup checkpoint.

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

Direct server/CI/Windows controls and private relay activity are not yet one authoritative scheduler-native job plane. Unknown/non-authoritative fields remain unknown rather than synthesized.

### Configured execution target fidelity

If a configured runner/target is unavailable, report blocked/unavailable truthfully. Do not silently run locally merely to hide unavailability. Worker location/capability/readiness must be accurate.

### Continuation

Authenticated private-relay durable continuation exists and is tested. Automatic dependency wake-up has live evidence. Full post-validator `wait → wake → useful resumed work → completed` remains unverified.

### Windows

Outbound typed/allowlisted operations only. Do not create generic Windows shell/inbound authority for convenience.

### Model-credit use

Do not spend external/scarce paid model credit for tests, probes or experiments without explicit user authorisation for that spend.

### Completion language

Do not present a skeleton, prototype, partial UI/backend demo or engineering preview as a finished/production-ready coherent Workbench product.

### Release verification

Merged source is not released. Release/tag/artifact, deployment/installation and semantic verification are separate evidence states.

## Mandatory post-reset workflow

Development resumes with a **corrections round before any new feature sprint**.

For every correction/sprint:

1. define observable acceptance from canonical requirements;
2. implement and test without removing/reinterpreting approved behaviour;
3. update decision/contract/acceptance documentation whenever a material decision changes;
4. take the work through applicable CI/release/deployment or installation gates;
5. provide the user with something genuinely inspectable on the correct surface;
6. record observation-driven corrections and execute them;
7. obtain user sign-off on the inspectable result before advancing to the next sprint.

Non-human waits do not terminate active work. Continue useful in-scope work through CI/build/runner/release/deployment waits. Stop only at evidence-backed completion, an inspectable result, or a genuine human-only blocker.

## First corrections round — P0

**Operations semantic false-running projection.**

0.9.54 maps some individual relay operations whose state is `completed`/`failed` to `TaskRunning` while the surrounding project/ChatGPT session lease is active. This produced misleading live counts such as `Running 100`.

Canonical acceptance must separate:

1. real executing/queued/waiting jobs;
2. session/project presence;
3. recent terminal operation history.

At minimum, tests must prove `completed + session active != running job` and `failed + session active != running job`. UI responsiveness alone is not semantic acceptance.

This is the first authorised product correction after reset closure.

## Other recorded post-reset priorities

After the corrections round is inspected and signed off, follow current `ROADMAP.md`/decision register. Recorded priorities include:

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
- require old chats to be reread before ordinary decisions;
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
