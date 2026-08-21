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

An individual job/task/operation may be:

- queued;
- routing;
- running;
- waiting on a dependency;
- waiting to retry;
- needs human attention;
- terminal (completed/failed/cancelled).

Only real non-terminal job state may contribute to live execution counts/lanes.

### 2. Project / ChatGPT session presence

A project or ChatGPT work session may have a bounded recent-activity lease. Presence answers **whether work on the project/session appears active or recent**. It does not prove that each recent bounded operation is still executing.

Presence may be shown in a dedicated presence/session surface or as secondary metadata, but it MUST NOT inflate `Running`, `Queued`, `Waiting` or `Needs You` job counts.

### 3. Recent operation history

Completed/failed/cancelled safe-hands operations may remain useful as recent history. Terminal history MUST NOT be projected back into live execution merely because its surrounding project/session remains active.

## Known 0.9.54 discrepancy

Workbench 0.9.54 currently violates the contract above: desktop projection maps some remote relay events whose individual state is `completed` or `failed` to `TaskRunning` when the runner's bounded `ActiveKnown && Active` session flag is true.

That behaviour is **REJECTED** and is an inherited implementation defect. The observed `Running 100` result is not accepted product semantics.

A future code correction must be made only after this canonical contract is in place and must include semantic regression tests that prove terminal operation history/session presence cannot become running jobs.

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

These names are CURRENT requirements as of the 2026-08-21 governance reset.

## Priority and ordering

Priority is:

Critical → High → Normal → Low.

Priority is persisted on scheduler-native work. Historical zero-value priority means Normal. Within one schedulable lane and equal priority, enqueue order is FIFO (with stable ID ordering only as a deterministic tie-breaker).

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

Where authoritative/available, a job may expose:

- project;
- job/title and type;
- state;
- priority;
- queue position (scheduler-owned only);
- lane;
- executor;
- machine;
- provider/tool;
- truthful progress and phase;
- dependency/retry reason;
- creation/start/update/elapsed timestamps;
- commit/revision;
- attempts;
- privacy-safe log/status summary;
- artifacts/results;
- safe controls such as prioritise/cancel/retry/requeue when the underlying execution source supports them.

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

A green build/UI responsiveness screenshot is supporting evidence only. It is not proof that these semantics are correct.
