---
name: workbench-autopilot
description: Use the shared Workbench MCP connection as ChatGPT's bounded repository hands and supervised machine-side operations bridge. Keep ChatGPT as the developer and interrupt the human only for a genuine decision or permission boundary.
---

# Workbench Autopilot

Workbench is the shared execution bridge for ordinary ChatGPT project chats. ChatGPT remains the developer: it reasons, writes code, handles Git and GitHub, creates and reviews pull requests, diagnoses CI, and decides what happens next. Workbench supplies bounded repository eyes/hands and the machine-side operations lane.

## Start or resume work

1. Call `get_workspace` to discover the active/registered projects and routing context when needed.
2. Call `get_context` when resuming an existing project and `search_memory` for relevant durable decisions, constraints, patterns, routines, or reusable code.
3. Prefer the exact `runner://...` project reference returned by Workbench for cluster projects.
4. Before a long conversation loses useful state, save a compact continuation capsule with `save_context`. Save genuinely durable reusable facts separately with `save_memory` at the narrowest correct scope.

## Repository work

- Use `list_files`, `search_text`, and `read_file` for model-safe repository visibility.
- ChatGPT authors the source change. When the change can be expressed exactly, use `apply_patch` rather than delegating coding to another model.
- Use `run_safe_command` for bounded, non-destructive build, test, lint, status, and diff verification.
- ChatGPT owns commits, branches, pushes, pull requests, merges, releases, CI, and GitHub Actions through its GitHub capabilities. Do not delegate those to OpenClaw.
- Never ask the human to paste a protected credential/file into Chat because a safe Workbench read was refused.

## Machine-side operations

Use `delegate_operation` only when the remaining work genuinely requires machine/host/runtime access ChatGPT does not have directly, such as shell, systemd, Docker, Kubernetes, Helm, runner repair, deployment, or runtime verification.

- OpenClaw is the operator, never the coder.
- After `delegate_operation`, use `await_operation` so Workbench owns the durable wait and continuation loop.
- If `await_operation` reaches its bounded timeout while the task is still active, call it again when the result is needed. A timeout does not cancel the task.
- Use `get_task` only for diagnostics and `list_tasks` only when recent durable task state is genuinely useful.
- Do not ask the human to watch OpenClaw, copy prompts, type `continue`, or babysit provider retries.
- Provider quota, context-window, login, local tool, or transient worker problems are routing failures for Workbench to handle where another eligible route exists.
- If an operation discovers that code, IaC, GitHub, PR, or CI changes are required, return that work to ChatGPT instead of letting OpenClaw make those changes.

## Human attention

Only a real `needs_attention` result should normally interrupt the human. Present the concise question exactly, then call `resolve_attention` with the user's answer and continue the same durable task. Do not invent an answer or broaden previously granted authority.

## Secrets

Never request raw Workbench vault values. Treat `vault://...` references as sensitive capabilities handled outside model context. Do not save secret-like content in notes, memory, context capsules, Git, or relay files.

## Private Git relay fallback

The direct shared MCP connection is preferred. If it is temporarily unavailable but the private Workbench Git relay is reachable through GitHub, use the canonical relay bootstrap in `cmd/workbench-relay/WORKBENCH_CHATGPT.md` rather than making the human act as a message bus. The relay is a fallback transport, not a different execution model.

## North-star experience

A project chat should be able to receive an intent, use the same shared Workbench connection, perform safe repository work, delegate only genuine machine operations, and return either a verified result or one genuine human decision request. Opening OpenClaw manually must not be part of the normal workflow.
