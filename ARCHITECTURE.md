# Architecture

## Components

### Native Windows desktop shell

`cmd/workbench` is a Win32 desktop application built entirely with the Go standard library. It owns the local UI, task list, provider dock, scratchpad, vault and routing preferences. No browser is used to render the Workbench interface.

### Core engine

`internal/core` owns durable state, provider discovery, cost-aware routing, task lifecycle, safe local tools, worker invocation, attention boundaries and durable compact knowledge.

Routing is capability based. Current ordering is roughly:

`local/free → included subscription → harness → scarce Work/Codex → metered API (opt-in)`

The router records every attempted worker so failures are auditable.

### Durable context and knowledge

Workbench treats chat history as a working buffer rather than the project database. `internal/core/knowledge.go` stores distilled project/global memory, compact project checkpoints, reusable routines and optional code templates in local protected application state.

Project identity prefers a normalised Git remote so knowledge survives checkout-path changes. Repositories without a usable remote receive a machine-local fallback identity.

A bounded context pack combines:

- the latest project checkpoint;
- relevant project decisions, constraints, lessons and outcomes;
- deliberately global cross-project knowledge;
- reusable project/global routines and code.

Tasks delegated through the MCP path receive relevant durable context automatically. Successful task outcomes are retained as compact project memories unless they appear secret-like.

See `docs/CONTEXT_MEMORY.md`.

### MCP server

`internal/mcp` exposes a localhost-only Streamable-HTTP-style JSON-RPC endpoint. It requires a bearer token and rejects unexpected browser origins.

The most important tools are:

- `get_context_pack` — resumes work from compact durable context rather than replaying chat history.
- `recall_memory` / `find_routines` — retrieve prior decisions and proven reusable procedures.
- `remember` / `save_checkpoint` / `save_routine` — persist distilled knowledge and reusable code.
- `apply_patch` — Chat supplies a patch; Workbench checks and applies it.
- `run_safe_command` — runs only allowlisted local build/test/status commands.
- `delegate_task` — hands a genuinely autonomous job to the router with relevant durable context attached.
- `get_task` — lets the lead chat poll without asking the human for progress.
- `resolve_attention` — resumes a task after a genuine human decision.

This is the mechanism behind “chat for brains, Workbench for hands”.

### Git relay

`cmd/workbench-relay` is a transport adapter for environments where direct Workbench write actions are unavailable. It accepts inbox requests, publishes durable outbox status/results and consumes human-answer envelopes. Real task intent/results belong in a private relay repository; public relay mode is status-only.

### Provider adapters

Current discovery targets:

- Ollama
- Antigravity CLI
- GitHub Copilot CLI
- Claude Code
- OpenClaw / configurable harness command
- Codex CLI

Provider adapters are intentionally thin. The core does not assume all models expose identical features.

### Vault

`internal/platform` uses Windows DPAPI to encrypt secret values for the current Windows user. The persisted state contains ciphertext only. MCP exposes no raw-secret read tool. Model-facing memory writes also reject probable secret material rather than turning the knowledge store into a second vault.

## Task lifecycle

```text
queued
  ↓
routing
  ↓
running ────────────────┐
  │                     │
  ├── completed         │
  ├── failed            │
  ├── cancelled         │
  └── needs_attention ──┘
          │ human answer
          └── queued → route/resume
```

A worker asks for human attention by emitting a line that begins exactly:

`ATTENTION_REQUIRED:`

Normal implementation decisions are not attention boundaries.

## Future architecture

The command-template OpenClaw adapter is intentionally a bridge. The next protocol layer should add a structured remote Runner API with capability advertisement, durable job IDs, progress events, artefacts, cancellation and resumable human input.

The knowledge layer intentionally separates durable records from retrieval. Deterministic lexical ranking is the first implementation; semantic indexing, consolidation, synchronised private stores and richer reusable artefact references can be added without changing the core project/global/checkpoint/routine model.
