# Workbench decision register

This is the durable decision and supersession register for material Workbench product behaviour. Status values are **CURRENT**, **SUPERSEDED**, **REJECTED**, **HISTORICAL**, **IMPLEMENTATION DETAIL** and **AMBIGUOUS**.

## Current decisions

### WB-DEC-001 — ChatGPT-first responsibility split — CURRENT

Workbench is a ChatGPT-first developer/operations workspace and execution bridge.

- ChatGPT owns primary reasoning, source-code decisions, Git/GitHub work, PR/CI/release orchestration and normal bounded machine operations.
- Workbench owns durable task state, safe repository eyes/hands, scheduling/execution infrastructure, machine operations, the outbound Windows bridge, private relay transport and continuity across chat boundaries.
- OpenClaw and other autonomous harnesses are optional execution/operator capacity, not the default coder and not required for routine bounded operations.
- The human is interrupted only for a genuine human-only decision, permission or authority boundary.

This aligns `README.md`, `VISION.md` and `docs/CHATGPT_BOOTSTRAP.md` and supersedes older architecture wording that made command-template delegation look central.

### WB-DEC-002 — Repository is the durable project authority — CURRENT

Canonical repository requirements/decisions govern Workbench. Conversations, PR descriptions, memory capsules and implementation history are evidence, not authority. See `docs/GOVERNANCE.md`.

### WB-DEC-003 — Operations is an execution dashboard, not an activity counter — CURRENT

The Operations dashboard MUST answer what real work is executing/queued/waiting, where it is happening, why it has not started, and what genuinely requires the human.

Three concepts MUST remain distinct:

1. **job execution state** — an individual operation/task is actually queued, routing, running, waiting or needs attention;
2. **project/session presence** — ChatGPT or a project may be considered recently active even when its latest bounded operation has completed;
3. **recent operation history** — completed/failed operations retained for observability/audit.

Session/presence state MUST NOT be promoted into `running` for each completed operation. Recent history MUST NOT inflate live job counts.

The 0.9.54 implementation violates this decision by mapping some `completed`/`failed` relay activity with `Active=true` to `TaskRunning`. That is an inherited implementation defect, not a requirement.

### WB-DEC-004 — Six execution lanes, truthful priority and progress — CURRENT

The canonical lanes are:

- `server_ops`
- `ci_builds`
- `windows_workstation`
- `ai_workers`
- `waiting`
- `needs_you`

Priority order is Critical → High → Normal → Low, then FIFO for equal priority within a lane. Persisted zero-value priority remains Normal for historical compatibility.

Progress may be measured, stage-based or indeterminate. Fake elapsed-time percentages are prohibited.

### WB-DEC-005 — Scheduler ownership and control-plane separation — CURRENT

Durable local tasks remain queued until the Workbench scheduler owns the queued → routing transition. Scheduler execution capacity is lane-aware. Completion/cancellation/wakeup may cause further dispatch. Queued tasks may be reprioritised.

Direct bounded machine controls are a distinct control plane. Long-running execution MUST NOT make ordinary status/control reads wait behind it. A future unified inventory may project both planes, but it must preserve truthful execution semantics rather than treating transport/activity history as native jobs.

### WB-DEC-006 — Authenticated private continuation is distinct from implicit delegation — CURRENT

Direct ChatGPT development operations MUST NOT silently become OpenClaw/autonomous coding delegation.

The authenticated private relay may carry an explicit durable continuation handoff. Continuation authority is HMAC-bound to the relay correlation, project and original continuation body; Workbench may append its exact dependency-result suffix when a durable wait becomes terminal. Arbitrary appended content remains fail-closed.

Automatic dependency wake-up has been observed live. A clean post-validator-fix proof of `waiting_dependency → automatic resume → useful work → completed` is not yet canonicalised as verified live acceptance.

### WB-DEC-007 — Windows bridge remains outbound and typed — CURRENT

Windows hosts connect outbound. Workbench exposes exact typed/allowlisted operations rather than a generic remote Windows shell. Convenience is not sufficient reason to add generic command authority.

Typed Blender and Unreal operations may exist inside this boundary. Their live product acceptance is independent from the security decision.

### WB-DEC-008 — Release publication no-op pushes are a temporary workaround — CURRENT

Workbench coordinates version surfaces and builds the Windows app/updater plus Linux cluster assets. A stable release is not complete merely because its release PR is green/merged.

The current `main`-push release workflow has occasionally required an identical-tree no-op push to retrigger publication after a version-bump merge. Such commits are a known temporary workaround/release-process defect and MUST NOT be normalised as the intended release protocol.

### WB-DEC-009 — Deployment terminology is explicit — CURRENT

Workbench has no canonical conventional website-style DEV deployment at this baseline.

Use these terms:

- **development source** — normal feature/governance branches and `main` source state;
- **PR/preview build** — CI artifact for a commit/PR; not DEV;
- **release candidate/request** — coordinated version-bump branch/commit awaiting release acceptance;
- **stable release** — official version tag/release artifacts;
- **cluster live** — installed runner/server/relay/MCP runtime version;
- **Windows live** — installed desktop application version;
- **live** alone — avoid when cluster versus Windows matters; state the surface explicitly.

### WB-DEC-010 — Non-human waits do not end development — CURRENT

The execution stopping rule in `docs/GOVERNANCE.md` is normative: CI/build/runner/release/deployment waits are not completion. Continue in-scope work until evidence-backed completion, an inspectable result, or a genuine human-only blocker.

## Superseded / rejected decisions and behaviours

### Conversation history as project authority — REJECTED

Old behaviour: large chats/handoffs could effectively become the only specification.

Replacement: canonical repository documents + decision register. Conversations are historical evidence only.

Do not reintroduce: never resolve a requirement conflict by choosing an old chat over current canonical docs without first updating the canonical record.

### Activity-first Dashboard as the primary Operations design — SUPERSEDED

Old behaviour: Recent Activity / Active Tasks / flat projects / flat worker health dominated the main dashboard.

Replacement: execution-oriented Operations semantics and lane-based work inventory. Activity/history may remain useful, but it is not the definition of running work.

### All-zero Operations lanes while real work exists — REJECTED

Old failure: remote work existed but the board presented zero work because the live inventory path failed or discarded it.

Replacement: bounded privacy-safe remote inventory plus truthful job projection.

### Active session lease means every recent operation is running — REJECTED

Old/current-0.9.54 behaviour: completed/failed safe-hands operations marked `Active=true` by a session lease are converted to `TaskRunning`, creating misleading counts such as `Running 100`.

Replacement: WB-DEC-003. Presence is not individual execution.

Do not reintroduce: no test may assert that `completed + session_active` alone equals a running job.

### Fake elapsed-time-derived percentage progress — REJECTED

Replacement: measured, stage-based or indeterminate progress only.

### Generic Windows command/shell authority for convenience — REJECTED

Replacement: outbound typed/allowlisted Windows operations.

### Implicit ChatGPT → OpenClaw development delegation — REJECTED

Replacement: direct bounded Workbench operations by default; autonomous delegation only through explicit eligible paths. Authenticated private continuation is a separate trusted transport and does not reopen implicit delegation.

### OpenClaw as the primary Workbench coder — SUPERSEDED

Replacement: WB-DEC-001. OpenClaw is optional operator/autonomous capacity.

### 90-second Unreal startup smoke — SUPERSEDED

Replacement: bounded five-minute diagnostic startup with privacy-safe classification.

### Old Unreal `TNotNull`/stack-overflow crash as the current failure — HISTORICAL

The old crash was removed. The latest inherited evidence before the reset is that Unreal 5.8.1 stayed alive beyond the old limit and reached a five-minute timeout classified `zen`. The root cause/current acceptance remains unresolved.

### Blender GUI preferences are sufficient for factory headless GPU rendering — SUPERSEDED

Replacement: the headless render path explicitly configures Cycles GPU/OptiX and enables non-CPU devices. End-to-end live render acceptance remains a separate verification question.

### No-op release retrigger as normal release procedure — REJECTED

It is tolerated only as the known temporary workaround in WB-DEC-008 until release publication is made reliable.

## Ambiguous / unresolved product decisions

These items are intentionally not promoted into requirements by assumption:

- whether the current roughly four-hour ChatGPT/session presence lease should remain exactly four hours;
- the final unified authoritative job model spanning scheduler-native tasks, CI, direct server operations, Windows operations and AI workers;
- interactive drag/reorder controls beyond persisted priority and FIFO;
- selected-job drawer details/controls (safe logs, artifacts, cancel/retry/requeue/prioritise) beyond what existing contracts explicitly guarantee;
- exact private relay retention/compaction policy for old transport history;
- whether the searchable decision/knowledge-graph capability remains desired in its old PR #110 shape. The capability may remain a future idea, but PR #110 is based on Workbench 0.9.10 and is not current implementation authority.

Resolve these by updating this register and the relevant contract before implementation.

## Do-not-reintroduce rules

Future changes MUST NOT:

- use deleted/pruned conversations as the project specification;
- equate active session/presence with each operation still executing;
- fabricate queue positions or progress percentages;
- hide real remote work as zero because the historical relay is large;
- restore generic inbound/generic-shell Windows control;
- silently route direct ChatGPT development into OpenClaw;
- treat OpenClaw as required for routine bounded operations;
- call a release complete before the required target runtime/release is actually verified;
- use a green responsiveness screenshot as proof of correct semantic state;
- restore the superseded 90-second Unreal smoke or describe the removed `TNotNull` crash as the current failure;
- assume Blender GUI preferences control factory-startup headless rendering;
- accept no-op release retrigger commits as the permanent desired release design.
