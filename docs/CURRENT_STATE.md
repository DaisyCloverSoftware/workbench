# Workbench current state

Last governance verification: **2026-08-21 (BST)**.

This document records what currently exists and how strongly it has been verified. It is not a roadmap. Product intent belongs in the canonical contracts and `docs/DECISIONS.md`.

## Repository baseline

- Repository: `DaisyCloverSoftware/workbench`
- Default branch: `main`
- Freeze-point `main` HEAD: `235305bccbef9a35d38445946c4bdab63364f859`
- HEAD subject: `release: retrigger Workbench 0.9.54 publication`
- Preceding substantive merge: `0afabdb075818beb0e97d1941ef02e16c14fe795` (PR #222, Workbench 0.9.54 release preparation)
- Validated release-PR head: `b97566b5a773fcb5d0a88066df4633d0c03ba2e3`
- The no-op HEAD is one commit ahead of the substantive merge with **no changed files**, so the source tree is identical.
- `v0.9.54` resolves to source whose canonical version is `0.9.54`.
- PR #222's required `build`, `runner` and `ui-responsiveness` workflows passed on its exact validated head.
- Open pull requests at the audit baseline: PR #110 only (`Add searchable decisions and a project knowledge graph`). It is based on Workbench 0.9.10 and is not approved for merge by this baseline.
- Open GitHub issues at the audit baseline: none.
- Remote branches observed: 288. This includes old release-request, proof/test, fix/feature and diagnostic branches. Branch age alone is not deletion authority.

## Stable/live surfaces

Workbench does not currently have a canonical website-style DEV deployment. See `docs/DECISIONS.md` for terminology.

### Stable release

- Current verified version tag: `v0.9.54`.
- The release workflow builds `Workbench.exe`, `Workbench-Updater.exe` and Linux runner/server/relay assets on pushes to `main` when that version's GitHub release does not already exist.
- The freeze-point handoff records the official `v0.9.54` release as published. The connected repository interface used for this reset can resolve the tag/source but does not expose an independent release-object listing in the audit path used here; the publication claim therefore retains its release-history/freeze-evidence classification rather than being upgraded by assumption.

### Cluster live

Privacy-safe live checks during this reset verified:

- the private Workbench capability manifest advertises Workbench `0.9.54`;
- Workbench MCP and relay services are active;
- loopback MCP health is good;
- the bounded Workbench health script reports `overall=ok`;
- registered cluster nodes were Ready and storage health reported no attached unhealthy volumes at the check time.

Private hostnames, addresses, paths and topology are intentionally omitted from this public record.

### Windows live

Freeze-point user-visible evidence showed the Windows desktop running **Workbench 0.9.54**.

A fresh bridge check during the reset verified an online outbound Windows host and detected:

- Blender 5.1.2;
- Unreal Engine 5.8.1.

That bridge check does not itself report the Workbench desktop application's installed version, so the desktop-version statement remains tied to the freeze-point observation rather than being silently reclassified as freshly machine-probed.

## Current Operations/scheduler implementation

### Scheduler-native durable tasks — implemented, tested, merged and released

Current source includes:

- durable `queued` task state;
- scheduler-owned queued → routing dispatch;
- lane capacity for server operations, CI/builds, Windows workstation and AI workers;
- waiting and needs-you lanes with no execution capacity;
- persisted priority;
- Critical → High → Normal → Low ordering, then FIFO;
- historical persisted zero-value priority interpreted as Normal;
- queued-task reprioritisation;
- measured, stage-based and indeterminate progress metadata;
- fallback to indeterminate when measured/stage totals are invalid;
- `WorkItem` projection with lane, priority, queue position, executor/machine/provider, progress, dependency and timestamps.

The scheduler and dashboard do **not** yet have one authoritative native job model spanning every remote direct-control operation. Local durable tasks are scheduler-native; remote relay work is partly projected from bounded transport/activity metadata.

### Operations semantic acceptance — FAILED / inherited defect

Workbench 0.9.54 sees real remote activity but maps the runner's session/activity lease incorrectly.

Current desktop source can convert a relay event whose individual state is `completed` or `failed` into `TaskRunning` when `ActiveKnown && Active` is true. The corresponding regression test explicitly expects that mapping.

This made the live Dashboard report misleading counts such as `Running 100` when many entries were recent completed safe-hands operations associated with an active project/ChatGPT session.

Canonical requirement: execution state, session/presence and recent completed history are separate concepts. See `docs/operations-dashboard-contract.md` and WB-DEC-003.

Status of the 0.9.54 Operations semantics: **released and live, but not accepted/verified correct**.

## Private relay and unattended continuation

### Bounded activity inventory — implemented, tested, merged and released

The runner no longer archives the entire append-oriented relay history on each Dashboard refresh. The bounded live view keeps every pending request, recent activity and matching request/result counterparts while excluding old completed history from the live projection.

This solves the historical scaling failure in which the live inventory could become unusable as relay history grew. It does **not** define a long-term retention/cleanup policy for the underlying private transport repository.

### Authenticated durable continuation — implemented, tested, merged and partially live-verified

Current source HMAC-seals private-relay continuation authority and validates the relay correlation/project/continuation body. A Workbench-owned dependency-result suffix may follow the proof after a durable dependency becomes terminal; arbitrary appended content remains fail-closed.

Evidence status:

- implementation: yes;
- regression tests: yes;
- merged/released: yes;
- live dependency watcher automatically waking a waiting task without a new chat message: previously observed;
- clean post-validator-fix proof of the full sequence `waiting_dependency → automatic resume → useful work → completed`: **not established by the accessible evidence in this reset**.

Treat full unattended productive completion as **UNVERIFIED** until a clean proof is recorded.

### Private capability manifest — reconciled during reset

At the freeze point, the private 0.9.54 machine-readable manifest understated later typed Windows capabilities. During this governance reset its transport metadata was reconciled with current Workbench source: outbound host discovery, Blender version, bounded Unreal startup smoke and host-job status are advertised, while the reviewed committed Blender render wrapper remains an exact typed operations-script path rather than a generic Windows command route.

No runtime authority was widened. The manifest still explicitly preserves the outbound-only/allowlisted/no-generic-Windows-shell boundary.

## Windows bridge, Blender and Unreal

### Security/control model — current

The Windows bridge is outbound and typed/allowlisted. Generic remote Windows shell authority is not part of the intended interface.

### Blender

Current release history/source establishes explicit Cycles GPU setup including OptiX/non-CPU-device enablement for headless rendering. The reset verified the Windows bridge detects Blender 5.1.2.

A fresh smoke was not completed during this reset: the committed audit wrapper correctly requires an explicit host argument and the no-argument governance probe failed at usage validation before Blender ran. Therefore end-to-end current GPU-render acceptance remains **UNVERIFIED** by this reset.

### Unreal

The old Unreal 5.8.1 `TNotNull`/stack-overflow startup crash is historical, not the current known failure. The current typed smoke uses a bounded five-minute startup window with privacy-safe classifications.

Latest inherited live evidence before the reset: Unreal stayed alive beyond the old ~90-second boundary and reached the five-minute limit classified `zen`.

The reset verified the Windows bridge detects Unreal Engine 5.8.1. A fresh smoke was not completed because the committed wrapper requires an explicit host argument and the no-argument governance probe stopped at usage validation before Unreal ran.

Current Unreal startup acceptance therefore remains **UNRESOLVED / UNVERIFIED** beyond the inherited `class=zen` evidence.

## Durable state/storage

Workbench's desktop/core durable state is JSON stored in the local Workbench configuration area. Current `State.Version` default is **3**.

The store:

- decodes on-disk state before applying runtime/default repairs;
- normalises the project registry during load/save;
- writes via a private temporary file and atomic rename;
- persists tasks, project registry, preferences and encrypted secret references;
- persists scheduler priority/progress as part of task state.

This is a file-format/state-version model rather than a relational database/schema migration system.

## Release process

The release-request workflow prepares coordinated version bumps from `.workbench-release.json`, runs `git diff --check` and `go test ./...`, commits the prepared release branch, and relies on the normal PR gates before merge.

The release workflow is push-to-`main` and idempotently skips work if the release already exists. A recurring failure mode has required an identical-tree no-op main commit to retrigger publication. The current `main` HEAD is such a retrigger.

Status: **known release-process defect with a temporary workaround**. Do not treat the no-op step as the desired permanent release protocol.

## Known governance/documentation discrepancies at freeze

The reset found and is correcting these repository-authority problems:

- `ARCHITECTURE.md` described an older command-template/delegation-centric model and omitted the current scheduler/continuation/Windows-control boundaries;
- the Operations contract did not explicitly separate job execution from session presence/history;
- `ROADMAP.md` presented searchable decisions/knowledge graph as simply next despite stale PR #110;
- `SECURITY.md` did not yet explicitly capture the typed outbound Windows bridge and authenticated-private-continuation distinction;
- the private capability manifest lagged later typed Blender/Unreal operations; that transport-document drift was reconciled during the reset;
- green UI responsiveness did not prevent a semantically false `Running 100` dashboard, so semantic acceptance needs explicit governance.

## Audit blind spots / not verified

Do not infer these as true or false:

- complete inventory/content audit of every historical Workbench project conversation (conversation retrieval failed/unavailable in this reset);
- local developer/cluster source checkout working-tree and worktree state beyond what existing privacy-safe health checks expose;
- deletion/cleanup safety of all 288 remote branches;
- formal long-term private-relay retention policy;
- full post-validator unattended continuation completion proof;
- fresh end-to-end Blender GPU render acceptance;
- fresh Unreal five-minute startup outcome/root cause;
- any feature proposition in old PR #110 without a new decision.

These blind spots keep the governance reset status **INCOMPLETE** until the applicable completion gates can be satisfied or explicitly resolved by a future audit with the required access.
