# Workbench current state

Last governance verification: **2026-08-21 (BST)**.

This document records what currently exists and how strongly it has been verified. Product intent belongs in canonical contracts and `docs/DECISIONS.md`.

## Governance reset state

The 2026-08-21 governance reset is **COMPLETE**.

The final pre-closure reset checkpoint was `25e8c106acefc54adea5df39ea53b5a6f4d1336b` (PR #230). At 2026-08-21 12:33 BST the user explicitly accepted the residual risk that the available interfaces cannot enumerate/read every historical Workbench conversation in full and authorised development to move forward.

That acceptance does not pretend unseen conversations were audited. Instead, current governance makes canonical repository state authoritative and makes historical conversations non-authoritative evidence. Ordinary development does not need to reread old chats; resurfaced historical material cannot silently override current decisions/contracts.

See `docs/GOVERNANCE_RESET_2026-08-21.md` and `docs/CONVERSATION_PRUNING_MANIFEST.md`.

## Freeze/release baseline

- repository: `DaisyCloverSoftware/workbench`;
- freeze `main`: `235305bccbef9a35d38445946c4bdab63364f859`;
- preceding substantive 0.9.54 release merge: `0afabdb075818beb0e97d1941ef02e16c14fe795` (PR #222);
- validated PR #222 release head: `b97566b5a773fcb5d0a88066df4633d0c03ba2e3`;
- stable version/tag baseline: `v0.9.54`;
- PR #222 required `build`, `runner` and `ui-responsiveness` workflows passed.

Governance/cleanup merge checkpoints:

- #223 canonical governance baseline → `1c5099a7bc11755377f9c575041500dc25f06caa`;
- #224 relay-experiment preservation/cleanup → `68e7459ce3b2b68eb0875851ecd11dd75ed64f95`;
- #225 worktree/branch audit + stale checkout realignment → `72d19d14d0af628256b1042a86082dde9e331bcf`;
- #226 stale-worktree cleanup → `c0c2cae23676b5e6b3d853aae66cce202d508f7b`;
- #227 fully merged remote-ref cleanup → `83c9218b15aa7c69e29b56455f87bb4dc6fc223c`;
- #228 patch-equivalence / PR-head branch audit → `476499cc3a405f093fe7a93f899421bddcafd9ce`;
- #229 historical branch archive consolidation → `ce4ecd5f0e47d764d6ad4221619390db1ea70af4`;
- #230 final canonical reset record → `25e8c106acefc54adea5df39ea53b5a6f4d1336b`.

Every reset PR passed exact-final-head `build`, `runner` and `ui-responsiveness` gates before merge. These prove repository/build consistency, not semantic correctness of inherited defects.

PR #110 (`Add searchable decisions and a project knowledge graph`) was closed unmerged because its implementation basis was Workbench 0.9.10. The capability remains an undecided future idea.

## Repository hygiene baseline

At the final reset checkpoint:

- active public branches: `main` only;
- open pull requests/issues: zero;
- registered operational checkout: clean and aligned to canonical `main`;
- local Workbench worktrees: one;
- one old local-only SEC-008 commit preserved behind a local audit ref, non-authoritative;
- remaining old public branch histories preserved behind `archive/pre-governance-reset-20260821`, historical only.

The reset removed stale active-looking development surfaces only after preservation/reachability proof. Operational/local state does not outrank canonical GitHub source.

## Stable/live surfaces

Workbench has no canonical website-style DEV deployment.

### Stable release

- reset baseline stable version: `v0.9.54`;
- release surfaces include `Workbench.exe`, `Workbench-Updater.exe` and Linux runner/server/relay assets;
- release publication remains imperfect: version-bump merges have sometimes needed an identical-tree no-op `main` push to retrigger publication. That is a known defect/workaround, not the intended design.

### Cluster live

Privacy-safe reset checks verified Workbench services/MCP healthy and registered cluster nodes Ready at check time.

### Windows live

Freeze evidence showed Workbench desktop 0.9.54. A fresh outbound bridge check verified a Windows host online with Blender 5.1.2 and Unreal Engine 5.8.1 detected. Detection does not prove application-specific acceptance.

## Operations/scheduler implementation

### Scheduler-native durable tasks — implemented/tested/merged/released

Current source includes:

- durable queued state;
- scheduler-owned queued → routing dispatch;
- server/CI/Windows/AI execution capacity plus waiting/needs-you lanes;
- persisted Critical → High → Normal → Low priority then FIFO;
- historical zero-value priority interpreted as Normal;
- truthful measured, stage-based and indeterminate progress;
- `WorkItem` projection with lane/priority/progress/dependency/executor metadata.

### Operations semantic acceptance — FAILED / P0 corrections-round target

Workbench 0.9.54 can convert a remote event whose individual state is `completed` or `failed` into `TaskRunning` when the surrounding project/session `Active` lease is true. This produced misleading live counts such as `Running 100`.

Canonical requirement: individual job execution, project/session presence and recent terminal operation history are separate concepts. Status of 0.9.54 Operations semantics: **released/live but not accepted/verified correct**.

The reset did not fix this. Reset closure now authorises it as the first corrections-round product change. Acceptance must explicitly prove `completed + session active != running job` and `failed + session active != running job`.

## Private relay and continuation

- Dashboard-facing relay projection is bounded; historical volume does not define current live Operations state.
- underlying private relay history remains intentionally retained pending a retention/compaction policy;
- authenticated durable continuation is implemented/tested and automatic dependency wake has previous live evidence;
- full clean post-validator proof of `waiting_dependency → automatic resume → useful work → completed` remains **UNVERIFIED**.

## Windows bridge, Blender and Unreal

- security/control model: outbound typed/allowlisted bridge only; no generic Windows shell;
- Blender source explicitly configures Cycles GPU/OptiX for headless rendering, but fresh current end-to-end GPU render acceptance remains **UNVERIFIED**;
- old Unreal 5.8.1 `TNotNull`/stack-overflow startup crash is historical and removed;
- current Unreal bounded smoke uses five minutes; latest inherited evidence reached timeout classified `zen`; fresh root cause/acceptance remains **UNRESOLVED / UNVERIFIED**.

## Durable memory/context state

Accessible Workbench durable project/global memory and current context were audited during the reset. Material rules recovered there were promoted into canonical documentation. Old bootstrap/private-relay memories remain advisory/history only.

Historical ChatGPT conversations do not need to be reread for ordinary decisions after reset closure. If old material is deliberately consulted, it is evidence only until accepted into the canonical record.

## Remaining product discrepancies / unresolved work

- **P0 Operations false-running correction** — first active corrections-round item;
- authoritative cross-plane job model;
- exact private-relay retention/compaction policy;
- full unattended continuation productive-completion proof;
- fresh Blender end-to-end GPU render acceptance;
- fresh Unreal five-minute startup/root-cause result;
- release-publication reliability without no-op retriggers;
- cross-process relay-state locking requirement under actual supported topology;
- future knowledge-graph/searchable-decisions product shape.

Historical-conversation full enumeration remains technically unavailable, but is **no longer a governance/development blocker** because the residual risk was explicitly accepted and repository authority is canonical.

## Active development workflow

Post-reset development begins with corrections, not new feature work. Each correction/sprint must produce an inspectable result, then an observation-driven corrections round and user sign-off before the next sprint.
