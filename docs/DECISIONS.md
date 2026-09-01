# Workbench decision register

This is the durable decision and supersession register for material Workbench product behaviour. Status values are **CURRENT**, **SUPERSEDED**, **REJECTED**, **HISTORICAL**, **IMPLEMENTATION DETAIL** and **AMBIGUOUS**.

## Current decisions

### WB-DEC-001 — ChatGPT-first responsibility split — CURRENT

Workbench is a ChatGPT-first developer/operations workspace and execution bridge.

- ChatGPT owns primary reasoning, source-code decisions, Git/GitHub work, PR/CI/release orchestration, subsequent engineering decisions and normal bounded machine operations.
- Workbench owns durable task state, safe repository eyes/hands, scheduling/execution infrastructure, machine operations, the outbound Windows bridge, private relay transport and continuity across chat boundaries.
- OpenClaw is not part of normal routing. It is an owner-selected execution mode and may be used only when the owner explicitly requests OpenClaw by name for the applicable operation.
- The human is interrupted only for a genuine human-only decision, permission or authority boundary.

This aligns `README.md`, `VISION.md`, `docs/CHATGPT_BOOTSTRAP.md` and the private relay bootstrap and supersedes older architecture wording that made command-template or autonomous delegation look central.

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

Direct ChatGPT development operations MUST NOT silently become autonomous coding or OpenClaw operations.

The authenticated private relay may carry an explicit durable continuation handoff. Continuation authority is HMAC-bound to the relay correlation, project and original continuation body; Workbench may append its exact dependency-result suffix when a durable wait becomes terminal. Arbitrary appended content remains fail-closed.

Authenticated continuation is a development-continuity transport. It is not OpenClaw authorization and cannot satisfy the explicit owner-by-name requirement in WB-DEC-018.

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

Generic memories/rules that say “DEV checkpoint” must be interpreted through these Workbench-specific surfaces; they do not create a conventional DEV deployment by implication.

### WB-DEC-010 — Non-human waits do not end development — CURRENT

The execution stopping rule in `docs/GOVERNANCE.md` is normative. Within an authorised sprint, ordinary asynchronous engineering latency is execution, not an owner handoff point.

- Once a sprint enters **IN DEVELOPMENT**, Development owns progression through **ENGINEERING VERIFICATION**, **DEPLOYING TO REVIEW TARGET**, **READY FOR OWNER OBSERVATION**, and then **AWAITING OWNER OBSERVATION** when the exact inspectable candidate is genuinely ready.
- CI/GitHub checks, automated tests, builds, PR/preview artifacts, in-scope release/image/package publication, deployment/installation, rollout/readiness/smoke checks, runner operations and equivalent asynchronous engineering waits do not require owner keepalive prompts such as `continue`, `carry on`, `check again`, or equivalent.
- Development re-checks those operations itself and continues when they become terminal.
- If an in-scope automated check fails, Development investigates it, makes an already-authorised in-scope correction, reruns the applicable verification, and follows the sprint forward without requiring a fresh owner prompt.
- Correction rounds obey the same rule from **CORRECTIONS IN DEVELOPMENT** through applicable verification/deployment to **READY FOR OWNER RE-OBSERVATION**.
- The normal owner-return point is **AWAITING OWNER OBSERVATION**, after applicable engineering verification has passed, the candidate has reached the agreed inspectable target, the target has been verified to be running/serving that candidate, and the Sprint Review is ready.
- Earlier owner interruption is reserved for genuine human-only decisions, permissions, approval gates or authority blockers that prevent safe in-scope continuation.

This decision governs orchestration continuity only. It does not widen sprint scope, bypass publication/security/approval boundaries, transfer publication/deployment authority to a coding worker, authorise external/scarce model-credit use, infer OpenClaw authorization, waive semantic acceptance or owner sign-off, or permit the next sprint before **SIGNED OFF**.

### WB-DEC-011 — Configured execution target and worker readiness must be truthful — CURRENT

When a project/task is explicitly configured for a runner, machine or execution target, Workbench MUST honour that target contract.

- If the configured runner/target is unavailable, show the work as blocked/unavailable or otherwise truthfully not runnable there.
- Do not silently execute locally merely to avoid reporting that the configured target is unavailable.
- Do not report a worker as ready when it is not actually available.
- Worker location, capability, readiness and fallback state must be reported truthfully.
- A fallback route may be used only when the current routing/policy contract explicitly permits it; fallback is not permission to lie about where work ran.
- This generic execution-target rule does not authorize OpenClaw. OpenClaw is governed exclusively by the explicit owner authorization boundary in WB-DEC-018.

### WB-DEC-012 — External model-credit consumption requires explicit user authorisation — CURRENT

Testing, experiments, probes and development must not consume external/metered/scarce model credit merely to see whether a provider works.

- Local, zero-marginal and already-authorised included routes may be used according to policy, except that OpenClaw still requires the separate explicit owner authorization in WB-DEC-018.
- Any action that will spend external model credit or scarce paid quota for a test/experiment requires explicit user authorisation for that spend.
- Provider availability may be inspected through non-consuming/categorical mechanisms when available.
- Never turn a governance, health or routing check into a paid model invocation by convenience.

### WB-DEC-013 — “Finished” means a coherent production product, not an engineering preview — CURRENT

Workbench MUST distinguish product completion from scaffolding or component progress.

A backend-only milestone, skeleton UI, prototype, partial control surface, test harness, engineering preview or isolated feature does not become a “finished”, “production-ready” or equivalent Workbench product merely because it runs or demonstrates architecture.

Completion claims must describe the actual whole-product experience and applicable acceptance evidence. Intermediate technical progress may be described precisely as such.

### WB-DEC-014 — Release, artifact and deployment verification are separate gates — CURRENT

A release is not complete merely because version code was merged or CI passed.

Where a release is expected, verify the actual release/tag and the expected downloadable artifact(s). Where deployment/installation is also required, verify that target separately.

Use precise evidence states:

- merged source is not yet released;
- a release/tag without the required artifact is not accepted as complete;
- released is not automatically deployed/installed;
- deployed/installed is not automatically semantically verified.

### WB-DEC-015 — Canonical GitHub source outranks operational checkouts — CURRENT

The canonical public GitHub `main` repository is the Workbench development/project source of truth. Cluster/local source checkouts are operational copies.

Operational checkouts SHOULD be clean and aligned to the intended canonical source when used for current Workbench operations, but their local commits, untracked files, stale remote-tracking refs or worktrees do not redefine the product.

Any local-only work discovered during governance/cleanup must be preserved/classified before deletion or realignment; it is not silently promoted into canonical source.

### WB-DEC-016 — Pre-reset historical source is archived for preservation, never authority — CURRENT

The governance reset consolidated removed public branch histories behind archival tag `archive/pre-governance-reset-20260821` after proving every removed branch tip remained reachable and the archive checkpoint tree exactly matched canonical `main` at the archive checkpoint.

- The archive exists only for historical/audit recovery.
- It is not a development branch, product baseline or requirements source.
- New work MUST start from canonical `main`.
- Do not copy/revive code, tests or decisions from the archive merely because they exist there; reintroducing any historical behaviour requires a fresh decision against current canonical docs.
- A separate local audit ref may preserve previously local-only operational history without becoming public/canonical authority.

### WB-DEC-017 — Private relay historical transport is intentionally retained pending a retention policy — CURRENT

The bounded live projection prevents historical relay volume from defining current Operations state. The underlying private relay history is intentionally retained at this reset checkpoint.

Mass deletion/compaction is not required to complete public-source repository cleanup. A future retention/compaction policy must explicitly protect pending requests, audit/rollback needs, privacy and transport correctness before destructive cleanup is authorised.

### WB-DEC-018 — OpenClaw is explicit-owner-request-only — CURRENT

OpenClaw is installed/available capacity only. It is not a normal Workbench route and it is unavailable to automatic selection.

- ChatGPT and Workbench MUST NOT select, invoke, suggest or use OpenClaw automatically.
- Only an explicit owner instruction naming OpenClaw for the applicable operation authorizes OpenClaw.
- Unless such an instruction exists, effective OpenClaw authorization is **DENIED**.
- Availability, difficulty, duration, an allowlist miss, missing Workbench capability, CI/deployment failure, Kubernetes/Docker/systemd/Helm trouble, Bash requirements, prior OpenClaw use, historical task state, provider/model/tool availability or old routing configuration are not authorization.
- Routine server/cluster/host/runtime work uses direct Workbench controls first. Reviewed `scripts/ops/*.sh` operations are the multi-step Bash path.
- If direct execution cannot express the operation, ChatGPT must safely decompose it, use an existing reviewed operation, implement an appropriate bounded Workbench capability/reviewed operation where authorised, or report the exact capability/authority blocker. It must not fall through to OpenClaw.
- `relay/inbox` may preserve deliberate explicit-use functionality, but `[workbench:operations]` is only routing metadata. A separate explicit owner-authorization signal is required, and normal routing must not be able to create authorization merely by entering the operations lane.
- Historical OpenClaw tasks and conversations do not influence current routing.

Compatibility intent: OpenClaw may remain installed and usable for a future operation the owner explicitly assigns to OpenClaw; otherwise it is inert from ChatGPT/Workbench routing perspective.

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

### Automatic OpenClaw routing/fallback — REJECTED

Old behaviour: ChatGPT/Workbench guidance or routing could treat OpenClaw/autonomous operations as an escalation/fallback when the direct structured machine surface could not express the task.

Replacement: WB-DEC-018. A direct-capability miss is a decomposition/capability/reviewed-operation/blocker decision, never OpenClaw authorization. Only an explicit owner instruction naming OpenClaw authorizes that operation.

Do not reintroduce: provider availability, `[workbench:operations]`, task difficulty, an allowlist miss or prior OpenClaw use must never create OpenClaw authorization.

### Implicit ChatGPT → OpenClaw development delegation — REJECTED

Replacement: direct bounded Workbench operations and ChatGPT-owned engineering. Authenticated private continuation is a separate trusted development-continuity transport and does not reopen OpenClaw routing.

### OpenClaw as the primary Workbench coder — SUPERSEDED

Replacement: WB-DEC-001 and WB-DEC-018. OpenClaw is owner-selected machine-operation capacity only when explicitly authorized by name.

### Silent local execution when a configured runner is unavailable — REJECTED

Replacement: WB-DEC-011. Unavailable configured execution must remain visible as unavailable/blocked unless a policy-authorised non-OpenClaw fallback is explicitly selected and truthfully reported. OpenClaw remains subject to WB-DEC-018.

### Spending external model credit for unapproved tests/probes — REJECTED

Replacement: WB-DEC-012.

### Engineering preview/skeleton presented as finished Workbench — REJECTED

Replacement: WB-DEC-013.

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
- whether the searchable decision/knowledge-graph capability remains desired in its old PR #110 shape. The capability may remain a future idea, but PR #110 is based on Workbench 0.9.10, was closed unmerged during this reset, and is not current implementation authority;
- whether cross-process locking is required for relay-state persistence under the supported process topology. A discovered local experiment was archived as evidence and explicitly not accepted as the design.

Resolve these by updating this register and the relevant contract before implementation.

## Do-not-reintroduce rules

Future changes MUST NOT:

- use deleted/pruned conversations as the project specification;
- treat the archival tag or an operational checkout as requirements/product authority;
- equate active session/presence with each operation still executing;
- fabricate queue positions or progress percentages;
- hide real remote work as zero because the historical relay is large;
- silently execute locally when a configured runner/target is unavailable;
- report worker location, capability or readiness falsely;
- spend external model credit/scarce paid quota on tests or experiments without explicit user authorisation;
- present a skeleton, prototype, partial UI/backend demo or engineering preview as a finished coherent Workbench product;
- restore generic inbound/generic-shell Windows control;
- silently route direct ChatGPT development into OpenClaw;
- select, invoke, suggest or use OpenClaw merely because direct Workbench execution is unavailable or inconvenient;
- treat `[workbench:operations]`, prior OpenClaw tasks, provider availability or historical routing state as owner authorization;
- treat OpenClaw as required for routine bounded operations;
- call a release complete before the expected actual release/tag and downloadable artifact are verified;
- conflate released, deployed/installed and semantically verified state;
- use a green responsiveness screenshot as proof of correct semantic state;
- restore the superseded 90-second Unreal smoke or describe the removed `TNotNull` crash as the current failure;
- assume Blender GUI preferences control factory-startup headless rendering;
- accept no-op release retrigger commits as the permanent desired release design.
