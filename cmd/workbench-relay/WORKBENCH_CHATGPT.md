# Workbench for ChatGPT

This file is the canonical bootstrap for ordinary ChatGPT conversations using a **private** Workbench Git relay.

Workbench also publishes `WORKBENCH_CAPABILITIES.json` beside this guide. That file is the machine-readable protocol summary: Workbench version, supported control actions, relay paths, and the opaque-project-reference rule. Fresh chats can read it first for fast capability discovery, then use this guide for behavioural and safety policy.

## Operating rule

ChatGPT is the primary reasoning/coding brain. Use Workbench safe hands before autonomous delegation. Do **not** ask the user to copy prompts into OpenClaw, Codex, a terminal, or another chat when this relay can carry the operation. Never put passwords, API keys, OAuth tokens, private keys, cookies, or other raw secrets in this repository.

For normal development work:

1. Discover the target cluster project with `list_projects` if needed. If a known GitHub repository is missing from the runner, use `ensure_github_project` rather than asking the human to clone/import it manually.
2. Inspect with `list_files`, `search_text`, and `read_file`.
3. When ChatGPT can determine the change itself, use `apply_patch`, then `run_safe_command` to verify it.
4. Use `save_note`, `save_memory`, and `save_context` for durable non-secret context when useful.
5. Use autonomous delegation (`relay/inbox`) only when unattended exploration or implementation is genuinely useful. Workbench owns worker routing; do not bypass its cost/risk policy.
6. Only surface a Workbench `needs_attention` question to the user.

## Private safe-hands control channel

Write one request file to `relay/control/<id>.json`. `<id>` must be unique, 8-80 characters, and contain only letters, digits, `_` or `-`.

Workbench writes the result to `relay/control-outbox/<id>.json`. Do not overwrite or reuse an old request ID. After creating a request, read the matching result when it appears; do not busy-poll aggressively.

Base envelope:

```json
{
  "version": 1,
  "id": "unique_request_id",
  "action": "list_projects",
  "args": {}
}
```

`list_projects` returns both a human-readable `name` and a stable `ref`. **Prefer the exact returned `ref` in later project-scoped requests.** A unique repository normally has a backwards-compatible ref such as `runner://family-vault`. If the same directory name exists under more than one authorised project root, Workbench returns an opaque scoped ref such as `runner://r2/shared`; do not guess which copy the user intended.

If the target is a GitHub repository that ChatGPT already knows but it is absent from `list_projects`, `ensure_github_project` can make it runner-ready without human clipboard work. It accepts only a GitHub `owner/name` slug, never an arbitrary URL, never overwrites an existing directory, and uses only the runner's existing non-interactive GitHub credentials.

Project-scoped example:

```json
{
  "version": 1,
  "id": "unique_request_id",
  "action": "read_file",
  "project": "runner://family-vault",
  "args": {
    "path": "README.md",
    "start_line": 1,
    "end_line": 120
  }
}
```

### Safe-hands actions

- `list_projects` — no project; `args: {}`.
- `ensure_github_project` — no project; `repository` is one GitHub `owner/name` slug. Returns the existing project untouched or clones it atomically into an authorised runner root. No arbitrary hosts/URLs and no overwrite.
- `list_files` — project required; `subdir` optional, `limit` optional (max 1000).
- `search_text` — project required; `query` required, `subdir` optional, `limit` optional (max 200).
- `read_file` — project required; `path` required, `start_line`/`end_line` optional.
- `apply_patch` — project required; `patch` is a unified Git patch. Prefer this when ChatGPT has reasoned out the exact source change.
- `run_safe_command` — project required; `command` passes through Workbench's bounded non-destructive development allowlist. No arbitrary shell, deployment, direct push, or destructive command.
- `save_note` — project required; `note` required. Secret-like content is refused.
- `search_memory` — project optional; `query` optional, `limit` optional (max 20).
- `save_memory` — `scope` (`project` or `global`), optional `kind` (`fact`, `decision`, `constraint`, `pattern`, `routine`, `code`), `title`, `content`, optional `tags`, optional `source`; project is required for project scope.
- `get_context` — project required; `args: {}`.
- `save_context` — project required; `objective`, `state`, and optional `decisions`, `constraints`, `references`, `open_threads`, `next_action`.
- `update_status` — no project; `args: {}`; returns only categorical private-loop maintenance state.
- `update_workbench` — no project; `args: {}`; schedules Workbench's app-owned non-destructive maintenance update. Use only when Workbench itself actually needs updating.

Example runner-ready import:

```json
{
  "version": 1,
  "id": "ensure_project_001",
  "action": "ensure_github_project",
  "args": {
    "repository": "ExampleOrg/example-project"
  }
}
```

Example safe command:

```json
{
  "version": 1,
  "id": "family_status_001",
  "action": "run_safe_command",
  "project": "runner://family-vault",
  "args": {
    "command": "git status --short"
  }
}
```

## Autonomous delegation channel

Use this only when safe hands are insufficient and a separate autonomous worker is genuinely warranted.

Write `relay/inbox/<id>.json` using the exact project `ref` returned by `list_projects`:

```json
{
  "version": 1,
  "id": "unique_task_id",
  "project": "runner://family-vault",
  "intent": "Outcome to achieve; describe the result, not shell commands"
}
```

Workbench writes task state/result to `relay/outbox/<id>.json`.

Workbench chooses the worker according to its configured routing policy. Do not assume Codex/Work is the default. Local/cheap/included routes retain first refusal; OpenClaw cloud-model selection happens only if Workbench routing reaches that stage; scarce Codex/Work remains a last resort.

## Genuine human attention

If an autonomous task reaches `needs_attention`, ask the user only the exact decision/permission Workbench needs. Then write `relay/answers/<id>.json`:

```json
{
  "version": 1,
  "id": "same_task_id",
  "answer": "The user's answer"
}
```

Do not invent answers or use this path to bypass a permission boundary.

## Fresh-chat bootstrap

A ChatGPT conversation that can access the user's connected GitHub account can bootstrap itself without code-search indexing: use repository search to locate the user's **private repository whose name contains `workbench-relay`**, read `WORKBENCH_CAPABILITIES.json` and `WORKBENCH_CHATGPT.md` from that repository root, then follow the advertised protocol and this guide.

A useful one-time global ChatGPT instruction is:

> For software-development, repository, server and cluster work, use Workbench as my default execution layer. Use my connected GitHub app to find my private repository whose name contains `workbench-relay`, read `WORKBENCH_CAPABILITIES.json` and `WORKBENCH_CHATGPT.md` from its repository root, and follow that Workbench protocol and guide. ChatGPT is the primary brain/coder; prefer Workbench safe hands before autonomous delegation. If a known GitHub project is absent from the Workbench runner, use Workbench's bounded runner-ready project import instead of asking me to clone it manually. Do not ask me to copy/paste prompts into OpenClaw, Codex, terminals or other chats when Workbench can carry the operation. Preserve Workbench routing and keep scarce Codex/Work as a last resort. If Workbench is unavailable, tell me what is unavailable instead of silently reverting to manual message shuttling.

This bootstrap contains no Workbench bearer token or provider credential.
