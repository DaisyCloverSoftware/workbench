# Personal ChatGPT Pro relay

Workbench's north-star loop needs two directions:

1. ChatGPT must be able to **request** work.
2. ChatGPT must be able to **receive** durable task state, results, and genuine attention questions.

As of August 2026, OpenAI documents full custom-MCP write/modify actions for Business and Enterprise/Edu. Personal Pro can connect custom MCP servers for read/fetch. Workbench therefore supports a Git-backed action transport without pretending a read action is a write action and without automating the ChatGPT website.

## Full-MCP path

On a ChatGPT workspace with full custom-MCP actions:

```text
ordinary Chat
  -> Workbench custom MCP delegate_task/apply_patch/run_safe_command
  -> private Workbench runner
  -> router/worker
  -> get_task/report
  -> ordinary Chat
```

## Personal Pro path

On personal Pro, Git can carry both directions:

```text
ordinary Chat
  -> supported GitHub app write action
  -> private relay repository: relay/inbox/<id>.json
  -> workbench-relay
  -> authenticated loopback Workbench MCP
  -> router/worker
  -> durable Workbench task
  -> workbench-relay
  -> private relay repository: relay/outbox/<id>.json
  -> supported GitHub app read action
  -> ordinary Chat
```

A Secure MCP Tunnel remains useful when direct read-only Workbench repository/status tools are desired, but it is no longer required for the basic request/result loop.

This is deliberately a **transport adapter**, not a second task engine. The relay hands requests and human answers into the same Workbench MCP process, so task state, routing, scarce-Work accounting and human-attention semantics remain identical.

## Request envelope

A relay request is one UTF-8 file:

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

The runner-side daemon:

- accepts only simple relay IDs;
- accepts only a single repository directory name for `project`;
- resolves that name beneath `WORKBENCH_RUNNER_ROOT`;
- caps envelope and intent size;
- fetches the envelope from a Git ref rather than executing repository content;
- submits through the bearer-authenticated loopback Workbench MCP server;
- persists a relay-ID -> Workbench-task-ID mapping locally;
- tags the durable task intent with `[relay:<id>]` for correlation.

## Result envelope

The relay publishes durable state to:

```text
relay/outbox/<relay-id>.json
```

A public-safe status envelope contains only transport identity, status and update time. On a verified private transport, `report` mode can additionally include the worker report, error text and attention question.

Example private result:

```json
{
  "version": 1,
  "id": "wb_20260812_201500_a1b2c3",
  "workbench_task_id": "task_123",
  "status": "needs_attention",
  "attention": "Choose A or B?",
  "updated_at": "2026-08-12T20:16:00Z"
}
```

Outbox publication is idempotent. The relay writes only when the generated envelope changes and retries a bounded number of times if another Git writer wins the branch race.

## Human answer envelope

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

Workbench consumes each distinct answer once, calls `resolve_attention`, and continues the same durable task. Ordinary worker/setup failures are still routed around automatically and do not become human questions.

## Public vs private relay repositories

A public relay is for **harmless dogfood only**. Public mode deliberately publishes status only. Never place private implementation details, credentials, unreleased product information, customer data, incident details, private task intent or other sensitive text in a public relay repository.

For real work, use a **dedicated private relay repository**. The Workbench source checkout and the relay transport clone are separate concerns. The relay inherits the operator's existing Git authentication; it does not create or store a GitHub token.

When report mode is enabled for a GitHub transport, the installer fails closed unless the repository can be verified as private. Unverified non-GitHub transports require an explicit private-transport override.

## Install

The Workbench MCP service must already be running. For a harmless status-only dogfood transport using the source checkout itself:

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
- `WORKBENCH_RELAY_PRIVATE` — set to `1` for a private transport; defaults to public-safe mode
- `WORKBENCH_RELAY_RESULT_MODE` — `status` or `report`; private mode defaults to `report`
- `WORKBENCH_RELAY_ASSUME_PRIVATE` — explicit override for a non-GitHub private transport that cannot be verified automatically
- `WORKBENCH_MCP_URL` — default loopback Workbench MCP endpoint

## Why not automate the browser?

Workbench does not scrape ChatGPT output, inject DOM events or turn a consumer web session into an unofficial API. The relay uses supported GitHub actions for transport and keeps execution behind Workbench's normal local policy boundary.
