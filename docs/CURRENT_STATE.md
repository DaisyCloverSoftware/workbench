# Workbench current state

Last cutover verification: **2026-08-21 (BST)**.

This document records what currently exists and how strongly it has been verified. Product intent belongs in canonical contracts and `docs/DECISIONS.md`.

## Governance and cutover state

The 2026-08-21 governance reset is **COMPLETE**. The repository is the durable project authority; historical ChatGPT conversations, memory capsules, PR descriptions and archived source are evidence only.

The legacy ChatGPT Project is now in **pre-cutover retirement audit**. Ordinary feature development is paused until the migration manifest and New Project handoff are complete. This pause does not reopen historical conversations as authority.

Reset evidence lives in:

- `docs/GOVERNANCE_RESET_2026-08-21.md`
- `docs/REPOSITORY_CLEANUP_MANIFEST.md`
- `docs/CONVERSATION_PRUNING_MANIFEST.md`
- `docs/LOCAL_CHECKOUT_AUDIT_2026-08-21.md`

## Current source and release baseline

- repository: `DaisyCloverSoftware/workbench`
- continuation branch: `main`
- 0.9.55 release merge: `5d08829a1924d6445d3578de9821bd3cae4dd823` (PR #233)
- P0 Operations correction merge: `cf97e8c1dab987782d91743df491e54f99a85103` (PR #232)
- validated correction head: `3d07e0336c7e79eba552e17c443638e7adb188b8`
- stable product version in source: `0.9.55`
- stable tag/release: `v0.9.55`

PR #232 corrected the 0.9.54 false-running projection and added semantic regressions, including the observed shape of 100 terminal history events plus one genuine running event. PR #233 coordinated the 0.9.55 release surfaces.

## Repository hygiene

The governance reset performed preservation-first cleanup of old branches/worktrees and closed stale PR #110 unmerged. At the final reset cleanup checkpoint, active public branches were reduced to `main` only and the registered checkout was clean/aligned.

The 0.9.55 correction/release flow subsequently created three understood remote branches in addition to `main`:

- `fix/operations-terminal-session-separation`
- `governance/close-reset-20260821`
- `release-request/v0.9.55`

They are release/audit residue, not competing continuation points. Do not delete merely for tidiness; remove only when separately proven safe.

A fresh cutover read of the registered `runner://workbench` checkout returned an empty `git status --short`, so no current uncommitted Workbench source is known there. Canonical GitHub source remains authoritative even if an operational checkout later becomes stale.

There are no open Workbench PRs or open Workbench issues at the cutover audit checkpoint. Closed audit-probe issues created accidentally during connector verification are explicitly non-project noise and must not be treated as backlog.

## Deployment terminology

Workbench has no canonical website-style DEV deployment.

Use:

- **development source** — branches and `main`
- **PR/preview build** — CI artifact for a commit/PR; not DEV
- **release candidate/request** — coordinated version-bump branch/commit
- **stable release** — official version tag/release artifacts
- **cluster live** — installed runner/server/relay/MCP runtime
- **Windows live** — installed desktop application

## Current runtime verification

### Cluster live

The 0.9.55 maintenance update reached terminal `succeeded`, and the private Workbench capabilities manifest now advertises `workbench_version: 0.9.55`. Cluster deployment of the 0.9.55 Workbench control/runtime surface is therefore **deployed and runtime-version verified**.

### Windows live

The last directly observed installed Windows desktop before the 0.9.55 correction was Workbench 0.9.54. The Windows host remains reachable through the outbound typed bridge, but this legacy Project has **not established a fresh installed-desktop 0.9.55 semantic inspection**.

Therefore:

- 0.9.55 Windows source/build/release: **implemented/tested/merged/released**
- installed Windows 0.9.55: **not yet verified by this cutover audit**
- corrected Operations semantics on the actual installed Windows UI: **awaiting user-visible inspection/sign-off**

This is the first item for the new Project's initial Correction Sprint; it is an acceptance/deployment closure item, not permission to redesign the dashboard.

## Operations/scheduler state

### Scheduler-native durable tasks

Implemented/tested/merged/released:

- durable queued state
- scheduler-owned queued → routing dispatch
- server/CI/Windows/AI execution capacity plus waiting/needs-you lanes
- persisted Critical → High → Normal → Low priority then FIFO
- historical zero-value priority = Normal
- measured, stage-based and indeterminate progress only
- `WorkItem` projection with lane/priority/progress/dependency/executor metadata

### Operations false-running correction

The 0.9.54 defect is now **historical implementation evidence**, not current source behaviour.

0.9.55 source preserves terminal `completed`/`failed` relay event state even when the surrounding project/session presence lease remains active; session presence is represented separately and terminal history no longer enters live Running lanes merely because the session is active.

Automated semantic coverage exists. Installed-Windows semantic verification/sign-off remains outstanding as described above.

## Private relay and continuation

- bounded Dashboard-facing relay projection prevents old history from defining current live Operations state
- underlying private relay history remains intentionally retained pending a retention/compaction policy
- authenticated durable continuation is implemented/tested
- automatic dependency wake has previous live evidence
- full clean post-validator proof of `waiting_dependency → automatic resume → useful work → completed` remains **UNVERIFIED**

## Windows bridge, Blender and Unreal

- outbound typed/allowlisted bridge only; no generic Windows shell
- Blender source explicitly configures Cycles GPU/OptiX for factory-startup headless rendering; fresh end-to-end GPU render acceptance remains **UNVERIFIED**
- old Unreal 5.8.1 `TNotNull`/stack-overflow startup crash is historical and removed
- current Unreal bounded smoke uses five minutes; latest inherited evidence reached timeout classified `zen`; root cause/current acceptance remains **UNRESOLVED / UNVERIFIED**

## Durable state / data model

Workbench Core currently persists private local JSON state (`State.Version` 3), not a relational database. State evolution is decode/normalisation/runtime-repair based and protected by compatibility tests. Secret values remain behind the encrypted local vault / protected configuration boundaries described in `SECURITY.md`.

## Remaining corrections and unfinished work

### Known corrections required before new feature development

1. **P0 acceptance closure — Windows Operations 0.9.55.** Install/update or otherwise establish the actual Windows live 0.9.55 surface, inspect the real Operations dashboard, and prove terminal history/session presence does not inflate live job counts. Record observation-driven corrections if any; obtain user sign-off before advancing.
2. **P1 release publication reliability.** Remove dependence on identical-tree/no-op `main` retriggers where the release workflow fails to publish promptly.
3. **P1 unattended continuation acceptance.** Produce a clean end-to-end live proof of wait → automatic wake → useful resumed work → completed.
4. **P1 private relay retention governance.** Define safe retention/compaction rules separately from the bounded live projection.
5. **P2 Blender acceptance.** Fresh end-to-end typed headless GPU render verification.
6. **P2 Unreal acceptance/investigation.** Fresh five-minute startup result and investigation of inherited `zen` classification.

### Approved architecture/roadmap work not yet a correction

- authoritative cross-plane job model spanning scheduler-native work, CI, direct server controls, typed Windows jobs and AI workers

### Ideas / decisions still requiring fresh approval/design

- searchable decisions / knowledge graph capability (old PR #110 implementation is rejected/stale)
- exact four-hour session-presence lease value
- richer selected-job drawer/reorder/control details beyond current contracts
- whether cross-process relay-state locking is required under the supported topology

## Development workflow after cutover

The new Project must begin with the **initial Correction Sprint**, not a new feature sprint. Each sprint must define observable acceptance, pass technical gates, produce an inspectable target, take observation-driven corrections, and receive user sign-off before the next sprint.

Non-human waits are not stopping points. Historical chats are not required bootstrap material.
