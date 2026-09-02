# Workbench current state

Last verified correction checkpoint: **2026-09-02 (BST)**.

This document records what currently exists and how strongly it has been verified. Product intent belongs in canonical contracts and `docs/DECISIONS.md`; Sprint 0 correction status belongs in `docs/BASELINE_CORRECTION_REGISTER.md`.

## Governance and development state

The 2026-08-21 governance reset is **COMPLETE**. The repository is the durable project authority; historical ChatGPT conversations, memory capsules, PR descriptions and archived source are evidence only.

**Sprint 0 — Existing Corrections & Baseline Stabilisation remains ACTIVE.** New product sprint work is still gated on Sprint 0 owner sign-off. The mandatory continuity rules in `docs/GOVERNANCE.md` and `docs/SPRINT_GOVERNANCE.md` apply: ordinary CI, test, build, publication, deployment and tool latency are execution states rather than owner-return boundaries.

## Current source and release baseline

- repository: `DaisyCloverSoftware/workbench`
- continuation branch: `main`
- Development Continuity & Routing implementation: PR #279, merged as `a309bae3621e2c12691c08bee16e9729ff5464c4`
- Workbench 0.9.60 release coordination: PR #280, merged as `f4609cb115f5aac308267023f283ac08e9427dba`
- stable product version: `0.9.60`
- stable tag/release: `v0.9.60`
- `v0.9.60` release target: `f4609cb115f5aac308267023f283ac08e9427dba`

The 0.9.60 main push passed the permanent `runner`, `build`, `ui-responsiveness` and `release` workflows. The release workflow tested the source, built the release assets and published `v0.9.60` from the exact release merge.

PR #280 exact-head validation also passed the six correction-relevant PR workflows: `runner`, `build`, `ui-responsiveness`, `sprint1-live-operations-integration`, `sprint1-operations-acceptance` and `sprint1-operational-telemetry`.

## Development Continuity & Routing correction — S0-009

S0-009 is **TARGET VERIFIED / READY FOR OWNER RE-OBSERVATION**, not signed off.

The correction establishes and verifies the following current behaviour:

- ChatGPT remains the primary engineering brain and owns source changes, Git/GitHub, PRs, reviews, CI, releases and subsequent engineering decisions.
- The private Workbench control relay and reviewed `scripts/ops/*.sh` operations are the normal machine-execution paths.
- OpenClaw is not part of normal Development routing. It is denied unless the owner explicitly requests OpenClaw by name for the applicable operation.
- A direct-capability miss, CI/deployment failure, Kubernetes/Docker/systemd/Helm problem, prior OpenClaw use or provider availability does not create OpenClaw authority.
- `[workbench:operations]` is routing metadata only; it is not owner consent.
- Ordinary engineering waits do not require owner keepalive prompts such as `continue`, `carry on` or `check again`.
- In-scope automated failures are investigated, corrected and revalidated without fresh owner authorisation unless a genuine owner-only boundary is reached.
- Durable Workbench recovery preserves authorised task state across process interruption and refuses to resurrect legacy/unauthorised OpenClaw operations.
- A ChatGPT/frontend/tool transport interruption is not semantically equivalent to cancelling the authorised Development correction. After transport resumes, Development restores canonical repository/durable state and continues from the next safe action.

The telemetry failure encountered during the correction was root-caused in the live Windows telemetry demonstration rather than suppressed. The corrected demonstration passed on the final #279 exact head and again on the 0.9.60 release PR exact head, including regression checks, live operational telemetry and proof upload.

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

### Cluster live / private control plane

The private maintenance update for the 0.9.60 correction reached terminal `succeeded` after the official release was published.

The live private capability manifest subsequently advertised:

- `workbench_version: 0.9.60`
- `primary_brain: chatgpt`
- `preferred_write_transport: private-git-relay`
- `openclaw_policy: explicit_owner_request_only`
- direct control actions for bounded repository work, machine inspection/mutation, reviewed operations scripts and update/status operations

A post-update execution of the reviewed `scripts/ops/workbench-health.sh` operation at exact release commit `f4609cb115f5aac308267023f283ac08e9427dba` returned `overall=ok`, with the Workbench binaries present, MCP/relay services active, loopback MCP health successful and the relay checkout clean. This establishes the 0.9.60 private control plane as **deployed and runtime-version verified**.

### Windows live

The official `v0.9.60` release contains current Windows desktop and updater assets. This correction did not independently establish that the owner's installed Windows desktop is running the 0.9.60 asset or complete the separate Windows Operations owner-observation gate.

Therefore the earlier Windows-live Sprint 0 items remain governed by the Baseline Correction Register. Do not infer installed-Windows acceptance merely from release publication or cluster-live success.

## Operations/scheduler state

Implemented/tested/released behaviour includes:

- durable queued state;
- scheduler-owned queued → routing dispatch;
- server/CI/Windows/AI execution capacity plus waiting/needs-you lanes;
- persisted Critical → High → Normal → Low priority then FIFO;
- measured, stage-based and indeterminate progress only;
- `WorkItem` projection with lane/priority/progress/dependency/executor metadata;
- terminal relay history separated from current session presence;
- explicit durable OpenClaw owner-authorisation state for operations tasks;
- recovery that retires non-terminal legacy/unauthorised OpenClaw operations instead of implicitly resuming them.

The historical 0.9.54 false-running projection remains corrected in current source. Installed-Windows semantic observation/sign-off remains a separate Sprint 0 acceptance item.

## Private relay and continuation

Current private-relay behaviour includes:

- bounded Dashboard-facing relay projection;
- authenticated durable continuation;
- direct bounded repository and machine controls that do not require external model credit;
- explicit separation between normal `relay/control` execution and deliberate owner-authorised OpenClaw `relay/inbox` execution;
- automatic dependency wake support and durable interrupted-task recovery.

The clean end-to-end Sprint 0 proof of `waiting_dependency → automatic resume → useful work → completed` remains a distinct acceptance item until the evidence required by S0-003 is recorded. The Development continuity correction does not silently mark that broader acceptance item signed off.

Underlying private relay history remains intentionally retained pending the separate retention/compaction governance correction.

## Windows bridge, Blender and Unreal

- outbound typed/allowlisted Windows bridge only; no generic Windows shell;
- Blender source has the intended bounded GPU path, while fresh end-to-end GPU render acceptance remains a separate Sprint 0 item;
- the historical Unreal 5.8.1 `TNotNull`/stack-overflow crash is removed; the current bounded startup acceptance/investigation remains separately tracked.

## Durable state / data model

Workbench Core persists private local JSON state with compatibility/normalisation protections. Secret values remain behind the encrypted local vault / protected configuration boundaries described in `SECURITY.md`. Durable task state is control-plane state; provider transcripts and credentials are not treated as public project authority.

## Remaining Sprint 0 corrections

Use `docs/BASELINE_CORRECTION_REGISTER.md` as the authoritative register. At this checkpoint:

- **S0-009 — Development Continuity & Routing:** target verified and ready for owner re-observation.
- The installed-Windows Operations/identity acceptance items remain open until the actual Windows live target is observed and signed off.
- Release publication reliability has additional positive evidence from the normal 0.9.60 release, but is not automatically closed without its defined acceptance conclusion.
- The broader unattended dependency continuation acceptance remains open as S0-003.
- Private relay retention governance, Blender acceptance, Unreal acceptance/investigation and the durable self-update recurrence-prevention item remain governed by their register entries.

Do not begin the next product sprint merely because S0-009 is ready for re-observation.

## Development workflow

Non-human waits are not stopping points. A transient conversation/tool transport interruption does not cancel the authorised sprint or correction and does not create a new owner-authorisation requirement. Restore canonical repository and durable Workbench state, continue the same authorised scope, and return to the owner at the actual observation/sign-off boundary or at a genuine owner-only blocker.

Historical chats are not required bootstrap material and cannot override current canonical governance.
