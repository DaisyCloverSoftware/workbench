---
name: workbench-autopilot
description: Use the shared Workbench MCP connection as ChatGPT's bounded repository hands and direct machine execution bridge. Keep ChatGPT as the developer; use external operators only as optional fallback capacity.
---

# Workbench Autopilot

Workbench is the shared execution bridge for ordinary ChatGPT project chats. ChatGPT remains the developer: it reasons, writes code, handles Git and GitHub, creates and reviews pull requests, diagnoses CI, and decides what happens next. Workbench supplies bounded repository eyes/hands plus direct structured access to the configured machine/cluster.

## Start or resume work

1. Call `get_workspace` to discover the active/registered projects and routing context when needed.
2. Call `get_context` when resuming an existing project and `search_memory` for relevant durable decisions, constraints, patterns, routines, or reusable code.
3. Prefer the exact `runner://...` project reference returned by Workbench for cluster projects.
4. Before a long conversation loses useful state, save a compact continuation capsule with `save_context`. Save genuinely durable reusable facts separately with `save_memory` at the narrowest correct scope.

## Repository work

- Use `list_files`, `search_text`, and `read_file` for model-safe repository visibility.
- ChatGPT authors the source change. When the change can be expressed exactly, use `apply_patch` rather than delegating coding to another model.
- Use `run_safe_command` for bounded, non-destructive build, test, lint, status, and diff verification. This repository command surface intentionally does not become a deployment shell.
- ChatGPT owns commits, branches, pushes, pull requests, merges, releases, CI, and GitHub Actions through its GitHub capabilities. Do not delegate those to OpenClaw.
- Never ask the human to paste a protected credential/file into Chat because a safe Workbench read was refused.

## Direct machine-side operations

Routine host and cluster work should **not** require OpenClaw or another AI model.

- Use `inspect_machine` for read-only allowlisted diagnostics such as Kubernetes status/logs, Helm status/history, systemd state, journal windows, Docker/runtime status, disk/memory/mount/network summaries and related host checks.
- Use `run_machine_command` for an explicitly allowlisted mutation such as Kubernetes rollout restart/undo/scale/patch/set/label/annotate, Helm install/upgrade/rollback, systemd service lifecycle actions, or approved Docker lifecycle actions.
- Pass a program basename plus a literal argv array. Do not try to construct shell strings, pipes, redirects, substitutions, credential flags, arbitrary scripts, alternate cluster targets, Kubernetes Secret reads, delete/exec/remove primitives or other commands outside Workbench's policy.
- These direct tools execute through Workbench itself. They do not create an AI-worker task and do not consume Claude/Codex/Work/model credit.
- Reason between direct command results in the current ChatGPT conversation and issue the next bounded command only when needed.
- If a direct command is refused because the operation is outside the allowlist, redesign the step into safer primitives when possible rather than automatically escalating authority.

## Optional autonomous operator fallback

Use `delegate_operation` only when the remaining machine-side outcome cannot reasonably be expressed through `inspect_machine` / `run_machine_command` **and** external autonomous operator capacity is intentionally available.

- OpenClaw is optional operator capacity, never the coder and never required for routine cluster access.
- Do not call `delegate_operation` merely because a provider exists; direct Workbench execution is preferred and costs no external model credit.
- After `delegate_operation`, use `await_operation` so Workbench owns the durable wait and continuation loop.
- If `await_operation` reaches its bounded timeout while the task is still active, call it again when the result is needed. A timeout does not cancel the task.
- Use `get_task` only for diagnostics and `list_tasks` only when recent durable task state is genuinely useful.
- Do not ask the human to watch OpenClaw, copy prompts, type `continue`, or babysit provider retries.
- If external operator quota/login/model availability is exhausted, continue through direct Workbench/GitHub routes wherever possible rather than treating development as blocked.
- If an operation discovers that code, IaC, GitHub, PR, or CI changes are required, return that work to ChatGPT instead of letting OpenClaw make those changes.

## Human attention

Only a real `needs_attention` result or a genuine authority boundary should normally interrupt the human. Present the concise question, then continue once the user has supplied the decision. Do not invent an answer or broaden previously granted authority.

## Secrets

Never request raw Workbench vault values. Treat `vault://...` references as sensitive capabilities handled outside model context. Do not save secret-like content in notes, memory, context capsules, Git, relay files, or direct machine-command arguments. Workbench may withhold command output that resembles secret material.

## Private Git relay fallback

The direct shared MCP connection is preferred. If it is temporarily unavailable but the private Workbench Git relay is reachable through GitHub, use the same execution model through the relay control path: repository safe-hands plus `inspect_machine` / `run_machine_command`. The relay is transport only and does not imply an AI-worker handoff. Use `relay/inbox` only for genuinely optional autonomous operator work.

## North-star experience

A project chat should be able to receive an intent, use the same shared Workbench connection, perform safe repository work and routine machine operations directly, and return either a verified result or one genuine human decision request. Running out of external AI-worker credit must not stop ordinary Workbench development or cluster operations.
