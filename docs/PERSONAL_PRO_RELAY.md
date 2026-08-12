# Personal ChatGPT Pro relay

Workbench's north-star loop needs two directions:

1. ChatGPT must be able to **request** work.
2. ChatGPT must be able to **read** durable task state and results.

As of August 2026, OpenAI documents full custom-MCP write/modify actions for Business and Enterprise/Edu. Personal Pro can connect custom MCP servers for read/fetch. Workbench therefore supports two transports without pretending a read action is a write action and without automating the ChatGPT website.

## Full-MCP path

On a ChatGPT workspace with full custom-MCP actions:

```text
ordinary Chat
  -> Workbench custom MCP delegate_task/apply_patch/run_safe_command
  -> private Workbench cluster
  -> router/worker
  -> get_task/report
  -> ordinary Chat
```

## Personal Pro path

On personal Pro:

```text
ordinary Chat
  -> supported GitHub app write action
  -> relay/inbox/<id>.json in the configured relay repository
  -> git fetch on the cluster
  -> workbench-relay
  -> authenticated loopback Workbench MCP delegate_task
  -> router/worker
  -> durable Workbench task
  -> read-only Workbench custom MCP through Secure MCP Tunnel
  -> ordinary Chat
```

This is deliberately a **transport adapter**, not a second task engine. The relay hands the request into the same Workbench MCP process, so task state, routing, scarce-Work accounting and human-attention semantics remain identical.

## Relay envelope

A relay request is one UTF-8 file:

```text
relay/inbox/<relay-id>.json
```

with:

```json
{
  "version": 1,
  "id": "wb_20260812_201500_a1b2c3",
  "project": "workbench",
  "intent": "Implement the requested outcome and verify it."
}
```

The cluster daemon:

- accepts only simple relay IDs;
- accepts only a single repository directory name for `project`;
- resolves that name beneath `WORKBENCH_RUNNER_ROOT` (default `~/src`);
- caps envelope and intent size;
- fetches the envelope from a Git ref rather than executing repository content;
- submits through the bearer-authenticated loopback Workbench MCP server;
- persists a relay-ID -> Workbench-task-ID mapping locally;
- tags the durable task intent with `[relay:<id>]` so read-only clients can find it with `list_tasks`.

## Public vs private relay repositories

The Daisy Clover `workbench` repository is public. It is useful for **harmless dogfood only**. Do not place private implementation details, credentials, unreleased product information, customer data, incident details or other sensitive text in its relay inbox.

For real work, configure Workbench Relay against a **private clone**. The relay intentionally uses normal `git fetch`, so it inherits the operator's existing Git authentication rather than introducing another GitHub token format into Workbench.

A future GitHub App adapter can make private-relay onboarding smoother, but the on-disk envelope and Workbench handoff remain the same.

## Install

The Workbench MCP service must already be running.

```bash
bash scripts/install-github-relay.sh
```

The installer builds `~/.local/bin/workbench-relay`, proves the Git remote and local MCP handoff, then installs a persistent user service when available.

Configuration environment variables:

- `WORKBENCH_RELAY_REMOTE` (default `origin`)
- `WORKBENCH_RELAY_BRANCH` (default `main`)
- `WORKBENCH_RELAY_INTERVAL` (default `10s`)
- `WORKBENCH_MCP_URL` (default `http://127.0.0.1:8765/mcp`)
- `WORKBENCH_RUNNER_ROOT` (service default `$HOME/src`)

## Why not automate the browser?

Workbench should not scrape ChatGPT output, inject DOM events or otherwise turn a consumer web session into an unofficial API. Besides being brittle, that would defeat the project's goal of a trustworthy open integration layer.

The relay path uses supported product integrations for each direction: GitHub write actions for intent transport, and custom-MCP read access for private status/results.
