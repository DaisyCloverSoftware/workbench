# Workbench governance reset — 2026-08-21

Status: **INCOMPLETE**

Ordinary feature development is frozen while this reset is active. The reset remains incomplete because full historical conversation coverage and some local/private operational state cannot be verified with the access available in this audit. This is deliberate: missing evidence is not converted into success by assumption.

## Purpose

Reconstruct and canonicalise current Workbench intent/state so a future maintainer or AI can work from the repository without depending on old conversations. Preserve rejected/superseded decisions so they cannot silently return.

## Freeze-point baseline

- Freeze time: approximately 2026-08-21 03:09 BST.
- Repository: `DaisyCloverSoftware/workbench`.
- `main`: `235305bccbef9a35d38445946c4bdab63364f859`.
- The HEAD is a release-publication retrigger with an identical tree to preceding substantive merge `0afabdb075818beb0e97d1941ef02e16c14fe795` (PR #222).
- Release PR #222 validated head: `b97566b5a773fcb5d0a88066df4633d0c03ba2e3`.
- `build`, `runner` and `ui-responsiveness` workflows passed on that validated head.
- `v0.9.54` source resolves to canonical version `0.9.54`.
- Only open PR found: #110, based on Workbench 0.9.10; not approved/merged by this reset.
- No open GitHub issues found.
- 288 remote branches were enumerated.
- Cluster/private Workbench capability manifest advertises 0.9.54; privacy-safe health checks found the Workbench control plane healthy at the audit time.
- Fresh outbound Windows bridge inventory found a host online with Blender 5.1.2 and Unreal Engine 5.8.1.
- User-visible freeze evidence identified Windows desktop Workbench 0.9.54.
- No separately established conventional Workbench DEV deployment was found.

See `docs/CURRENT_STATE.md` for detailed evidence classification.

## Development freeze rule

During this reset:

- no dashboard bug fix;
- no scheduler redesign;
- no Unreal/Blender debugging;
- no release-process fix;
- no knowledge-graph implementation;
- no ordinary feature/refactor work.

Permitted changes are governance, documentation, audit evidence, safe cleanup and reset-validation tooling.

## Audit coverage record

| Source/category | Coverage | Result / blind spot |
| --- | --- | --- |
| Repository root/current `main` | INSPECTED | Exact HEAD/default branch/version surfaces/root inventory verified. |
| Root product docs | INSPECTED | `README.md`, `VISION.md`, `ARCHITECTURE.md`, `ROADMAP.md`, `SECURITY.md`, `PUBLIC_SOURCE_POLICY.md`, `CONTRIBUTING.md`, `CHANGELOG.md` inventory; current-sensitive docs compared/reconciled. |
| `docs/` inventory | INVENTORIED | Current docs list captured. High-impact governance/architecture/knowledge/privacy/UI/Operations/bootstrap materials inspected. Not every historical prose line was treated as current authority merely because it exists. |
| Current Core scheduler/work-item source | INSPECTED | `model.go`, `work_item.go`, `scheduler.go`, scheduler regression tests; priority/progress/lane/storage semantics verified. |
| Current Dashboard Operations projection | INSPECTED | 0.9.54 completed/failed + session-active → Running mapping confirmed as implementation discrepancy. |
| Durable state store | INSPECTED | JSON state, State.Version 3, private temp + atomic rename, project normalisation verified. |
| Private continuation source/history | INSPECTED | Current HMAC validator plus PR #211 change/test rationale verified. Full post-fix live productive completion remains unverified. |
| Recent release/feature PRs | INSPECTED | #203–#222 audited at decision/change level; superseded proof/release PRs distinguished from current behaviour. |
| Older relevant control-boundary PRs | SAMPLED/INSPECTED | Direct ChatGPT/OpenClaw boundary, Windows typed operations and earlier release architecture checked where relevant. |
| PR #110 | INSPECTED | Old 0.9.10-based implementation; kept open but classified as stale implementation basis requiring fresh decision. |
| Open issues | INSPECTED | None at baseline. |
| Workflow configuration | INSPECTED | Current build/release-request/release/runner/UI workflow inventory; release publication trigger/workaround mechanism verified. |
| Remote branches | INVENTORIED | 288 branches. Diagnostic and active-session branch disposition specifically checked; full per-branch semantic audit not completed. |
| Diagnostic dashboard branch | INSPECTED | One 12-line live `chat_activity` probe; useful evidence preserved in current-state record; branch is cleanup candidate. |
| Private relay guide/capability contract | INSPECTED + RECONCILED | 0.9.54 manifest/guide checked against current source. Stale Windows capability wording was corrected during the reset without widening runtime authority. |
| Private relay historical transport | STRUCTURE/BEHAVIOUR INSPECTED | Bounded live-selection implementation reviewed. Payload-by-payload historical audit intentionally not performed in public reset; long-term retention policy unresolved. |
| Live cluster health | VERIFIED (bounded/privacy-safe) | Workbench services/MCP health good; registered nodes Ready; no private topology copied publicly. |
| Live Windows bridge | VERIFIED (bounded/privacy-safe) | Online host; Blender/Unreal detection verified. Fresh application smokes did not run because audit wrapper calls lacked required explicit host argument. |
| File Library Workbench artifacts | INVENTORIED/SAMPLED | Older 0.6.x/0.7.x release/setup artifacts found; classified historical because they predate current 0.9.54 architecture. |
| Workbench durable memory/context mechanism | INSPECTED IN SOURCE/DOCS | Four-layer knowledge/context mechanism known. Actual stored user/project capsule contents were not available through the audit interface, so unique-memory decision coverage is unverified. |
| Historical project conversations | PARTIAL / BLOCKED | Current handoff/current-project context accessible, but complete conversation enumeration/retrieval failed/unavailable. This blocks blanket conversation-deletion approval. |
| Local developer/cluster checkout `git status` and worktrees | BLOCKED | Current private control interface does not expose generic repo-status/branch commands; the reset did not widen authority. Existing health proved relay checkout health, not every developer worktree. |

## Reconciled current decisions

Canonical current/superseded/rejected decisions are in `docs/DECISIONS.md`.

The most important reset outcomes are:

- repository documentation, not conversations, is project authority;
- ChatGPT is primary brain/coder, Workbench is durable/safe execution infrastructure, autonomous harnesses are optional capacity;
- Operations job execution is distinct from session presence and terminal history;
- six Operations lanes remain current;
- priority is Critical → High → Normal → Low then FIFO, with zero-value persisted priority = Normal;
- progress is measured/staged/indeterminate only;
- direct machine controls and scheduler-native tasks are separate execution planes until a true authoritative unified job model exists;
- direct ChatGPT implicit OpenClaw delegation remains blocked;
- authenticated private continuation is a separate trusted path;
- outbound typed Windows operations remain the required security boundary;
- release no-op retrigger pushes are a temporary workaround/defect, not a desired release contract;
- conventional website-style DEV terminology is not applicable unless explicitly created later.

## Current inherited implementation discrepancies

These are recorded, not fixed during the governance freeze:

1. **P0 — Operations false-running projection.** 0.9.54 maps session-active completed/failed relay actions to `TaskRunning` and can report misleading counts such as `Running 100`.
2. **Cross-plane job model gap.** Scheduler-native tasks and relay/direct-control activity do not yet share one authoritative job-state source.
3. **Release publication reliability.** Version-bump merges have sometimes needed an identical-tree push to retrigger release publication.
4. **Private relay retention.** Live reads are bounded, but historical transport retention/cleanup is not canonically governed.
5. **Unattended continuation acceptance.** Automatic dependency wake-up was observed, but full post-validator productive completion is not cleanly evidenced.
6. **Blender live acceptance.** Explicit GPU/OptiX implementation exists; reset did not establish a fresh end-to-end GPU render proof.
7. **Unreal live acceptance.** Old `TNotNull` crash is historical; inherited current evidence is five-minute `class=zen`, root cause unresolved.

## Resolved governance/documentation discrepancies

- The private machine-readable Windows capability description lagged later 0.9.54 typed operations. It was reconciled during the reset to match current source while preserving the outbound-only/allowlisted/no-generic-shell security boundary.
- The first governance edit of `docs/PERSONAL_PRO_RELAY.md` over-compressed valid envelope/install/runbook detail. The diff audit caught this before merge and the operational detail was restored under the new governance/source-of-truth rules rather than silently discarded.

## Repository cleanup review

Reviewed cleanup candidates include:

- stale release-request branches;
- proof/test branches;
- `diag/dashboard-activity-live-20260821`;
- merged `fix/operations-active-session-state-20260821` branch (contains no unique commits relative to main at audit time);
- old PR #110 branch;
- no-op/proof branches;
- private relay historical transport growth.

No broad destructive branch deletion is performed by this audit record alone. With 288 branches and incomplete historical-conversation/memory coverage, blanket deletion would fail the reset's preservation rule. `docs/REPOSITORY_CLEANUP_MANIFEST.md` records disposition rather than pretending cleanup is complete.

## Validation evidence

The governance branch is documentation/governance only. PR #223 head `534990eaa9279babbb8f95482093e59bdf70f1d3` passed all three required pull-request workflows:

- `build` — success;
- `runner` — success;
- `ui-responsiveness` — success.

That proves repository/build/UI-responsiveness consistency for the validated governance tree. It does **not** prove the known 0.9.54 Dashboard semantics correct. The evidence-recording commit that follows this validated head must itself pass the same exact-head gates before merge; no further product or governance-content changes are permitted after that final validation.

After merge, canonical docs remain authoritative even while product discrepancies await post-reset implementation.

## Completion gate status

- [x] Feature development frozen before reset changes.
- [x] Starting repository/release/runtime baseline recorded to available evidence level.
- [ ] All relevant historical project conversations inventoried — **BLOCKED by conversation retrieval/access**.
- [x] Relevant repository documentation inventoried.
- [ ] Workbench durable project-memory contents fully audited — **mechanism inspected; stored content unavailable through current audit interface**.
- [x] Important accessible decisions extracted and classified.
- [x] Accessible conflicting decisions reconciled.
- [x] Current requirements/authority hierarchy documented.
- [x] Significant superseded/rejected behaviours documented.
- [x] Do-not-reintroduce rules documented.
- [x] Architecture documentation reconciled.
- [x] Current implementation/release/runtime state documented with evidence levels.
- [x] Repository compared against the highest-risk current requirements.
- [x] Stale/obsolete repository material reviewed at manifest level.
- [ ] Repository cleanup completed — **blocked pending full preservation/branch safety audit; manifest exists**.
- [x] Governance branch tests/builds/checks passed on validated PR #223 head `534990eaa9279babbb8f95482093e59bdf70f1d3`; final evidence-only head still requires exact-head confirmation before merge.
- [x] Remaining inherited failures/discrepancies recorded.
- [x] Permanent governance rules added on governance branch.
- [ ] Conversation pruning safety fully checked — **blocked by incomplete conversation inventory**.
- [x] Conversation pruning manifest created.
- [ ] Proven that no important information exists solely in a conversation scheduled for deletion — **cannot yet prove**.
- [x] Fresh post-reset development handoff created, explicitly conditional on the reset gate.
- [ ] Repository is fully proven to be the sole durable project truth — **target state documented, historical blind spots remain**.
- [ ] Future ordinary development authorised — **NO while this reset status remains INCOMPLETE**.

## Rule for closing this reset

Do not flip this document to COMPLETE until the unchecked gates are resolved with evidence. If an inaccessible historical source can never be recovered, resolve that explicitly as a conscious governance decision/risk acceptance rather than silently treating it as audited.
