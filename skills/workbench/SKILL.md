---
name: workbench-autopilot
description: Use Workbench as the developer's AI control plane. Prefer ordinary chat reasoning plus safe Workbench eyes and hands; use durable compact context and reusable routines; delegate autonomous coding only when useful; never make the human babysit progress.
---

# Workbench Autopilot

Workbench exists so the human specifies intent instead of operating the AI switchboard.

## Core rule

**Use chat for brains. Use Workbench for eyes, hands and durable memory. Spend scarce agentic capacity only for autonomy.**

Treat the current conversation as a working buffer, not the project database. A project must remain resumable after this chat is compacted or replaced.

When direct Workbench read tools are available, start or resume development work with `get_workspace` and `get_context_pack`. The context pack is the compact source of prior project state, relevant decisions/constraints, cross-project knowledge and reusable routines.

## Preferred execution ladder

1. Load the compact project context relevant to the current intent.
2. Do planning, reasoning, code generation, debugging and review in the current ordinary Chat conversation when that is sufficient.
3. Use `list_files`, `search_text`, and `read_file` to inspect model-safe repository content. Do not delegate to a coding agent merely because Chat needs to see source files.
4. Check `find_routines` before inventing a procedure or code pattern that may already have been solved.
5. If you can express the change as an exact unified patch and the Workbench write tools are available, call `apply_patch` rather than delegating to a coding agent.
6. Call `run_safe_command` for non-destructive builds, tests, linting, diffs and repository status when the tool is available.
7. Call `delegate_task` when autonomous repository exploration or multi-step execution is genuinely useful and the connected plan permits the action. Workbench owns worker selection, attaches relevant durable context and protects scarce Work/Codex usage.
8. Do not choose a metered API simply because it exists. Respect the workspace routing policy.

If a read/search tool refuses a file because it appears sensitive, do not ask the human to paste that file into Chat. Work with the safe context available or request only the minimum non-secret information actually required.

## Durable memory and compaction

Save **distilled conclusions**, not transcripts.

Use `remember` when a decision, constraint, lesson or pattern has become durable. Keep project-specific facts project-scoped. Promote something to global memory only when it is genuinely reusable across unrelated projects.

Use `save_routine` when a procedure, template or code pattern has proven reusable. Prefer updating the existing scoped routine name over saving a second near-duplicate. Store the general mechanism, not project secrets or copied credentials.

Use `save_checkpoint` at meaningful milestones and before the working conversation becomes unwieldy. A good checkpoint contains:

- a compact current-state summary;
- decisions that must survive compaction;
- unresolved loops;
- likely next actions.

Do not try to preserve every turn. The goal is a small active context backed by durable structured memory. A fresh conversation should be able to call `get_context_pack` and continue without asking the human to reconstruct history.

Successful tasks delegated through the durable-knowledge path are retained as compact project outcome memory automatically when safe to do so.

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

4. Poll `relay/outbox/<relay-id>.json` through the GitHub app. Do not ask the human to check progress.
5. For `queued`, `routing`, or `running`, keep polling without progress chatter.
6. For `completed`, consume the returned report when private report mode is configured, verify important claims when useful, and continue the original request.
7. For `failed`, diagnose and recover automatically when another safe route exists.
8. For `needs_attention`, present only the concise Workbench question. After the human answers, create or replace `relay/answers/<relay-id>.json`:

```json
{
  "version": 1,
  "id": "<relay-id>",
  "answer": "<human decision>"
}
```

Then resume polling the same outbox file until terminal.

The cluster-side relay validates every envelope, maps `project` only beneath `WORKBENCH_RUNNER_ROOT`, hands requests/answers into the authenticated loopback Workbench MCP server, and publishes result envelopes back through Git. The transport distinction is an implementation detail; do not make the human act as the message bus.

A **public** relay repository is only for non-sensitive dogfood tasks and publishes status-only output. Never put private source, credentials, customer information, internal incident details, unreleased product strategy, private task intent, or other sensitive text into a public relay. For real private work, Workbench must be configured against a private relay clone with appropriate Git credentials. If only a public relay is available and the task is sensitive, that is a genuine setup boundary and must be surfaced rather than leaked.

Until personal-plan relay control envelopes for memory writes are available, do not leak checkpoint/memory content into a public relay merely to persist it. Use direct Workbench memory writes when the connected plan permits them, and otherwise keep the compact context in the current session until a private supported path is configured.

## Delegated tasks

After `delegate_task`, or after submitting through the Git relay, keep working. Poll the relevant status mechanism yourself until the task reaches a terminal state.

- `queued`, `routing`, `running`: keep polling. Do not ask the user to check it.
- `completed`: inspect the report, verify important claims with safe tools when useful, persist any genuinely reusable lesson/routine, then continue the original request.
- `failed`: diagnose the report and recover automatically when another safe approach exists.
- `needs_attention`: this is the only normal reason to interrupt the human. Present the concise Workbench question without progress chatter. After the human answers, use `resolve_attention` when available or the relay answer envelope on personal-plan transport, then continue polling.

A provider's local login, permission-mode, tool-denial, quota or setup problem is not by itself a human decision. Workbench should route around eligible worker failures where possible.

## Human authority

Do not silently expand authority. Require the human for genuine external, destructive, production, financial, credential-disclosure or materially scope-expanding decisions.

Normal in-scope repository reads, edits, local builds, tests, review, context retrieval and safe local memory maintenance do not require progress confirmations when the task already authorizes implementation.

## Secrets

Never ask Workbench to reveal raw vault values to the model. Treat vault references as sensitive capabilities handled outside model context. If memory, routine, note, file-read or search tools protect secret-like content, tell the user only that Workbench protected it; do not ask them to paste the secret into chat.

## North-star experience

The human should be able to say **“Implement the next Workbench task”**, leave, and return to either a completed verified result or one concise decision that genuinely required them — even if the original conversation no longer exists.
