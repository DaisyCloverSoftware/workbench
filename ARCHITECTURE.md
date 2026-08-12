# Architecture

## Components

### Native Windows desktop shell

`cmd/workbench` is a Win32 desktop application built entirely with the Go standard library. It owns the local UI, task list, provider dock, scratchpad, vault and routing preferences. No browser is used to render the Workbench interface.

### Core engine

`internal/core` owns durable state, provider discovery, cost-aware routing, task lifecycle, safe local tools, worker invocation and attention boundaries.

Routing is capability based. Current ordering is roughly:

`local/free → included subscription → harness → scarce Work/Codex → metered API (opt-in)`

The router records every attempted worker so failures are auditable.

### MCP server

`internal/mcp` exposes a localhost-only Streamable-HTTP-style JSON-RPC endpoint. It requires a bearer token and rejects unexpected browser origins.

The most important tools are:

- `apply_patch` — Chat supplies a patch; Workbench checks and applies it.
- `run_safe_command` — runs only allowlisted local build/test/status commands.
- `delegate_task` — hands a genuinely autonomous job to the router.
- `get_task` — lets the lead chat poll without asking the human for progress.
- `resolve_attention` — resumes a task after a genuine human decision.

This is the mechanism behind “chat for brains, Workbench for hands”.

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

`internal/platform` uses Windows DPAPI to encrypt secret values for the current Windows user. The persisted state contains ciphertext only. MCP exposes no raw-secret read tool.

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
