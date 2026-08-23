# Sprint 1 operational telemetry correction

This branch extends the existing Operations surface with canonical live execution telemetry. It does not introduce a parallel dashboard state model.

- Measurable providers may emit validated current/total telemetry through the structured harness progress channel.
- Unmeasurable active work uses Workbench's real routing/execution/finalization stages without a fabricated percentage.
- Accepted telemetry updates the durable Task.Progress and Task.UpdatedAt fields and notifies existing Engine subscribers.
- The Windows Operations rows render progress, elapsed runtime, latest activity age, worker/priority context, and queue order directly without requiring selection.
- A UI clock updates only derived elapsed/activity ages; execution state changes remain Engine-event-driven.
