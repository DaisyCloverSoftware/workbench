# Chat ↔ Workbench private dogfood

This document describes a generic private development loop:

```text
ordinary chat
  → Workbench MCP tools
  → private MCP transport
  → Workbench MCP on a private runner host
  → Workbench router
  → eligible coding worker / harness
  → durable Workbench task/report
  → chat
```

The private runner should not be exposed directly to the public internet. The cluster MCP server binds loopback only, and the transport sidecar makes outbound connections to the remote control plane.

## Install the private MCP service

From a private runner clone:

```bash
bash scripts/install-cluster-mcp.sh
```

The installer builds and starts the headless Workbench server, pins an active workspace, proves `/health`, and performs an authenticated MCP `initialize` smoke test on loopback.

Workbench stores a random local Authorization value in a mode-`0600` file under the current user's config directory. The value is not printed. A tunnel sidecar can inject it from that file rather than embedding it in source, model context, shell history or service arguments.

## Tunnel authorization

If using a managed private MCP tunnel, provision the tunnel in the relevant workspace/control plane, then run:

```bash
bash scripts/install-openai-tunnel.sh
```

The installer is designed to:

1. install a supported tunnel client for the local architecture;
2. verify downloaded checksums when published;
3. require the existing local Workbench Authorization file;
4. prompt locally for the tunnel identifier when not supplied by environment;
5. prompt silently for the tunnel runtime credential and save it in a mode-`0600` local file;
6. run a transport preflight against the loopback Workbench MCP endpoint;
7. start a persistent user service when available; and
8. verify transport health/readiness.

Keep the Workbench loopback credential and tunnel runtime credential separate. Neither belongs in model context, Git history, screenshots, public issues, or shell command lines.

## Connection metadata

A private developer-mode MCP connection may assign a technical connection identifier needed by a plugin mapping. Treat that identifier as deployment metadata. It is not a runtime credential, but it still does not need to be committed to the generic public source tree.

## Expected tool behaviour

A lead chat should normally begin with `get_workspace`, then use safe repository eyes and hands before autonomous delegation:

- `get_workspace` — active project and routing policy.
- `list_files` / `search_text` / `read_file` — model-safe repository inspection.
- `apply_patch` — exact Chat-generated patches.
- `run_safe_command` — allowlisted tests, builds, linting and status.
- `delegate_task` — autonomous implementation when actual agency is useful.
- `get_task` / `list_tasks` — durable task status.
- `resolve_attention` — resume after one genuine human decision.
- `save_note` — non-secret project notes.

After delegation, the client should poll task status itself. `running` is not a reason to interrupt the human. Only `needs_attention` should normally surface a human decision.

## Public-repository hygiene

Private dogfood topology is intentionally not documented here. Do not publish:

- machine or tailnet names;
- local usernames or home-directory paths;
- private addresses;
- provider-account inventories or entitlement details;
- credentials, tokens or connection secrets;
- private task content or logs.

Public source should describe deployment generically; environment-specific values belong only in private configuration.
