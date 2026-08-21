# Structured Harness Adapter Protocol — v1

Workbench is harness-agnostic. OpenClaw, Claude Code or another coding harness may be used as optional worker capacity, but no harness is the architecture or the default coder.

This document describes the current **structured harness adapter** contract implemented by `internal/core/harness_protocol.go`. Workbench Runner transport is a separate Workbench execution surface; do not confuse runner transport with the third-party harness protocol.

## Execution model

Workbench launches exactly one explicitly configured adapter executable directly:

- no command shell;
- no operator/model-supplied adapter arguments;
- no `{project}` / `{prompt}` command-template expansion;
- one bounded JSON job on stdin;
- exactly one bounded JSON result on stdout.

The adapter runs with its working directory set to the Workbench-owned isolated project/worktree path supplied in the job.

Workbench remains responsible for task lifecycle, isolated workspace ownership, review finalisation and publication. A harness result is never itself authority to push, publish or deploy.

## Protocol version and bounds

Current protocol version: **1**.

The encoded input job is bounded to 512 KiB. Worker output capture is bounded by Workbench's worker-stream limit. Oversized/truncated output is rejected rather than persisted as an apparently valid result.

The adapter's stdout must decode as exactly one JSON value. Unknown JSON fields and trailing JSON values are rejected.

## Input — `HarnessJob`

The v1 job contains:

```json
{
  "version": 1,
  "task_id": "...",
  "project_path": "...",
  "intent": "...",
  "prompt": "...",
  "capabilities": {
    "repository_read": true,
    "repository_write": true,
    "local_commands": true,
    "network_access": false,
    "publish": false,
    "deploy": false,
    "secrets": false
  }
}
```

`task_id`, `project_path` and `intent` are required.

The capability object is a least-authority contract. The current Workbench-built job permits work inside the isolated repository and local commands, while explicitly denying network access, publication, deployment and secret access.

The job deliberately contains no publication target, credentials, vault values, remote URL or provider-account information.

## Output — `HarnessJobResult`

The adapter returns:

```json
{
  "version": 1,
  "task_id": "...",
  "status": "completed | needs_attention | unavailable | failed",
  "report": "...",
  "attention": "...",
  "unavailable": "...",
  "category": "authentication | quota | permissions | timeout | temporary | adapter",
  "retryable": false
}
```

The returned `task_id` must exactly match the submitted job. Result version must be 1.

### `completed`

- May contain a bounded report.
- Must not also contain attention, unavailable or failure-category fields.
- A successful harness result does **not** publish the work. Workbench owns finalisation/review publication separately.

### `needs_attention`

- Must contain one bounded attention question.
- Must not also claim unavailable/failure state.
- This is reserved for a genuine human-only boundary; routine implementation choices and non-human waits should not be converted into attention requests.

### `unavailable`

- Represents an adapter/provider capacity problem rather than completed task work.
- May classify the reason as authentication, quota, permissions, timeout, temporary or adapter.
- Is treated as retryable by Workbench's execution layer.

### `failed`

- Represents task failure.
- Must not also contain attention/unavailable fields.
- Retryability is explicit in the result rather than inferred from prose markers.

## Adapter executable configuration

The adapter path is host-local operator configuration (`runner-harness.json` for runner-host configuration), not task transport or model-visible project state.

Workbench validates that the configured path resolves to one regular executable file:

- on Windows it must be a native `.exe` or `.com` executable, not a batch/shell script;
- on non-Windows it must have executable permission.

The status surface reports configuration/availability using the adapter filename only; deployment-specific full paths are not part of the public protocol.

## Security / ownership boundary

A structured harness receives enough authority to modify its isolated working tree and run local development commands. It does **not** receive Workbench's publication/deployment/secrets authority.

Workbench owns:

- durable task identity and lifecycle;
- scheduler/routing policy;
- isolated worktree creation/recovery;
- task/project validation;
- bounded input/output handling;
- human-attention state;
- deterministic review commit/fingerprint;
- review-branch publication and PR delivery;
- provider retry/cooldown policy.

See `docs/TASK_WORKSPACES.md`, `docs/CHANGESET_PUBLISHING.md`, `SECURITY.md` and `docs/GOVERNANCE.md`.

## Superseded harness behaviour

The old v0.4 document described SSH runner transport as the harness protocol and retained a shell command-template adapter with `{project}`/`{prompt}` expansion and prose `ATTENTION_REQUIRED` markers.

That model is **SUPERSEDED** for structured harness adapters. Do not reintroduce shell-template expansion or prose-marker control flow as the preferred harness contract.

Workbench Runner may still have its own structured request/response transport, but that is a Workbench execution transport rather than this third-party adapter protocol.

## Evolution rule

Changes to protocol version, authority/capabilities, accepted statuses or adapter execution semantics are material architecture/security changes. Update the canonical decision/security/architecture record and tests before or with implementation; do not let an adapter implementation silently redefine the protocol.
