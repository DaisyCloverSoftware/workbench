# Workbench Operations dashboard contract

Workbench's Dashboard is an **operational control surface**, not an activity counter.

The primary user questions are:

- what real work is happening or queued now?
- where will/is it executing?
- why has something not started or progressed?
- what actually requires human intervention?

## Normative semantic separation

The Dashboard MUST keep these concepts distinct:

### 1. Job execution state

An individual job/task/operation may be queued, routing, running, waiting on a dependency, waiting to retry, needs human attention, or terminal (completed/failed/cancelled).

Only real non-terminal job state may contribute to live execution counts/lanes.

### 2. Project / ChatGPT session presence

A project or ChatGPT work session may have a bounded recent-activity lease. Presence answers **whether work on the project/session appears active or recent**. It does not prove that each recent bounded operation is still executing.

Presence may be shown in a dedicated presence/session surface or as secondary metadata, but it MUST NOT inflate `Running`, `Queued`, `Waiting` or `Needs You` job counts.

### 3. Recent operation history

Completed/failed/cancelled safe-hands operations may remain useful as recent history. Terminal history MUST NOT be projected back into live execution merely because its surrounding project/session remains active.

## Historical 0.9.54 discrepancy and 0.9.55 correction

Workbench 0.9.54 violated the contract above by mapping some remote relay events whose individual state was `completed` or `failed` to `TaskRunning` when the runner's bounded session-presence flag was active. The observed `Running 100` result is permanent regression evidence, not desired behaviour.

Workbench 0.9.55 corrects the source projection:

- terminal `completed`/`failed` event state remains terminal even when project/session presence is active;
- real remote running/queued/routing/waiting/attention state is projected from the event state itself;
- session presence is retained as separately labelled context;
- regression coverage includes completed+active, failed+active, genuine remote state and the observed 100-terminal-history failure shape.

Status at the 2026-08-21 project cutover:

- source correction: **implemented/tested/merged/released in 0.9.55**;
- cluster Workbench runtime: **updated to 0.9.55**;
- actual installed Windows 0.9.55 Dashboard semantic inspection/user sign-off: **still required**.

The new Project's initial Correction Sprint must close that installed-Windows acceptance gap before new feature work begins. It must not redesign the contract merely because acceptance is still pending.

## Truthfulness rules

- Never display a queue position unless Workbench owns the scheduler/order for that lane/job source.
- Never display a percentage unless the underlying work item reports measured progress with a genuine total.
- Stage-based progress must name the current phase and use a genuine known stage count.
- Indeterminate work reports phase/elapsed information without a fabricated percentage.
- Every active job identifies its lane and, when known, executor/machine/provider.
- Waiting work identifies its dependency/retry reason.
- Human attention is a separate lane and must remain exceptional.
- Terminal history does not count as live work.
- Session/presence recency does not count as individual job execution.
- Unknown state must remain unknown/indeterminate rather than being guessed from elapsed time or transport recency.

## Lanes

The canonical execution/state lanes are:

- `server_ops`
- `ci_builds`
- `windows_workstation`
- `ai_workers`
- `waiting`
- `needs_you`

## Priority and ordering

Priority is Critical → High → Normal → Low.

Priority is persisted on scheduler-native work. Historical zero-value priority means Normal. Within one schedulable lane and equal priority, enqueue order is FIFO, with stable ID ordering only as a deterministic tie-breaker.

Do not imply a global cross-system queue position for work sources whose scheduler Workbench does not own.

## Progress

Allowed progress kinds:

- **measured** — real current/total units;
- **stages** — real current stage/known stage total;
- **indeterminate** — truthful phase/elapsed status when completion cannot be measured.

Elapsed time alone is never a completion percentage.

## Scheduler boundary

The durable Core Task engine is scheduler-native. Delegation/attention-resume enqueues durable work; scheduler dispatch owns queued → routing transition and per-lane execution capacity.

Direct server/CI/Windows controls and relay operations are separate execution/control sources. They may be projected into the same Operations UI only when their state is represented by an authoritative job record or a clearly labelled non-job presence/history projection.

Transport files and recent-activity leases are not a substitute for an authoritative job model.

## Control-plane responsiveness

Long-running jobs MUST NOT make unrelated status, health, queue or cancellation reads wait behind the execution simply because they share transport infrastructure.

The intended direction is a responsive control plane plus durable asynchronous execution plane.

## Work-item information

Where authoritative/available, a job may expose project, title/type, state, priority, scheduler-owned queue position, lane, executor, machine, provider/tool, truthful progress/phase, dependency/retry reason, timestamps, commit/revision, attempts, privacy-safe status/log summary, artifacts/results and safe controls supported by its execution source.

Fields that are not authoritative MUST be omitted/unknown rather than synthesized.

## Semantic acceptance gates

Operations changes are not accepted merely because the window renders or remains responsive.

At minimum, tests/verification for state-projection changes must cover:

- real running job → Running;
- queued scheduler-native job → Queued with truthful lane-local order;
- waiting dependency/retry → Waiting, not Running;
- human attention → Needs You;
- completed operation + active project/session lease → **not Running**;
- failed operation + active project/session lease → **not Running**;
- inactive completed history → not live work;
- real remote work remains visible even when relay history is large;
- no fake percentage/queue position when authority is absent.

A green build/UI responsiveness screenshot is supporting evidence only. Installed-target semantic inspection remains a separate acceptance gate where the change depends on live Windows/cluster data.
