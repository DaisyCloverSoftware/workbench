# Workbench governance reset — 2026-08-21

Status: **INCOMPLETE — REPOSITORY CLEAN, CONVERSATION-PRUNING PROOF BLOCKED**

Ordinary feature development remains frozen. The repository, operational checkout, worktrees, branch refs and accessible Workbench memory/context have now been audited and cleaned/canonicalised to the evidence level available. The remaining reset blocker is narrower: the available interfaces still cannot enumerate/read every historical Workbench project conversation in full, so conversation pruning cannot yet be proven safe globally.

Missing evidence is not converted into success by assumption.

## Purpose

Reconstruct and canonicalise current Workbench intent/state so future work can proceed from the repository rather than old conversations. Preserve superseded/rejected decisions and historical implementation evidence without leaving stale branches/worktrees as active-looking development surfaces.

## Freeze-point baseline

- freeze time: approximately 2026-08-21 03:09 BST;
- repository: `DaisyCloverSoftware/workbench`;
- freeze `main`: `235305bccbef9a35d38445946c4bdab63364f859`;
- preceding substantive 0.9.54 merge: `0afabdb075818beb0e97d1941ef02e16c14fe795` (PR #222);
- PR #222 validated head: `b97566b5a773fcb5d0a88066df4633d0c03ba2e3`;
- `v0.9.54` source resolves to canonical version 0.9.54;
- PR #222 required build/runner/UI-responsiveness gates passed;
- initial open PR: #110 only;
- initial open issues: none;
- initial remote branch inventory: 288;
- privacy-safe cluster/Workbench health: good at audit time;
- fresh Windows bridge inventory: host online; Blender 5.1.2 and Unreal Engine 5.8.1 detected;
- freeze evidence: Windows desktop Workbench 0.9.54;
- no conventional website-style Workbench DEV deployment established.

## Development freeze rule

During this reset no ordinary feature/product work is authorised, including:

- Dashboard semantic correction;
- scheduler redesign;
- Blender/Unreal debugging;
- release-process correction;
- knowledge-graph implementation;
- unrelated refactors/features.

Permitted work: governance, documentation, evidence collection, reset-validation tooling and preservation-first repository cleanup.

## Governance / cleanup PR chain

All of the following passed exact-final-head `build`, `runner` and `ui-responsiveness` gates before merge:

- #223 canonical governance baseline → `1c5099a7bc11755377f9c575041500dc25f06caa`;
- #224 local relay-experiment preservation/cleanup → `68e7459ce3b2b68eb0875851ecd11dd75ed64f95`;
- #225 worktree/branch audit + stale checkout realignment → `72d19d14d0af628256b1042a86082dde9e331bcf`;
- #226 exact stale-worktree cleanup → `c0c2cae23676b5e6b3d853aae66cce202d508f7b`;
- #227 fully merged remote-ref cleanup → `83c9218b15aa7c69e29b56455f87bb4dc6fc223c`;
- #228 patch-equivalence/PR-head audit → `476499cc3a405f093fe7a93f899421bddcafd9ce`;
- #229 historical branch archive consolidation → `ce4ecd5f0e47d764d6ad4221619390db1ea70af4`.

Passing these gates proves the governance/cleanup mechanisms did not break normal repository builds; it does not prove inherited product-semantic defects correct.

## Audit coverage record

| Source/category | Coverage | Final reset result / remaining blind spot |
| --- | --- | --- |
| Repository root/current `main` | INSPECTED | Exact freeze/current checkpoints and canonical source verified. |
| Root/docs contracts | INSPECTED + RECONCILED | Governance, decisions, architecture, security, roadmap, UI/Operations, harness, relay and knowledge docs reconciled. |
| Current Core scheduler/work-item source | INSPECTED | Lane, priority, progress, queue ownership/storage semantics verified. |
| Dashboard Operations projection | INSPECTED | 0.9.54 terminal-operation + active-session → Running defect confirmed and recorded, not fixed. |
| Durable Core state store | INSPECTED | JSON State.Version 3, normalisation and atomic-write model verified. |
| Authenticated private continuation | INSPECTED | HMAC validator/tests/current contract verified; full productive live resume still unverified. |
| Relevant PR/release history | INSPECTED | Recent release/feature PR chain audited; PR #110 closed unmerged as stale implementation basis. |
| Workflow configuration | INSPECTED | Release-request/release/build/runner/UI workflows audited; release retrigger defect recorded. |
| Remote branch history | FULL REACHABILITY/PRESERVATION AUDIT | First removed 157 fully merged refs; remaining histories classified; final 137 source refs archived before deletion. Active branch surface reduced to `main` only at cleanup checkpoint. |
| Public historical branch archive | VERIFIED | Tag `archive/pre-governance-reset-20260821`; archive head `bcb7a1a...`; archive checkpoint tree exactly equals PR #229 `main` checkpoint tree `73b84fd...`; all removed source tips proven reachable before deletion. |
| Registered operational checkout | VERIFIED + CLEANED | Old dirty/uncommitted work preserved/classified, stale lineage realigned, final status clean. |
| Local worktrees | FULLY INVENTORIED + CLEANED | Eight discovered; duplicate/dirty work proven already published before restoration; seven stale secondary worktrees removed; one clean main worktree remains. |
| Local-only source history | PRESERVED | One old SEC-008 local-only commit preserved behind local audit ref before realignment; not public/canonical authority. |
| Private capability guide/manifest | INSPECTED + RECONCILED | Windows typed-operation wording aligned without widening authority. |
| Private relay historical transport | INTENTIONALLY RETAINED | Live projection bounded; underlying history retained pending future retention policy. Not a public repo-cleanup blocker. |
| Live cluster health | VERIFIED (bounded/privacy-safe) | Workbench services/MCP healthy; registered nodes Ready at audit time. |
| Windows bridge | VERIFIED (bounded/privacy-safe) | Host/tool detection verified; fresh Blender/Unreal application acceptance not established. |
| Workbench durable context/memory contents | AUDITED (accessible store) | Current context plus accessible project/global memories inspected; material recovered rules canonicalised. |
| File Library Workbench artifacts | INVENTORIED/SAMPLED | Older 0.6.x/0.7.x material classified historical. |
| Historical project conversations | PARTIAL / BLOCKED | Recent summaries/current handoff and some recoverable prior context audited, but full enumeration/content access is unavailable. This is the remaining reset blocker. |

## Canonical decisions recovered during final audit

In addition to the earlier reset decisions, accessible durable memory/prior context recovered rules now made explicit in `docs/DECISIONS.md`:

- configured runner/target unavailability must be truthfully blocked/unavailable, not silently hidden by local fallback;
- worker location/capability/readiness must be truthful;
- external/scarce paid model-credit tests/experiments require explicit user authorisation;
- skeletons/prototypes/partial engineering previews must not be presented as a finished coherent product;
- actual release/tag and expected artifact verification are distinct from merge, deployment and semantic verification;
- canonical GitHub `main` outranks operational source checkouts;
- archived pre-reset history is preservation only and cannot re-authorise old behaviour;
- private relay history is intentionally retained until a real retention policy exists;
- generic “DEV checkpoint” memory does not invent a conventional Workbench DEV deployment.

## Current inherited product discrepancies — recorded, not fixed

1. **P0 — Operations false-running projection** (`completed`/`failed` + active session can become `TaskRunning`).
2. Cross-plane authoritative job model gap.
3. Release-publication reliability/no-op retrigger defect.
4. Exact long-term private-relay retention/compaction policy unresolved.
5. Full post-validator unattended-continuation productive-completion proof missing.
6. Fresh Blender end-to-end GPU-render acceptance missing.
7. Unreal five-minute `zen` startup/root cause unresolved.
8. Cross-process relay-state locking requirement remains an open technical question; discovered local experiment was not accepted.
9. Knowledge-graph/searchable-decisions future shape remains undecided after closing old PR #110 unmerged.

## Repository cleanup result

Public source / registered checkout cleanup is **complete** to the reset's preservation rules:

- PR #110 closed unmerged;
- no open PRs at the post-#229 cleanup checkpoint;
- stale local experiment preserved as documentation then removed;
- stale local lineage preserved behind audit ref then operational checkout realigned;
- all local Workbench worktrees audited and reduced to one clean `main` worktree;
- 157 fully merged remote branch refs deleted after reachability proof;
- remaining branch histories preserved before deletion under one historical archive tag;
- archive tree verified equal to canonical checkpoint tree;
- active public branch surface reduced to `main` only at the cleanup checkpoint;
- private relay historical transport intentionally retained rather than destructively purged without a policy.

See `docs/REPOSITORY_CLEANUP_MANIFEST.md` and `docs/LOCAL_CHECKOUT_AUDIT_2026-08-21.md`.

## Completion gate status

- [x] Feature development frozen before reset changes.
- [x] Starting repository/release/runtime baseline recorded.
- [ ] All relevant historical project conversations inventoried in full — **BLOCKED by conversation access**.
- [x] Relevant repository documentation inventoried/reconciled.
- [x] Accessible Workbench durable project/global memory and current context audited.
- [x] Important accessible decisions extracted/classified.
- [x] Accessible conflicts reconciled.
- [x] Current requirements/authority hierarchy documented.
- [x] Significant superseded/rejected behaviours documented.
- [x] Do-not-reintroduce rules documented.
- [x] Architecture/security/operations/harness/knowledge documentation reconciled.
- [x] Current implementation/release/runtime state documented with evidence levels.
- [x] Repository compared against highest-risk current requirements.
- [x] Stale/obsolete repository material preservation-audited and cleaned.
- [x] Registered checkout/worktree cleanup completed and verified clean.
- [x] Remote branch cleanup completed with preservation-first archive evidence.
- [x] Governance/cleanup PR gates passed before each merge.
- [x] Remaining inherited product failures/discrepancies recorded rather than silently fixed.
- [x] Permanent governance rules added.
- [ ] Conversation pruning safety fully checked — **blocked by incomplete full conversation inventory**.
- [x] Conversation pruning manifest exists and has been updated with recovered rules.
- [ ] Proven that no important information exists solely in an unseen conversation scheduled for deletion — **cannot prove with current access**.
- [x] Fresh repository-based post-reset handoff exists and is updated.
- [ ] Repository proven to contain every recoverable historical decision from every Workbench conversation — **cannot prove for unseen conversations**.
- [ ] Future ordinary development authorised — **NO while status remains INCOMPLETE**.

## Exact remaining blocker

The repository itself is no longer the blocker. The remaining human/access boundary is historical conversation coverage/pruning.

To close the reset, one of these must occur:

1. provide/enable a way to enumerate/read the remaining historical Workbench project conversations so each can pass the pruning test; or
2. explicitly accept the residual risk that inaccessible historical conversations may contain uncaptured information and authorise closing the reset without proving them individually.

Until then, do not change this status to COMPLETE and do not resume ordinary feature development.
