# Workbench

**The AI coder's control plane: use chat for brains, give it safe eyes, hands and durable memory, and spend scarce agentic usage only when the job genuinely needs an autonomous worker.**

Workbench is an open-source, standalone developer IDE/control plane for coordinating AI accounts, local models and agent harnesses.

The core idea is deliberately simple:

> Tell Workbench the outcome. It chooses the cheapest eligible route, remembers what matters, keeps going without constant supervision, and interrupts only for a decision that genuinely requires a human.

## Why Workbench exists

Modern AI-assisted development has a strange bottleneck: the models can do a huge amount of the intellectual work, but the human still ends up as the clipboard, dispatcher and babysitter between chats, coding agents, terminals and servers. Long-running work also gets trapped inside individual conversation windows, so new chats repeatedly rediscover decisions and rebuild familiar solutions.

Workbench separates **intelligence** from **agency**, and working context from **durable memory**:

1. Use ordinary chat for brains whenever possible.
2. Use safe repository eyes so Chat can inspect source without spending an autonomous-agent run.
3. Use safe hands for exact patches, tests and builds.
4. Store compact checkpoints and distilled decisions outside the conversation so a fresh chat can resume.
5. Reuse saved routines and code patterns instead of rebuilding the same solution class repeatedly.
6. Route true autonomous coding work to zero-marginal or included workers first.
7. Escalate to scarce agentic capacity only when cheaper eligible routes have not solved the task.
8. Keep metered APIs opt-in.
9. Interrupt the human only for a real permission or decision boundary.

## What works today

- Native standalone Windows application.
- Headless Linux/cluster runner with cost-aware routing.
- Adapter discovery for local models, coding CLIs and external harnesses.
- Model-safe repository `list_files`, `search_text`, and `read_file` tools.
- Safe hands for exact patch application and allowlisted build/test/status commands.
- Durable autonomous task delegation, retries, reports and attention boundaries.
- Bidirectional Git relay for request/result/attention transport, with public-safe and private modes.
- Durable project and cross-project memory with stable Git-backed project identity.
- Compact project checkpoints designed to survive conversation compaction/replacement.
- Bounded context packs containing the latest checkpoint plus relevant memories and routines.
- Reusable scoped routines/code templates with upsert/deduplication by routine name.
- Automatic compact successful-task outcome memory on knowledge-aware delegated tasks.
- Secret-like memory/routine rejection and a local encrypted vault whose plaintext is not exposed through MCP.
- Harness-agnostic architecture: OpenClaw is an adapter, not the foundation.

## Desired user experience

You say:

> Implement the next Workbench task.

Then you go do something else.

Workbench returns either:

- **Done** — with a concise verified report; or
- **Needs you** — one concise decision or permission request that genuinely could not be resolved autonomously.

And if that conversation disappears or gets compacted, the next one can ask Workbench for the compact project context and continue rather than making you reconstruct history.

No progress babysitting. No human acting as a message bus between AIs. No pretending chat history is a database.

## Architecture

```text
Human intent
    |
lead chat / Workbench desktop
    |
compact context + repo eyes/hands
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
    |
durable project/global memory + checkpoints + routines
```

Workbench is intentionally provider- and harness-agnostic. Private infrastructure, machine names, account inventories, credentials and dogfood topology are configuration, **not public source material**.

See [docs/CONTEXT_MEMORY.md](docs/CONTEXT_MEMORY.md) for the durable-context model.

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

For real workloads, use private authenticated transport. Never place private task intent or durable memory content into a public relay repository.

## Security posture

Workbench assumes developers may paste secrets by accident. AI-facing tools are deliberately narrow, secret-like content is refused in exposed paths and memory writes, private MCP services bind loopback, credentials remain local, and metered/scarce routes are protected by policy.

See [SECURITY.md](SECURITY.md).

## Public-source privacy rule

Public source, issues, pull requests, release notes and examples must remain environment-agnostic. Do not publish machine or tailnet names, local usernames or home-directory paths, private addresses, deployment topology, provider-account inventory, entitlement state, runtime credentials, private task content, durable private memory, or private dogfood logs.

Maintainer-specific deployment information belongs only in local protected configuration or a private authenticated repository.

## Licence

MIT.
