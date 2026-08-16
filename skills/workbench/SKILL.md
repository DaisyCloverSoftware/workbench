---
name: workbench-autopilot
description: Use Workbench as the developer's AI control plane. Prefer ordinary chat reasoning plus safe Workbench eyes and hands; reuse durable memory and routines; delegate autonomous coding only when it is useful; never make the human babysit progress.
---

# Workbench Autopilot

Workbench exists so the human specifies intent instead of operating the AI switchboard.

## Core rule

**Use chat for brains. Use Workbench for eyes and hands. Reuse what already worked. Spend scarce agentic capacity only for autonomy.**

Start development work by calling `get_workspace`, `get_context`, and `search_memory` when those Workbench read tools are available. A fresh chat should reconstruct the useful working state from compact context + durable memory rather than asking the human to repeat the project history.

## Persistent knowledge and compaction

Workbench memory is deliberately layered:

- a compact continuation context for the current project;
- project-scoped facts, decisions, constraints, patterns, routines and reusable code;
- global cross-project facts, patterns, routines and reusable code.

Before implementing a non-trivial feature, search memory for similar prior work and reusable routines. Prefer a known tested routine or code asset over recreating another version from scratch.

When a conversation is becoming long, do not wait for useful context to fall out of the model window. Save a bounded `save_context` capsule containing only the current objective, verified state, decisions, constraints, durable references, open threads and next useful action. Promote durable information separately with `save_memory` rather than bloating the capsule.

Project memories must not silently become global. Promote something to global scope only when it is genuinely reusable beyond the current project. Never store secret-like material in memory or context; use the vault for secrets.

## Preferred execution ladder

1. Restore compact context and search durable memory/routines.
2. Do planning, reasoning, code generation, debugging and review in the current ordinary Chat conversation when that is sufficient.
3. Use `list_files`, `search_text`, and `read_file` to inspect model-safe repository content. Do not delegate to a coding agent merely because Chat needs to see source files.
4. If you can express the change as an exact unified patch and the Workbench write tools are available, call `apply_patch` rather than delegating to a coding agent.
5. Call `run_safe_command` for non-destructive builds, tests, linting, diffs and repository status when the tool is available.
6. Call `delegate_task` when autonomous repository exploration or multi-step execution is genuinely useful and the connected plan permits the action. Workbench owns worker selection and protects scarce Work/Codex usage.
7. When an external dependency becomes the only blocker, hand the dependency to Workbench as a durable wait instead of holding an AI worker or chat turn open. Then do other independent useful work.
8. After a useful new pattern/routine is verified, save it at the narrowest correct scope so future tasks can reuse it.
9. Do not choose a metered API simply because it exists. Respect the workspace routing policy.

If a read/search tool refuses a file because it appears sensitive, do not ask the human to paste that file into Chat. Work with the safe context available or request only the minimum non-secret information actually required.

## Durable waits and external dependencies

**Never say “I’m monitoring”, “I’ll keep an eye on it”, or equivalent unless a durable Workbench watch actually exists.** A chat model promising to remember a future CI result is not monitoring.

When a GitHub Actions run is the only thing preventing useful continuation, create a durable continuation task through `delegate_task`. Put this exact control envelope on the first line of `intent`, followed by the work that should resume after the run becomes terminal:

```text
WORKBENCH_WAIT_GITHUB_ACTIONS: {"repository":"owner/repository","run_id":123456789}
Continue the original task after CI. Inspect the terminal result. If CI failed, diagnose and fix the in-scope failure autonomously; if it succeeded, continue the remaining in-scope work. Ask the human only for a genuine decision or authority boundary.
```

Use the real `owner/repository` slug and numeric GitHub Actions run ID. Do not create duplicate watches for the same project, run and continuation; Workbench deduplicates exact repeats as a second defence.

A task in `waiting_dependency` is **not an occupied AI worker**. Workbench persists the locator and continuation, checks the dependency with progressive backoff, survives restart, and queues the continuation automatically when the run completes. While it waits:

- do not tight-poll `get_task` or GitHub merely to ask “done yet?”;
- do not keep a coding worker/session alive just to wait;
- continue other independent useful work, including another project/task when appropriate;
- check the task later only when its result is needed for the work currently being performed;
- if Workbench reports the dependency probe unavailable, let its backoff continue unless another safe route is genuinely useful.

The initial implementation supports GitHub Actions. Treat the durable-wait concept as the general rule for future deploys, renders, approvals, availability and other external dependencies as Workbench adds adapters.

## Personal ChatGPT Pro relay

At the time this skill was authored, personal ChatGPT Pro can connect custom MCP servers for read/fetch but full custom-MCP write/modify actions are restricted to eligible workspace plans. Workbench therefore has a Git-backed fallback that does not automate or scrape ChatGPT.

When a Workbench write tool is unavailable because of the ChatGPT plan **and the GitHub app is connected with write permission**:

1. Choose a configured relay repository. For real work it must be private; a public relay is only for harmless dogfood.
2. Create a unique relay ID such as `wb_20260812_201500_a1b2c3`.
3. Through the GitHub app, create exactly one UTF-8 JSON file at `relay/inbox/<relay-id>.json`:

```json
{
  "version": 1,
  "id": "<relay-id>",
  "project": "<repository-directory-name>",
  "intent": "<the autonomous implementation outcome, with no credentials or secrets>"
}
```

4. Read `relay/outbox/<relay-id>.json` when the result is useful. Do not ask the human to check progress and do not hammer the relay in a tight loop.
5. For `queued`, `routing`, or `running`, continue other useful work when possible and re-check later rather than blocking the whole turn on status churn.
6. For `waiting_retry`, leave the durable retry timer to Workbench.
7. For `waiting_dependency`, leave the durable dependency watch to Workbench; it will resume the continuation automatically.
8. For `completed`, consume the returned report when private report mode is configured, verify important claims when useful, and continue the original request.
9. For `failed`, diagnose and recover automatically when another safe route exists.
10. For `needs_attention`, present only the concise Workbench question. After the human answers, create or replace `relay/answers/<relay-id>.json`:

```json
{
  "version": 1,
  "id": "<relay-id>",
  "answer": "<human decision>"
}
```

Then continue from the same task; do not turn the human into a message bus.

### Private relay memory/control

When the relay is private, use its control channel for memory and compaction whenever direct `save_memory`/`save_context` MCP actions are unavailable. Create one unique request at `relay/control/<control-id>.json`, then read `relay/control-outbox/<control-id>.json` when needed. Supported actions are `save_memory`, `search_memory`, `save_context`, and `get_context`.

Use `save_context` before a long conversation loses useful state, then use `get_context` in a fresh conversation rather than asking the human to recap it. Use `search_memory` before rebuilding similar logic. Save verified reusable routines/code with `save_memory` at project scope unless they are genuinely cross-project, in which case global scope is explicit.

The private control channel is one-shot per ID, project paths are resolved beneath the runner root, and secret-like results are withheld. **Never create or process memory/control requests through a public relay.**

The runner-side relay validates every envelope, maps `project` only beneath `WORKBENCH_RUNNER_ROOT`, hands requests/answers/control actions into the authenticated loopback Workbench MCP server, and publishes result envelopes back through Git. The transport distinction is an implementation detail; do not make the human act as the message bus.

A **public** relay repository is only for non-sensitive dogfood tasks and publishes status-only output. Never put private source, credentials, customer information, internal incident details, unreleased product strategy, private task intent, memory/context, or other sensitive text into a public relay. For real private work, Workbench must be configured against a private relay clone with appropriate Git credentials. If only a public relay is available and the task is sensitive, that is a genuine setup boundary and must be surfaced rather than leaked.

## Delegated tasks

After `delegate_task`, Workbench owns the durable task state. Do not ask the human to watch it and do not waste the rest of the turn on a status-only polling loop when useful work remains.

- `queued`, `routing`, `running`: check when needed; continue other safe independent work between checks instead of hammering status.
- `waiting_retry`: Workbench’s persisted retry timer owns the wait. Do not poll-loop it.
- `waiting_dependency`: Workbench’s persisted dependency watch owns the wait and will queue the continuation automatically. Do not poll-loop it.
- `completed`: inspect the report, verify important claims with safe tools when useful, then continue the original task. Save genuinely reusable discoveries as memory/routines.
- `failed`: diagnose the report and recover automatically when another safe approach exists.
- `needs_attention`: this is the only normal reason to interrupt the human. Present the concise Workbench question without progress chatter. After the human answers, use `resolve_attention` when available or the relay answer envelope on personal-plan transport, then continue the task.

A provider's local login, permission-mode, tool-denial, quota or setup problem is not by itself a human decision. Workbench should route around eligible worker failures where possible.

## Human authority

Do not silently expand authority. Require the human for genuine external, destructive, production, financial, credential-disclosure or materially scope-expanding decisions.

Normal in-scope repository reads, edits, local builds, tests and review do not require progress confirmations when the task already authorizes implementation.

## Secrets

Never ask Workbench to reveal raw vault values to the model. Treat vault references as capabilities handled outside model context. If `save_note`, `save_memory`, `save_context`, `read_file`, or `search_text` protects secret-like content, tell the user only that Workbench protected it; do not ask them to paste the secret into chat.

## North-star experience

The human should be able to say **“Implement the next Workbench task”**, leave, and return to either a completed verified result or one concise decision that genuinely required them — even if the original chat has long since been compacted or replaced. External waits must not strand the task or reserve an AI that could be doing something else.
