# Workbench development handoff — canonical reset baseline

**Activation status: NOT YET AUTHORISED.**

This handoff is intentionally based on the canonical repository record rather than historical conversations. Do not resume ordinary feature development until `docs/GOVERNANCE_RESET_2026-08-21.md` is changed to COMPLETE with evidence.

## What Workbench is

Workbench is a ChatGPT-first developer/operations control plane and durable execution bridge.

- ChatGPT is the primary reasoning/coding brain.
- Workbench provides durable project/task state, safe repository eyes/hands, scheduler/execution infrastructure, private transport, bounded machine controls and an outbound typed Windows bridge.
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

Use implementation/PR/chat history only as evidence. Conversations are not project authority.

## Exact inherited repository/release state

Freeze baseline:

- `main`: `235305bccbef9a35d38445946c4bdab63364f859`
- preceding substantive release merge: `0afabdb075818beb0e97d1941ef02e16c14fe795` (PR #222)
- PR #222 validated release head: `b97566b5a773fcb5d0a88066df4633d0c03ba2e3`
- current version tag/source: `v0.9.54`
- required PR #222 build/runner/UI-responsiveness workflows: passed
- open PRs: #110 only (stale 0.9.10-based knowledge-graph implementation; do not merge as-is)
- open issues: none at reset baseline
- formal website-style DEV deployment: none established

The freeze `main` HEAD is an identical-tree no-op release-publication retrigger. Treat that as release-process evidence, not product code.

## Runtime baseline

At reset verification:

- cluster/private capability version: 0.9.54;
- privacy-safe Workbench MCP/relay health: good;
- registered cluster nodes: Ready at the check time;
- outbound Windows bridge: online;
- Blender detected: 5.1.2;
- Unreal Engine detected: 5.8.1;
- freeze-point Windows desktop observation: Workbench 0.9.54.

Do not copy private host/topology/path details into public source.

## Current architecture

### Scheduler-native tasks

- durable queued state;
- scheduler-owned queued → routing dispatch;
- scheduler execution lanes/capacity;
- persisted Critical/High/Normal/Low priority then FIFO;
- zero-value persisted priority = Normal;
- truthful measured/stage/indeterminate progress;
- waiting dependency/retry and needs-attention states.

### Remote/direct operations

Server/CI/Windows direct controls and private relay transport are not yet one native authoritative scheduler job plane. The UI may project them, but session/activity recency is not execution state.

### Continuation

Authenticated private-relay durable continuation exists and is tested. Automatic dependency wake-up has live evidence. Full post-validator `wait → wake → useful resumed work → completed` remains unverified.

### Windows

Outbound typed/allowlisted operations only. No generic remote Windows shell/inbound listener for convenience.

## Highest-priority known defect after reset

**P0 — Operations semantic false-running projection.**

0.9.54 maps some individual relay operations whose state is `completed`/`failed` to `TaskRunning` when their project/ChatGPT session lease remains active. This produced misleading live counts such as `Running 100`.

Canonical target:

1. real executing/queued/waiting jobs;
2. session/project presence;
3. recent terminal operation history;

must remain distinct.

Do not start the code correction until the governance reset activation gate passes.

Acceptance must prove, among other cases, `completed + session active != running job`. UI responsiveness alone is not semantic acceptance.

## Other post-reset priorities

After the reset is formally complete, use `ROADMAP.md` priority order. The next recorded work includes:

- authoritative cross-plane job model;
- full unattended-continuation live acceptance proof;
- reliable release publication without no-op retriggers;
- private relay retention policy;
- fresh Blender typed GPU-render acceptance;
- Unreal five-minute `zen` startup investigation;
- fresh decision on knowledge-graph capability before touching PR #110.

## Critical do-not-reintroduce rules

Do not:

- use historical conversations as requirements;
- map session presence/recency to individual running jobs;
- invent queue positions or progress percentages;
- make real remote work disappear because relay history is large;
- add generic Windows shell/inbound authority for convenience;
- silently route direct ChatGPT development through OpenClaw;
- treat OpenClaw as the primary/default coder;
- call release/build/UI-health success equivalent to target semantic verification;
- restore the 90-second Unreal smoke or the removed `TNotNull` crash as the current failure;
- assume Blender GUI preferences determine factory headless GPU rendering;
- normalise no-op main commits as the intended release process.

## Development workflow rule

Non-human waits do not terminate an active workflow. Continue useful in-scope work through CI/build/runner/release/deployment waits. Stop only at evidence-backed completion, an inspectable pushed/deployed result, or a genuine human-only blocker.

This does not override governance/change-control requirements.

## Exact development-resume condition

Before any ordinary product work resumes:

- merge/accept the governance record after required checks;
- resolve the unchecked items in `docs/GOVERNANCE_RESET_2026-08-21.md` or explicitly record accepted residual risk;
- record final clean repository/branch/worktree/pruning status;
- change the reset status to COMPLETE;
- update this handoff's activation status and exact `main` SHA to the post-reset merged commit.

Until then: **feature development remains frozen**.
