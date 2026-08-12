---
name: workbench-autopilot
description: Use Workbench as the developer's AI control plane. Prefer ordinary chat reasoning plus safe Workbench eyes and hands; delegate autonomous coding only when it is useful; never make the human babysit progress.
---

# Workbench Autopilot

Workbench exists so the human specifies intent instead of operating the AI switchboard.

## Core rule

**Use chat for brains. Use Workbench for eyes and hands. Spend scarce agentic capacity only for autonomy.**

Start development work by calling `get_workspace` so you know the active repository, routing policy, and available workers.

## Preferred execution ladder

1. Do planning, reasoning, code generation, debugging and review in the current ordinary Chat conversation when that is sufficient.
2. Use `list_files`, `search_text`, and `read_file` to inspect model-safe repository content. Do not delegate to a coding agent merely because Chat needs to see source files.
3. If you can express the change as an exact unified patch and the Workbench write tools are available, call `apply_patch` rather than delegating to a coding agent.
4. Call `run_safe_command` for non-destructive builds, tests, linting, diffs and repository status when the tool is available.
5. Call `delegate_task` when autonomous repository exploration or multi-step execution is genuinely useful and the connected plan permits the action. Workbench owns worker selection and protects scarce Work/Codex usage.
6. Do not choose a metered API simply because it exists. Respect the workspace routing policy.

If a read/search tool refuses a file because it appears sensitive, do not ask the human to paste that file into Chat. Work with the safe context available or request only the minimum non-secret information actually required.

## Personal ChatGPT Pro relay

At the time this skill was authored, personal ChatGPT Pro can connect custom MCP servers for read/fetch but full custom-MCP write/modify actions are restricted to eligible workspace plans. Workbench therefore has a supported fallback that does not automate or scrape ChatGPT.

When a Workbench write tool is unavailable because of the ChatGPT plan **and the GitHub app is connected with write permission**:

1. Use the Workbench read-only tools to inspect the active workspace and decide the exact outcome.
2. Create a unique relay ID such as `wb_20260812_201500_a1b2c3`.
3. Through the GitHub app, create exactly one UTF-8 JSON file in the configured relay repository at `relay/inbox/<relay-id>.json` with this schema:

```json
{
  "version": 1,
  "id": "<relay-id>",
  "project": "<repository-directory-name>",
  "intent": "<the autonomous implementation outcome, with no credentials or secrets>"
}
```

4. The cluster-side Workbench relay fetches that Git ref, validates the envelope, maps `project` only beneath `WORKBENCH_RUNNER_ROOT`, and hands the request to the authenticated loopback Workbench MCP server.
5. Poll the read-only `list_tasks` tool until a task whose intent begins `[relay:<relay-id>]` appears, then poll that task with `get_task` until terminal.
6. Do not ask the human to check progress. The transport distinction is an implementation detail.

A **public** relay repository is only for non-sensitive dogfood tasks. Never put private source, credentials, customer information, internal incident details, unreleased product strategy, or other sensitive task text into a public relay. For real private work, Workbench must be configured against a private relay clone with appropriate Git credentials. If the available relay is public and the task is sensitive, request private-relay setup instead of leaking the task.

## Delegated tasks

After `delegate_task`, or after a task appears from the GitHub relay, keep working. Poll `get_task` yourself until the task reaches a terminal state.

- `queued`, `routing`, `running`: keep polling. Do not ask the user to check it.
- `completed`: inspect the report, verify important claims with safe tools when useful, then continue the original request.
- `failed`: diagnose the report and recover automatically when another safe approach exists.
- `needs_attention`: this is the only normal reason to interrupt the human. Present the concise Workbench question without progress chatter. After the human answers, call `resolve_attention` if that write action is available. On a read-only personal-plan connection, use the configured supported relay/approval transport rather than disguising a write as a read tool.

A provider's local login, permission-mode, tool-denial, quota or setup problem is not by itself a human decision. Workbench should route around eligible worker failures where possible.

## Human authority

Do not silently expand authority. Require the human for genuine external, destructive, production, financial, credential-disclosure or materially scope-expanding decisions.

Normal in-scope repository reads, edits, local builds, tests and review do not require progress confirmations when the task already authorizes implementation.

## Secrets

Never ask Workbench to reveal raw vault values to the model. Treat vault references as capabilities handled outside model context. If `save_note`, `read_file`, or `search_text` protects secret-like content, tell the user only that Workbench protected it; do not ask them to paste the secret into chat.

## North-star experience

The human should be able to say **“Implement the next Workbench task”**, leave, and return to either a completed verified result or one concise decision that genuinely required them.
