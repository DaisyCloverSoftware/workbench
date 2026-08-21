# Workbench current state

Last governance verification: **2026-08-21 (BST)**.

This document records what currently exists and how strongly it has been verified. It is not a roadmap. Product intent belongs in the canonical contracts and `docs/DECISIONS.md`.

## Repository and reset baseline

Freeze-point repository baseline:

- repository: `DaisyCloverSoftware/workbench`;
- freeze `main`: `235305bccbef9a35d38445946c4bdab63364f859`;
- preceding substantive 0.9.54 release merge: `0afabdb075818beb0e97d1941ef02e16c14fe795` (PR #222);
- validated PR #222 release head: `b97566b5a773fcb5d0a88066df4633d0c03ba2e3`;
- version/tag source: `v0.9.54`;
- PR #222 required `build`, `runner` and `ui-responsiveness` workflows passed.

Governance/cleanup merge checkpoints completed after the freeze:

- PR #223 — canonical governance baseline → `1c5099a7bc11755377f9c575041500dc25f06caa`;
- PR #224 — preserve/clean inherited relay-lock experiment → `68e7459ce3b2b68eb0875851ecd11dd75ed64f95`;
- PR #225 — audit worktrees/branches and realign stale source checkout → `72d19d14d0af628256b1042a86082dde9e331bcf`;
- PR #226 — remove exact stale local worktrees → `c0c2cae23676b5e6b3d853aae66cce202d508f7b`;
- PR #227 — delete fully merged remote branches only → `83c9218b15aa7c69e29b56455f87bb4dc6fc223c`;
- PR #228 — patch-equivalence / PR-head branch audit → `476499cc3a405f093fe7a93f899421bddcafd9ce`;
- PR #229 — consolidate remaining public historical branch graphs behind an archival tag → `ce4ecd5f0e47d764d6ad4221619390db1ea70af4`.

Every governance/cleanup PR above passed the repository's required `build`, `runner` and `ui-responsiveness` gates on its exact final head before merge. Those gates prove repository/build consistency for the cleanup mechanisms; they do not convert known product-semantic defects into accepted behaviour.

PR #110 (`Add searchable decisions and a project knowledge graph`) was closed **unmerged** during the reset because its implementation basis was Workbench 0.9.10. The capability itself remains an undecided future idea, not rejected product scope.

## Repository hygiene state

At the post-PR #229 cleanup checkpoint:

- canonical public `main`: `ce4ecd5f0e47d764d6ad4221619390db1ea70af4`;
- active public branches: **`main` only**;
- open Workbench pull requests: **zero**;
- registered Workbench source checkout: clean `main` at the canonical checkpoint;
- local Workbench worktrees: **one**;
- a private/local audit ref preserves the one previously local-only pre-reset SEC-008 commit; that local history is not public/canonical authority.

Public branch cleanup was preservation-first:

1. 157 branch refs whose complete tip history was already reachable from `main` were deleted.
2. Remaining refs were audited with `git cherry` and GitHub PR-head provenance; some had patch-equivalent history and others carried genuinely novel commit graphs.
3. Before deleting the remaining public source branch refs, **137 source branch tips** were preserved behind lightweight archival tag `archive/pre-governance-reset-20260821`.
4. The archive points to synthetic history-anchor commits whose checkpoint file tree is exactly the canonical `main` tree at the archive checkpoint.

Verified archive evidence:

- archive tag: `archive/pre-governance-reset-20260821`;
- archive head: `bcb7a1a6056b6f2d4a132bf51dbbf224b57f8832`;
- archive checkpoint tree: `73b84fd417a567ab2a51baf06cc7f6019dde0ac7`;
- PR #229 checkpoint `main` tree: `73b84fd417a567ab2a51baf06cc7f6019dde0ac7`;
- archive operation exit code: 0;
- post-cleanup active remote branch count: 1 (`main`).

The archive is historical/audit preservation only. It MUST NOT be used as a development baseline or requirements source.

## Local checkout findings resolved during reset

The registered operational checkout was initially much older than canonical GitHub source and contained local-only/uncommitted work.

### Relay-state lock experiment — preserved as knowledge, implementation discarded

The reset found one modified relay-state file plus three untracked files implementing a cross-process lock-file experiment. It was unaccepted and could not be tested through the live safe-command policy. Its rationale and risks are preserved in `docs/LOCAL_CHECKOUT_AUDIT_2026-08-21.md`; the working-tree experiment was removed and canonical source restored.

### Stale local lineage — preserved then realigned

The registered checkout's old `main` contained one local-only commit, `2defa97101447c04e8350bfae88414cbacafe237` (`sec: pin GitHub Actions to full commit SHAs (SEC-008)`). Its only code effect—workflow action SHA pinning—already existed in current public source.

The old local commit was preserved behind a local audit ref before operational `main` was realigned to canonical GitHub `main`.

### Stale local worktrees — audited before deletion

Eight local worktrees were found. Six old detached worktrees were clean. One old named file-mode worktree was dirty in exactly two tracked files; bounded diff/blob inspection proved those exact working blobs were already published by public commit `0b601caab1859e42767c5019ba61c01cf3af8c55`.

Only after that proof were the duplicate edits restored and all seven secondary worktrees removed. Post-cleanup verification proved one clean worktree remained.

## Stable/live surfaces

Workbench has no canonical website-style DEV deployment. See WB-DEC-009.

### Stable release

- current stable version at the reset baseline: `v0.9.54`;
- release build surfaces include `Workbench.exe`, `Workbench-Updater.exe` and Linux runner/server/relay assets;
- the freeze-point handoff records the official 0.9.54 release as published.

Release process remains imperfect: version-bump merges have sometimes required an identical-tree no-op `main` push to retrigger publication. This is a known defect/workaround, not the intended permanent release protocol.

### Cluster live

Privacy-safe live checks during the reset verified:

- private Workbench capability manifest advertises 0.9.54;
- Workbench MCP and relay services active;
- loopback MCP healthy;
- bounded Workbench health script `overall=ok`;
- registered cluster nodes Ready at the check time;
- no attached unhealthy storage volumes reported at the check time.

### Windows live

Freeze evidence showed Workbench desktop 0.9.54. A fresh outbound bridge check verified a Windows host online with Blender 5.1.2 and Unreal Engine 5.8.1 detected.

That detection proves bridge/tool presence only; it does not upgrade application-specific acceptance claims below.

## Current Operations/scheduler implementation

### Scheduler-native durable tasks — implemented/tested/merged/released

Current source includes:

- durable queued state;
- scheduler-owned queued → routing dispatch;
- server/CI/Windows/AI execution capacity plus waiting/needs-you state lanes;
- persisted Critical → High → Normal → Low priority then FIFO;
- historical zero-value priority interpreted as Normal;
- truthful measured, stage-based and indeterminate progress;
- `WorkItem` projection with lane/priority/progress/dependency/executor metadata.

### Operations semantic acceptance — FAILED / inherited P0 defect

Workbench 0.9.54 can convert a remote event whose individual state is `completed` or `failed` into `TaskRunning` when the surrounding project/session `Active` lease is true. This produced misleading live counts such as `Running 100`.

Canonical requirement: individual job execution, project/session presence and recent terminal operation history are separate concepts. Status of 0.9.54 Operations semantics: **released/live but not accepted/verified correct**.

No product correction was made during the governance freeze.

## Private relay and unattended continuation

### Bounded live activity projection — current

The live Dashboard-facing relay projection keeps pending and recent matching request/result activity without reading all historical completed transport content on every refresh. It is a scaling/projection rule, not a deletion/retention policy.

### Historical relay transport — intentionally retained

Underlying private relay history was **not mass-deleted** during this reset. The history is intentionally retained pending a separate retention/compaction policy. This is not a public-source repository-cleanup failure.

### Authenticated durable continuation — partially live-verified

Current source HMAC-seals private-relay continuation authority and validates correlation/project/original continuation. Workbench may append its owned dependency-result suffix; arbitrary appended content remains fail-closed.

Evidence:

- implementation/tests/merge/release: yes;
- automatic dependency wake without a new chat message: previously observed live;
- clean post-validator proof of full `waiting_dependency → automatic resume → useful work → completed`: **UNVERIFIED**.

## Windows bridge, Blender and Unreal

### Security/control model — current

Outbound typed/allowlisted bridge only. Generic remote Windows shell authority is not part of the intended interface.

### Blender

Current source configures Cycles GPU/OptiX explicitly for the headless render path. The reset freshly verified Blender 5.1.2 detection but did not establish a fresh current end-to-end GPU render proof. Current live render acceptance: **UNVERIFIED by this reset**.

### Unreal

The old 5.8.1 `TNotNull`/stack-overflow startup crash is historical and removed. The current bounded smoke uses a five-minute diagnostic window. Latest inherited evidence before reset: process survived the old 90-second boundary and reached five-minute timeout classified `zen`.

Fresh reset verification confirmed Unreal 5.8.1 detection, not a new startup result. Current startup root cause/acceptance: **UNRESOLVED / UNVERIFIED beyond inherited `class=zen` evidence**.

## Durable memory/context audit

The accessible Workbench durable project context and project/global memory store were audited during the reset, not merely the memory implementation.

The accessible store contained a small set of durable memories plus the current context capsule. Important rules recovered from that store and accessible prior-work context are now canonicalised in `docs/DECISIONS.md`, including:

- canonical repository state outranks memory/context;
- no conventional Workbench DEV environment is invented by generic “DEV checkpoint” wording;
- configured runner/target availability, location and readiness must be truthful;
- no external model-credit/scarce paid test invocation without explicit user authorisation;
- do not present a skeleton/prototype/engineering preview as finished Workbench;
- release/tag/artifact verification is distinct from deployment/semantic verification;
- operational checkouts are copies, not product authority.

Old bootstrap/private-relay memories remain advisory/history only.

## Durable state/storage

Workbench Core durable state is private local JSON (`State.Version` 3 at this baseline), using decode/normalisation and private temporary-file + atomic rename writes. It persists project registry, tasks, encrypted secret references, preferences, scheduler priority/progress and related durable task state.

This is a file-format/state-version model, not a relational database migration system.

## Remaining unverified / unresolved items

Do not infer these as solved:

- full-content inventory of every historical Workbench project conversation;
- conversation-pruning safety for unseen/unrecoverable chats;
- P0 Operations false-running correction;
- authoritative cross-plane job model;
- exact private-relay long-term retention/compaction policy;
- full post-validator unattended continuation productive-completion proof;
- fresh Blender end-to-end GPU render acceptance;
- fresh Unreal five-minute startup/root-cause result;
- release-publication reliability without no-op retriggers;
- cross-process relay-state locking requirement under the actual supported process topology;
- future knowledge-graph/searchable-decisions product shape.

Repository/worktree/branch hygiene and accessible Workbench memory/context audit are no longer blind spots. The governance reset remains **INCOMPLETE solely because the historical-conversation/pruning proof cannot be completed with the available conversation access without explicit residual-risk acceptance or additional source access**.
