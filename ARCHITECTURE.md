# Architecture

This document describes the current intended Workbench architecture. Product/governance decisions are recorded in `docs/DECISIONS.md`; implementation status is recorded in `docs/CURRENT_STATE.md`.

## Responsibility model

Workbench is ChatGPT-first.

- **ChatGPT** is the primary reasoning/coding brain. It owns source decisions, Git/GitHub/PR work, CI/release orchestration and normal bounded machine-operation decisions.
- **Workbench** supplies durable project/task state, safe repository eyes/hands, scheduling/execution infrastructure, machine controls, private transport, Windows bridging and continuity across waits/chat boundaries.
- **Autonomous harnesses** such as OpenClaw are optional execution/operator capacity. They are not required for routine bounded operations and are not the default coder.
- **The human** is an authority/decision boundary, not a clipboard or message bus.

## Components

### Native Windows desktop shell

`cmd/workbench` is the production Win32 desktop application. It owns the local UI, project/task views, Operations projection, provider/worker status, vault and routing/preferences surfaces.

The desktop consumes two different kinds of operational data:

1. scheduler-native durable local task state from the Core engine;
2. privacy-safe remote inventory/activity from the cluster runner/private relay.

Those sources may be projected into one UI, but they are not automatically the same execution model. In particular, remote session presence is not equivalent to an individual job still running.

### Core engine

`internal/core` owns:

- durable JSON state and state normalisation;
- multi-project registry;
- provider discovery and capability/cost routing;
- durable task lifecycle, retries and external dependency waits;
- lane-aware scheduling and priority;
- truthful work-item/progress projection;
- safe repository operations;
- attention boundaries;
- private-relay continuation proof validation;
- project/global knowledge and continuation state.

Routing remains capability/trust/cost based. Metered and scarce routes remain policy-protected.

### Durable scheduler

Queued autonomous tasks are durable. Scheduler dispatch, not the delegation call itself, owns queued → routing transition.

Execution lanes with current scheduler capacity are:

- `server_ops`
- `ci_builds`
- `windows_workstation`
- `ai_workers`

`waiting` and `needs_you` are state lanes rather than executable capacity.

Priority is persisted and ranked Critical → High → Normal → Low, then FIFO for equal priority. Persisted zero-value priority means Normal for compatibility with historical tasks.

Progress is only:

- measured when a real total exists;
- stage-based when real stage count exists;
- indeterminate otherwise.

Elapsed time is not converted into a fake percentage.

### MCP server

`internal/mcp` exposes a loopback-only authenticated Streamable-HTTP/JSON-RPC surface. It rejects unexpected browser origins.

The model-safe design separates **bounded hands** from **autonomous delegation**:

- repository list/search/read and exact patch application;
- allowlisted build/test/status commands;
- durable task status and human-attention resolution;
- autonomous delegation only through eligible explicit routes.

Direct ChatGPT development operations must not silently instantiate or route through OpenClaw. Authenticated private-relay continuation is a distinct trusted path, not an exception that reopens implicit delegation.

### Private Git relay and runner

The private relay is a durable transport for Personal-Pro-style workflows and cluster controls. It carries bounded control requests/results and explicit autonomous/deferred work without placing raw credentials in payloads.

For Dashboard/activity reads, the runner builds a bounded current view of the append-oriented relay:

- all pending requests;
- recent activity;
- request/result counterparts;
- old completed history excluded from the live view.

This is a **projection/scaling mechanism**, not a retention policy. Underlying private transport retention/cleanup remains a separately governed operational concern.

`RunnerChatActivityInfo` can carry both transport/action state and a bounded `Active`/`ActiveKnown` session-presence decision. That session-presence signal may be useful to show an active project/chat, but MUST NOT be interpreted as proof that each completed action is still executing.

### Durable continuation across waits

Workbench can park a task on an external GitHub Actions dependency without keeping an AI worker alive. When the dependency becomes terminal, Workbench can append its owned dependency result and resume the original continuation.

Private-relay development continuation is authenticated with an HMAC derived from the local MCP credential and bound to:

- relay correlation ID;
- project;
- original continuation body.

The dependency locator line is intentionally excluded from the signed worker body because Workbench removes it on wake-up. The validator accepts only the single Workbench-owned dependency-update suffix after the proof; arbitrary appended content fails closed.

Automatic wake-up has live evidence. Full post-fix productive completion remains an acceptance item; see `docs/CURRENT_STATE.md`.

### Outbound Windows host bridge

Windows execution uses an outbound host bridge. The security boundary is intentionally typed/allowlisted.

Examples include typed Blender and Unreal operations. Workbench MUST NOT add a generic inbound listener or generic remote Windows shell merely for convenience.

The cluster/private control plane refers to hosts/projects through privacy-minimal identifiers where possible. Public source must not expose deployment-specific host identifiers, addresses, paths or topology.

### Provider / harness adapters

Provider/harness integrations are thin adapters around explicit capabilities. Current discovery can include local models, included coding CLIs, structured external harnesses and scarce/metered fallbacks according to policy.

OpenClaw is an adapter/operator capacity, not an architectural foundation.

### Vault

`internal/platform` uses Windows DPAPI to protect secret values for the current Windows user. Persisted state contains encrypted secret references/ciphertext; model-facing tools do not expose vault plaintext.

## Execution state versus presence/history

This distinction is normative and exists to prevent a repeat of the 0.9.54 Operations defect.

```text
actual job state                 session/project presence            operation history
----------------                 ------------------------            -----------------
queued                           active/recent session               completed
routing                          inactive session                    failed
running                                                              cancelled
waiting_dependency
waiting_retry
needs_attention
```

A projection may visually correlate these concepts, but MUST NOT coerce presence/history into job execution state.

## Durable task lifecycle

```text
queued
  │ scheduler dispatch
  v
routing
  v
running ───────────────┬── completed
  │                    ├── failed
  │                    ├── cancelled
  │                    ├── waiting_retry ── automatic wake ──> queued/routing
  │                    ├── waiting_dependency ── dependency terminal ──> resume
  │                    └── needs_attention ── human answer ──> queued/resume
```

Normal implementation choices are not attention boundaries.

## State/storage

The current durable Core state is a private local JSON file (`State.Version` 3 at this baseline), written through a temporary private file and atomic rename. It includes project registry, tasks, encrypted secret references and preferences.

This is not a relational database schema. State evolution is handled through decode/normalisation/runtime repair; changes to durable meaning require compatibility tests and documentation.

## Control plane versus execution plane

Status/health/cancellation and other bounded control operations should remain responsive while long operations execute. The architecture must not serialize unrelated control-plane reads behind long-running execution simply because both travel through the same transport.

Future unification should create an authoritative job/inventory contract rather than scrape activity and infer execution from recency.

## Deployment surfaces

There is no canonical website-style DEV instance. Use the terminology in WB-DEC-009:

- development source;
- PR/preview build;
- release candidate/request;
- stable release;
- cluster live;
- Windows live.

State which live surface was verified.

## Architecture evolution rule

Before changing a material responsibility boundary, durable-state meaning, trust boundary, execution-state semantic or deployment contract, update the canonical decision/architecture record first. Code does not silently become the new architecture.
