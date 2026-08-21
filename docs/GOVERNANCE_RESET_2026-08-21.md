# Workbench governance reset — 2026-08-21

Status: **COMPLETE — CLOSED BY EXPLICIT RESIDUAL-CONVERSATION-RISK ACCEPTANCE**

The governance reset is complete. Ordinary product development is authorised to resume from canonical `main`, beginning with the corrections-first workflow defined by current governance.

## Closure decision

At **2026-08-21 12:33 BST**, after the repository, documentation, release/PR history, accessible Workbench memory/context, operational checkout, worktrees, remote branches and recoverable historical project context had been audited, the user explicitly accepted the residual risk that the available interfaces cannot enumerate/read every historical Workbench conversation in full and authorised the project to move forward.

This closes the only remaining reset blocker recorded by PR #230.

The acceptance does **not** assert that unseen conversations were read. Instead it establishes the permanent operating rule that makes such access unnecessary for normal development:

- canonical repository requirements and decisions are the project authority;
- historical conversations are non-authoritative evidence only;
- future decisions do not require rereading historical chats;
- an old or newly surfaced chat statement cannot silently override current canonical documents;
- if historical evidence is intentionally reintroduced, it must be evaluated against current canonical state and, if accepted, recorded through the normal governance/decision process before it becomes binding;
- absence of access to an old conversation is therefore not a reason to reconstruct requirements from guesswork or to halt ordinary development.

This is the mechanism that prevents Workbench from drifting backwards into superseded conversation state.

## Purpose and result

The reset reconstructed and canonicalised current Workbench intent/state so future work can proceed from the repository rather than old conversations. Superseded/rejected decisions and historical implementation evidence were preserved without leaving stale branches/worktrees as active-looking development surfaces.

### Freeze baseline

- freeze time: approximately 2026-08-21 03:09 BST;
- repository: `DaisyCloverSoftware/workbench`;
- freeze `main`: `235305bccbef9a35d38445946c4bdab63364f859`;
- preceding substantive 0.9.54 merge: `0afabdb075818beb0e97d1941ef02e16c14fe795` (PR #222);
- PR #222 validated head: `b97566b5a773fcb5d0a88066df4633d0c03ba2e3`;
- stable version baseline: `v0.9.54`;
- initial open PR: #110 only; initial open issues: none; initial remote branch inventory: 288;
- privacy-safe cluster/Workbench health was good at audit time;
- Windows bridge was online with Blender 5.1.2 and Unreal Engine 5.8.1 detected;
- freeze evidence showed Windows desktop Workbench 0.9.54;
- no conventional website-style Workbench DEV deployment exists.

## Governance / cleanup chain

Every reset PR below passed exact-final-head `build`, `runner` and `ui-responsiveness` gates before merge:

- #223 canonical governance baseline → `1c5099a7bc11755377f9c575041500dc25f06caa`;
- #224 local relay-experiment preservation/cleanup → `68e7459ce3b2b68eb0875851ecd11dd75ed64f95`;
- #225 worktree/branch audit + stale checkout realignment → `72d19d14d0af628256b1042a86082dde9e331bcf`;
- #226 exact stale-worktree cleanup → `c0c2cae23676b5e6b3d853aae66cce202d508f7b`;
- #227 fully merged remote-ref cleanup → `83c9218b15aa7c69e29b56455f87bb4dc6fc223c`;
- #228 patch-equivalence/PR-head audit → `476499cc3a405f093fe7a93f899421bddcafd9ce`;
- #229 historical branch archive consolidation → `ce4ecd5f0e47d764d6ad4221619390db1ea70af4`;
- #230 final canonical record / conversation blocker isolation → `25e8c106acefc54adea5df39ea53b5a6f4d1336b`.

Passing these gates proves reset/cleanup mechanisms remained repository-consistent; it does not prove inherited product-semantic defects correct.

## Audit coverage

The reset inspected/reconciled:

- canonical repository source and root/docs contracts;
- Core scheduler/work-item semantics and durable state;
- Operations projection semantics;
- authenticated continuation contract/tests;
- relevant PR/release/workflow history;
- public branch history with preservation-first reachability audit;
- registered operational checkout and all discovered local Workbench worktrees;
- local-only/uncommitted inherited source, preserved/classified before cleanup;
- private capability guide/manifest without widening runtime authority;
- privacy-safe live cluster and Windows bridge state;
- accessible Workbench durable project/global memory and current context;
- recoverable prior Workbench project-context summaries/material;
- sampled File Library Workbench artifacts.

The one source that could not be exhaustively proven was every full historical ChatGPT conversation body. That limitation remains recorded in `docs/CONVERSATION_PRUNING_MANIFEST.md`, but is no longer a development blocker because the user explicitly accepted the residual risk and the repository-authority rule prevents unseen chats from silently governing the product.

## Repository cleanup result

Public source / registered checkout cleanup completed to the reset's preservation rules:

- PR #110 closed unmerged;
- stale local relay-lock experiment documented then removed;
- stale local lineage preserved behind an audit ref then operational checkout realigned;
- eight local Workbench worktrees audited and reduced to one clean `main` worktree;
- 157 fully merged remote refs deleted after reachability proof;
- remaining public branch histories preserved behind `archive/pre-governance-reset-20260821` before source-branch deletion;
- active public branch surface reduced to `main` only at the cleanup checkpoint;
- private relay historical transport intentionally retained pending a real retention policy.

See `docs/REPOSITORY_CLEANUP_MANIFEST.md` and `docs/LOCAL_CHECKOUT_AUDIT_2026-08-21.md` for detailed evidence.

## Inherited product discrepancies — reset did not fix these

1. **P0 — Operations false-running projection** (`completed`/`failed` + active session can become `TaskRunning`).
2. Cross-plane authoritative job model gap.
3. Release-publication reliability/no-op retrigger defect.
4. Exact private-relay retention/compaction policy unresolved.
5. Full post-validator unattended-continuation productive-completion proof missing.
6. Fresh Blender end-to-end GPU-render acceptance missing.
7. Unreal five-minute `zen` startup/root cause unresolved.
8. Cross-process relay-state locking requirement remains an open technical question; discovered local experiment was not accepted.
9. Knowledge-graph/searchable-decisions future shape remains undecided after closing old PR #110 unmerged.

## Completion gate status

- [x] Feature development frozen before reset changes.
- [x] Starting repository/release/runtime baseline recorded.
- [x] Relevant repository documentation inventoried/reconciled.
- [x] Accessible Workbench durable project/global memory and current context audited.
- [x] Important accessible decisions extracted/classified and conflicts reconciled.
- [x] Current requirements/authority hierarchy documented.
- [x] Significant superseded/rejected/do-not-reintroduce behaviours documented.
- [x] Architecture/security/operations/harness/knowledge documentation reconciled.
- [x] Current implementation/release/runtime state documented with evidence levels.
- [x] Repository compared against highest-risk current requirements.
- [x] Stale/obsolete repository material preservation-audited and cleaned.
- [x] Registered checkout/worktree cleanup completed and verified clean.
- [x] Remote branch cleanup completed with preservation-first archive evidence.
- [x] Governance/cleanup PR gates passed before each merge.
- [x] Remaining inherited product failures/discrepancies recorded rather than silently fixed.
- [x] Permanent governance rules added.
- [x] Historical-conversation access limitation explicitly recorded rather than falsely claimed solved.
- [x] Residual risk of inaccessible historical conversations explicitly accepted by the user on 2026-08-21.
- [x] Future chats/history explicitly non-authoritative unless reintroduced through canonical governance.
- [x] Fresh repository-based post-reset handoff exists.
- [x] Future ordinary development authorised.

## Development resumes here

The first post-reset product work MUST be a **corrections round**, not a new feature sprint. The first correction is the P0 Operations false-running semantic defect.

The corrections-first cycle is:

1. define observable acceptance against canonical requirements;
2. implement and test the correction without removing approved behaviour;
3. take it through required CI/release/deployment or installation gates for the surface being changed;
4. provide a genuinely inspectable result to the user;
5. record observation-driven corrections;
6. obtain sign-off before advancing to the next product sprint.

Historical conversations are not part of this bootstrap path. Start from current canonical `main`, `docs/GOVERNANCE.md`, `docs/DECISIONS.md`, `docs/CURRENT_STATE.md`, the applicable contract/tests, and current implementation evidence.
