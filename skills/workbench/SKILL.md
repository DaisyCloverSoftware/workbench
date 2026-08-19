---
name: workbench-autopilot
description: Use Workbench as ChatGPT's bounded repository hands and direct machine execution bridge. Select the transport the current ChatGPT plan actually permits; keep ChatGPT as the developer and external operators optional.
---

# Workbench Autopilot

Workbench is the shared execution bridge for ordinary ChatGPT project chats. ChatGPT remains the developer: it reasons, writes code, handles Git and GitHub, creates and reviews pull requests, diagnoses CI, and decides what happens next. Workbench supplies bounded repository eyes/hands plus direct structured access to the configured machine/cluster.

## Choose the transport, not a different execution model

- If the current ChatGPT workspace exposes Workbench's full custom MCP write actions, use the direct Workbench app/MCP tools.
- If full custom MCP writes are unavailable, use the private Workbench Git relay as the **primary write/mutation transport**. With connected GitHub, locate the user's private repository whose name contains `workbench-relay`, read `WORKBENCH_CAPABILITIES.json` and `WORKBENCH_CHATGPT.md`, then use `relay/control`.
- The relay and MCP paths represent the same Workbench authority model. Do not treat the relay as an AI-worker handoff.
- Never make the human upgrade a plan, paste terminal commands, or open OpenClaw merely because one ChatGPT transport cannot expose writes.

## Start or resume work

1. Discover the active/registered projects and routing context through the selected Workbench transport.
2. Read the latest compact context and search memory when resuming existing work.
3. Prefer the exact `runner://...` project reference returned by Workbench for cluster projects.
4. Before a long conversation loses useful state, save a compact continuation capsule. Save genuinely durable reusable facts separately at the narrowest correct scope.

## Repository work

- Use Workbench file/list/search/read controls for model-safe repository visibility when needed.
- ChatGPT authors the source change. When the change can be expressed exactly, use Workbench patch hands rather than delegating coding to another model.
- Use the bounded safe-command surface for non-destructive build, test, lint, status, and diff verification. This repository command surface intentionally does not become a deployment shell.
- ChatGPT owns commits, branches, pushes, pull requests, merges, releases, CI, and GitHub Actions through its GitHub capabilities.
- Never ask the human to paste a protected credential/file into Chat because a safe Workbench read was refused.

## Direct machine-side operations

Routine host and cluster work should **not** require OpenClaw or another AI model.

- Use `inspect_machine` (or the relay action of the same name) for read-only allowlisted diagnostics such as Kubernetes status/logs, Helm status/history, systemd state, journal windows, Docker/runtime status, disk/memory/mount/network summaries and related host checks.
- Use `run_machine_command` for an explicitly allowlisted mutation such as Kubernetes rollout restart/undo/scale/patch/set/label/annotate, Helm install/upgrade/rollback, systemd service lifecycle actions, or approved Docker lifecycle actions.
- Pass a program basename plus a literal argv array. Do not construct shell strings, pipes, redirects, substitutions, credential flags, arbitrary scripts, alternate cluster targets, Kubernetes Secret reads, delete/exec/remove primitives or commands outside Workbench's policy.
- Direct commands execute through Workbench itself. They do not create an AI-worker task and do not consume Claude/Codex/Work/model credit.
- Reason between command results in the current ChatGPT conversation and issue the next bounded command only when needed.
- If a direct command is refused because an operation is outside the allowlist, redesign the step into safer primitives when possible rather than silently escalating authority.

## Optional autonomous operator fallback

Use `delegate_operation` / `relay/inbox` only when the remaining machine-side outcome cannot reasonably be expressed through direct Workbench commands **and** external autonomous operator capacity is intentionally available.

- OpenClaw is optional operator capacity, never the coder and never required for routine cluster access.
- Do not delegate merely because a provider exists; direct Workbench execution is preferred and costs no external model credit.
- Workbench owns the durable wait/continuation loop for any optional delegated task.
- Do not ask the human to watch OpenClaw, copy prompts, type `continue`, or babysit provider retries.
- If external operator quota/login/model availability is exhausted, continue through direct Workbench/GitHub routes wherever possible rather than treating development as blocked.
- If an operation discovers that code, IaC, GitHub, PR, or CI changes are required, return that work to ChatGPT instead of letting OpenClaw make those changes.

## Human attention

Only a real `needs_attention` result or a genuine authority boundary should normally interrupt the human. Present the concise question, then continue once the user has supplied the decision. Do not invent an answer or broaden previously granted authority.

## Secrets

Never request raw Workbench vault values. Treat `vault://...` references as sensitive capabilities handled outside model context. Do not save secret-like content in notes, memory, context capsules, Git, relay files, or direct machine-command arguments. Workbench may withhold command output that resembles secret material.

## North-star experience

A project chat should be able to receive an intent, use the Workbench transport available to that plan, perform safe repository work and routine machine operations directly, and return either a verified result or one genuine human decision request. Running out of external AI-worker credit or lacking full-MCP write support must not stop ordinary Workbench development or cluster operations.
