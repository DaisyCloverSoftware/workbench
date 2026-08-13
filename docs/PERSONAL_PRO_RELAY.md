# Personal ChatGPT Pro relay

Workbench's north-star loop needs more than task submission:

1. ChatGPT must be able to **request** work.
2. ChatGPT must be able to **receive** durable task state, results, and genuine attention questions.
3. A fresh conversation must be able to **resume compact project context** and persist new checkpoints/routines without making the human reconstruct history.

As of August 2026, OpenAI documents full custom-MCP write/modify actions for Business and Enterprise/Edu. Personal Pro can connect custom MCP servers for read/fetch. Workbench therefore supports a Git-backed action transport without pretending a read action is a write action and without automating the ChatGPT website.

## Full-MCP path

On a ChatGPT workspace with full custom-MCP actions:

```text
ordinary Chat
  -> Workbench custom MCP eyes / hands / memory tools
  -> private Workbench runner
  -> router/worker
  -> durable task + compact context
  -> ordinary Chat
```

## Personal Pro path

On personal Pro, a **private Git relay** can carry task execution and compact-context control traffic in both directions:

```text
ordinary Chat
  -> supported GitHub app write action
  -> private relay repository
        relay/inbox/<task-id>.json
        relay/control/<control-id>.json
  -> workbench-relay
  -> authenticated loopback Workbench MCP
  -> router / worker / durable knowledge
  -> workbench-relay
  -> private relay repository
        relay/outbox/<task-id>.json
        relay/control-outbox/<control-id>.json
  -> supported GitHub app read action
  -> ordinary Chat
```

A Secure MCP Tunnel remains useful when direct read-only Workbench repository/status tools are desired, but it is not required for the basic request/result or compact-context loop.

This is deliberately a **transport adapter**, not a second task or memory engine. The relay hands requests, human answers and private context controls into the same authenticated Workbench MCP process.

## Task request envelope

A relay task request is one UTF-8 file:

```text
relay/inbox/<relay-id>.json
```

```json
{
  "version": 1,
  "id": "wb_20260812_201500_a1b2c3",
  "project": "workbench",
  "intent": "Implement the requested outcome and verify it."
}
```

The runner-side daemon validates the ID/project, resolves the repository only beneath `WORKBENCH_RUNNER_ROOT`, caps input size, and submits through the bearer-authenticated loopback Workbench MCP server.

## Task result and human answer

Task state is published to:

```text
relay/outbox/<relay-id>.json
```

Public-safe mode contains only transport identity, status and update time. Verified private mode can additionally include the worker report, error text and genuine attention question, unless secret-like detail is detected and withheld.

When a task genuinely needs human input, Chat writes:

```text
relay/answers/<relay-id>.json
```

```json
{
  "version": 1,
  "id": "wb_20260812_201500_a1b2c3",
  "answer": "Choose A."
}
```

Workbench consumes each distinct answer once, calls `resolve_attention`, and continues the same durable task. Ordinary worker/setup failures are still routed around automatically.

## Private compact-context controls

**Control envelopes are ignored in public-safe mode.** They are processed only when the relay is configured as a verified private transport.

A control request is written to:

```text
relay/control/<control-id>.json
```

and its response appears at:

```text
relay/control-outbox/<control-id>.json
```

Every control ID is idempotent for the exact envelope content. A changed envelope gets a new digest and can update the same control ID deliberately.

### Resume a fresh conversation

If direct `get_context_pack` is unavailable, Chat writes:

```json
{
  "version": 1,
  "id": "ctx_20260812_201500_a1b2c3",
  "action": "context",
  "project": "workbench",
  "query": "the work I am about to continue",
  "max_items": 10,
  "max_chars": 16000
}
```

The response contains Workbench's bounded context pack: latest checkpoint plus relevant project/global memory and reusable routines. The old conversation is not required.

### Save a compaction checkpoint

```json
{
  "version": 1,
  "id": "cp_20260812_202000_a1b2c3",
  "action": "checkpoint",
  "project": "workbench",
  "summary": "Compact current project state.",
  "decisions": ["One durable decision."],
  "open_loops": ["One unresolved thread."],
  "next_actions": ["One likely next action."]
}
```

### Save durable memory

Project-scoped:

```json
{
  "version": 1,
  "id": "mem_20260812_202100_a1b2c3",
  "action": "remember",
  "project": "workbench",
  "scope": "project",
  "kind": "constraint",
  "title": "Compact active context",
  "summary": "Fresh conversations resume from durable context instead of replaying transcripts.",
  "tags": ["context", "resume"]
}
```

Deliberately cross-project knowledge uses `"scope": "global"` and does not require a project. Global scope should contain general reusable knowledge, not private project detail.

### Save or update a reusable routine/code pattern

```json
{
  "version": 1,
  "id": "routine_20260812_202200_a1b2c3",
  "action": "routine",
  "scope": "global",
  "name": "Atomic JSON state update",
  "description": "Persist a small state document without partial writes.",
  "triggers": ["json state", "atomic persistence"],
  "steps": ["Write a sibling temporary file.", "Rename it over the target."],
  "language": "text",
  "tags": ["storage"]
}
```

Saving the same routine name in the same scope updates it instead of creating a second copy.

### Recall without a direct MCP read connection

The same private control path supports:

- `"action": "recall"` with `query` and optional project;
- `"action": "routines"` with `query` and optional project.

Control results pass through the same secret-like output check before being written back to Git.

## Public vs private relay repositories

A public relay is for **harmless dogfood only**. Public mode deliberately publishes task status only and does not process `relay/control` at all. Never place private task intent, source detail, durable memory, compact checkpoints, credentials, customer data, incident detail or unreleased information in a public relay repository.

For real work, use a **dedicated private relay repository**. The Workbench source checkout and the relay transport clone are separate concerns. The relay inherits existing Git authentication; it does not create or store a GitHub token.

Private mode is fail-closed in the installer: reports and memory/context controls are not enabled unless a GitHub transport is verified private, or an operator explicitly attests privacy for a non-GitHub transport.

## Install

The Workbench MCP service must already be running. Harmless status-only dogfood can use the source checkout itself:

```bash
bash scripts/install-github-relay.sh
```

For real work, point the daemon at a separate private clone:

```bash
WORKBENCH_RELAY_REPO_DIR="$HOME/path/to/private-relay-clone" \
WORKBENCH_RELAY_PRIVATE=1 \
bash scripts/install-github-relay.sh
```

The private relay repository needs an existing target branch and non-interactive read/write Git authentication. It can otherwise be a minimal repository containing only the `relay/` transport files.

Configuration environment variables:

- `WORKBENCH_RELAY_REPO_DIR` — transport clone; defaults to the Workbench source checkout for harmless dogfood
- `WORKBENCH_RELAY_REMOTE` — default `origin`
- `WORKBENCH_RELAY_BRANCH` — default `main`
- `WORKBENCH_RELAY_INTERVAL` — default `10s`
- `WORKBENCH_RELAY_PRIVATE` — set to `1` for verified private transport; enables reports and memory/context controls
- `WORKBENCH_RELAY_RESULT_MODE` — `status` or `report`; private mode defaults to `report`
- `WORKBENCH_RELAY_ASSUME_PRIVATE` — explicit override for a non-GitHub private transport that cannot be verified automatically
- `WORKBENCH_MCP_URL` — default loopback Workbench MCP endpoint

## Why not automate the browser?

Workbench does not scrape ChatGPT output, inject DOM events or turn a consumer web session into an unofficial API. The relay uses supported GitHub actions for transport and keeps execution and durable memory behind Workbench's normal local policy boundary.
