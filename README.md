# Workbench

**The AI coder's control plane: use chat for brains, give it safe eyes and hands, and spend scarce agentic usage only when the job genuinely needs an autonomous worker.**

Workbench is an open-source, standalone developer IDE/control plane for coordinating AI accounts, local models and agent harnesses.

The core idea is deliberately simple:

> Tell Workbench the outcome. It chooses the cheapest eligible route, keeps going without constant supervision, and interrupts only for a decision that genuinely requires a human.

## Why Workbench exists

Modern AI-assisted development has a strange bottleneck: the models can do a huge amount of the intellectual work, but the human still ends up as the clipboard, dispatcher and babysitter between chats, coding agents, terminals and servers.

Workbench separates **intelligence** from **agency**:

1. Use ordinary chat for brains whenever possible.
2. Reuse compact project context, durable memory and proven routines before rediscovering work.
3. Use safe repository eyes so Chat can inspect source without spending an autonomous-agent run.
4. Use safe hands for exact patches, tests and builds.
5. Route true autonomous coding work to zero-marginal or included workers first.
6. Escalate to scarce agentic capacity only when cheaper eligible routes have not solved the task.
7. Keep metered APIs opt-in.
8. Interrupt the human only for a real permission or decision boundary.

## What works today

- One production task-first Windows application with a first-class multi-project sidebar, project-scoped notes/tasks, global attention navigation and advanced controls under Settings.
- Headless Linux/cluster runner with cost-aware routing, durable detached jobs and operator-only verified updates.
- Cluster project discovery across multiple authorised runner roots: legacy `WORKBENCH_RUNNER_ROOT` remains supported, `WORKBENCH_RUNNER_ROOTS` can configure a path-list, and conventional `~/src` plus `~/projects` roots are recognised by default when present. Duplicate repository names fail closed and receive opaque scoped `runner://rN/name` references rather than exposing host paths.
- Adapter discovery for local models, coding CLIs and external harnesses.
- Model-safe repository `list_files`, `search_text`, and `read_file` tools.
- Persistent project/global knowledge for facts, decisions, constraints, patterns, routines and reusable code.
- Compact continuation capsules so a fresh conversation can resume without replaying a long transcript.
- Safe hands for exact patch application and allowlisted build/test/status commands.
- Durable autonomous task delegation, retries, reports and genuine attention boundaries.
- Workbench-owned isolated task worktrees, deterministic review commits, controlled review-branch publication and retryable GitHub PR delivery without recoding.
- Project-aware MCP workspace discovery that exposes safe routing facts without exposing project notes, secrets or publication targets.
- Bounded worker and runner output so noisy CLIs cannot grow Workbench memory/state without limit while final attention/unavailable signals remain detectable.
- Secret-like content protection for model-readable memory plus a local encrypted vault whose plaintext is not exposed through MCP.
- Harness-agnostic architecture: OpenClaw is an adapter, not the foundation.
- Bidirectional Git relay transport for Personal Pro-style workflows. Private mode can discover repository roots and carries the same bounded `list_files`/`search_text`/`read_file`/`apply_patch`/`run_safe_command` safe-hands path as direct MCP, while autonomous `delegate_task` remains a separate escalation channel; public mode stays status-only.
- Verified `Workbench-Updater.exe` plus transactional Linux cluster maintenance with checksum/architecture validation and rollback.

## Desired user experience

You say:

> Implement the next Workbench task.

Then you go do something else.

Workbench returns either:

- **Done** — with a concise verified report; or
- **Needs you** — one concise decision or permission request that genuinely could not be resolved autonomously.

A new conversation can pick up the same project from its compact context and durable memory. Similar tasks can retrieve an existing routine or code pattern instead of starting from scratch.

No progress babysitting. No human acting as a message bus between AIs.

## Architecture

```text
Human intent
    |
lead chat / Workbench desktop
    |
project registry + context capsule + project/global memory + reusable routines
    |
MCP / relay / structured task transport
    |
Workbench router
    |-- local / zero-marginal
    |-- included-subscription workers
    |-- scarce agentic fallback
    `-- metered fallback (opt-in)
    |
runner / harness adapters
```

Workbench is intentionally provider- and harness-agnostic. Private infrastructure, machine names, account inventories, credentials and dogfood topology are configuration, **not public source material**.

See [docs/KNOWLEDGE_SYSTEM.md](docs/KNOWLEDGE_SYSTEM.md) for the persistent-memory and compaction model.

## Build

Workbench has no third-party Go dependencies.

```bash
go test ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w -H=windowsgui" -o Workbench.exe ./cmd/workbench
go build -o workbench-runner ./cmd/workbench-runner
go build -o workbench-server ./cmd/workbench-server
go build -o workbench-relay ./cmd/workbench-relay
```

## Private deployment

The repository contains generic installers for the runner, MCP service, tunnel sidecar and Git-backed relay. They intentionally store runtime credentials only in local protected files and do not include any maintainer-specific hostnames, usernames, addresses, account inventories or secrets.

For real workloads, use private authenticated transport. Never place private task intent into a public relay repository.

## Security posture

Workbench assumes developers may paste secrets by accident. AI-facing tools are deliberately narrow, secret-like content is refused in exposed paths, private MCP services bind loopback, credentials remain local, and metered/scarce routes are protected by policy.

See [SECURITY.md](SECURITY.md).

## Public-source privacy rule

Public source, issues, pull requests, release notes and examples must remain environment-agnostic. Do not publish machine or tailnet names, local usernames or home-directory paths, private addresses, deployment topology, provider-account inventory, entitlement state, runtime credentials, private task content, or private dogfood logs.

Maintainer-specific deployment information belongs only in local protected configuration or a private authenticated repository.

The public project is periodically republished from a sanitized source snapshot when required, so private dogfood metadata does not become part of its durable history.

## Licence

MIT.
