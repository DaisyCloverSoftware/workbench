# Workbench operations dashboard contract

Workbench's Dashboard is an operational control surface, not an activity feed.

## Truthfulness rules

- Never display a queue position unless Workbench owns a scheduler for that lane.
- Never display a percentage unless the underlying work item reports measured progress.
- Stage-based progress must name the current phase and total known phases.
- Indeterminate work reports elapsed time/current phase instead of a fabricated percentage.
- Every active work item identifies its execution lane and, when known, executor/machine/provider.
- Waiting work identifies its dependency/retry reason.
- Human attention is a separate lane and must remain exceptional.

## Lanes

- server_ops
- ci_builds
- windows_workstation
- ai_workers
- waiting
- needs_you

## Priority

Critical > High > Normal > Low. Priority is persisted on the work item. Within a lane and equal priority, enqueue order is FIFO.

## Initial scheduler boundary

The durable autonomous Task engine is the first scheduler-owned execution plane. It must stop treating queued as a transient label: Delegate/attention-resume enqueue durable tasks, while scheduler dispatch owns transition into routing/running. Direct machine controls remain a separate control plane and will be folded into the unified work-item inventory without making dashboard/status reads wait behind long execution.
