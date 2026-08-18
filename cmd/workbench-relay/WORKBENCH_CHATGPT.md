# Workbench for ChatGPT

This file is the canonical bootstrap for ordinary ChatGPT conversations using a **private** Workbench Git relay.

Workbench also publishes `WORKBENCH_CAPABILITIES.json` beside this guide. That file is the machine-readable protocol summary: Workbench version, supported control actions, relay paths, the supervised-operations marker, and the opaque-project-reference rule. Fresh chats can read it first for fast capability discovery, then use this guide for behavioural and safety policy.

## Operating rule

ChatGPT is the primary reasoning and coding brain. **The human is never the transport layer.** Use Workbench safe hands before autonomous delegation. Do **not** ask the user to copy prompts into OpenClaw, Codex, a terminal, or another chat when this relay can carry the operation. Do **not** ask the user to watch OpenClaw and type “continue” when it stops: Workbench owns supervised continuation. Never put passwords, API keys, OAuth tokens, private keys, cookies, or other raw secrets in this repository.

There are two deliberately different autonomous lanes:

- **Development lane:** ordinary autonomous coding escalation. ChatGPT remains the normal coder and should prefer safe hands when it can determine the change itself.
- **Operations lane:** host/server/cluster/runtime work that ChatGPT cannot execute directly. Workbench sends this to OpenClaw as an operator, not as the application coder. Workbench automatically re-engages progress-only or bounded stalled OpenClaw invocations until the requested outcome is verified or the continuation budget is exhausted. OpenClaw source edits are isolated and rejected/discarded; source changes come back to ChatGPT.

For normal development work:

1. Discover the target cluster project with `list_projects` if needed. If a known GitHub repository is missing from the runner, use `ensure_github_project` rather than asking the human to clone/import it manually.
2. Inspect with `list_files`, `search_text`, and `read_file`.
3. When ChatGPT can determine the change itself, use `apply_patch`, then `run_safe_command` to verify it.
4. Use `save_note`, `save_memory`, and `save_context` for durable non-secret context when useful.
5. Use ordinary autonomous delegation (`relay/inbox`) only when unattended exploration or implementation is genuinely useful. Workbench owns worker routing; do not bypass its cost/risk policy.
6. For shell/systemd/Docker/Kubernetes/Helm/deployment/runner/host operations that ChatGPT itself cannot perform, use the **operations lane** described below instead of telling the human to open OpenClaw.
7. Only surface a Workbench `needs_attention` question to the user.

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

For ordinary development delegation, use a normal intent. Workbench chooses the coding worker according to its configured routing policy. Do not assume Codex/Work is the default. Local/cheap/included routes retain first refusal; OpenClaw cloud-model selection happens only if Workbench routing reaches that stage; scarce Codex/Work remains a last resort.

### Supervised OpenClaw operations lane

For **host/server/cluster/runtime operations that ChatGPT cannot execute through safe hands**, use the same inbox but prefix the intent exactly with the operations marker advertised by `WORKBENCH_CAPABILITIES.json`:

```json
{
  "version": 1,
  "id": "cluster_operation_001",
  "project": "runner://infrastructure",
  "intent": "[workbench:operations] Deploy the already-approved DEV change, verify rollout health and report the resulting image/revision. Do not modify source files."
}
```

This marker is a control instruction between ChatGPT and Workbench; the user should never have to type it.

Operations-lane guarantees:

- Workbench reserves execution for its cluster-runner/OpenClaw operator path rather than handing infrastructure work to a coding model.
- OpenClaw is explicitly told that ChatGPT owns application/source changes.
- A clean OpenClaw exit is **not** treated as success merely because it stopped talking. It must verify the objective and emit Workbench's completion marker.
- If OpenClaw returns a progress-only response, Workbench automatically invokes it again against the current host/cluster state. Bounded unresponsive invocations are also re-engaged automatically.
- Workbench performs up to six supervised continuation passes before reporting a worker failure back to ChatGPT. It does not ask the human to type “continue”.
- If OpenClaw changes repository files anyway, those edits stay in an isolated Workbench task workspace and are discarded/rejected. ChatGPT must make any required code/IaC change itself.
- Ordinary login/quota/tool problems are worker failures for ChatGPT/Workbench to handle, not reasons to make the user babysit OpenClaw.
- Only a genuine human decision or ungranted irreversible/destructive/production permission becomes `needs_attention`.

When an operations task finishes or fails, read its `relay/outbox/<id>.json` report and continue the ChatGPT-side workflow yourself. Do not ask the human to copy the report back from OpenClaw.

## Genuine human attention

If an autonomous task reaches `needs_attention`, ask the user only the exact decision/permission Workbench needs. Then write `relay/answers/<id>.json`:

```json
{
  "version": 1,
  "id": "same_task_id",
  "answer": "The user's answer"
}
```

Do not invent answers or use this path to bypass a permission boundary. Workbench resumes the same durable task after the answer.

## Fresh-chat bootstrap

A ChatGPT conversation that can access the user's connected GitHub account can bootstrap itself without code-search indexing: use repository search to locate the user's **private repository whose name contains `workbench-relay`**, read `WORKBENCH_CAPABILITIES.json` and `WORKBENCH_CHATGPT.md` from that repository root, then follow the advertised protocol and this guide.

A useful one-time global ChatGPT instruction is:

> For software-development, repository, server and cluster work, use Workbench as my default execution layer. Use my connected GitHub app to find my private repository whose name contains `workbench-relay`, read `WORKBENCH_CAPABILITIES.json` and `WORKBENCH_CHATGPT.md` from its repository root, and follow that Workbench protocol and guide. ChatGPT is the primary brain/coder; prefer Workbench safe hands before autonomous delegation. If a known GitHub project is absent from the Workbench runner, use Workbench's bounded runner-ready project import instead of asking me to clone it manually. For host/server/cluster/runtime operations that ChatGPT cannot execute itself, use Workbench's supervised operations lane so OpenClaw acts as the operator and Workbench automatically keeps it moving; do not ask me to copy/paste prompts into OpenClaw, Codex, terminals or other chats, and do not ask me to nudge OpenClaw with “continue”. Preserve Workbench routing and keep scarce Codex/Work as a last resort. If Workbench is unavailable, tell me what is unavailable instead of silently reverting to manual message shuttling.

This bootstrap contains no Workbench bearer token or provider credential.
