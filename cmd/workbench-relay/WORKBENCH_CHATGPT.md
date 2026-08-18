# Workbench for ChatGPT

This file is the canonical bootstrap for ordinary ChatGPT conversations using a **private** Workbench Git relay.

Workbench also publishes `WORKBENCH_CAPABILITIES.json` beside this guide. That file is the machine-readable protocol summary: Workbench version, supported control actions, relay paths, the supervised-operations marker, and the opaque-project-reference rule. Fresh chats can read it first for fast capability discovery, then use this guide for behavioural and safety policy.

## Operating rule

**ChatGPT owns the software-development loop. The human is never the transport layer.** ChatGPT writes and edits source, makes Git/GitHub changes, creates and updates PRs, triggers/reads/fixes CI, operates GitHub Actions, reviews results, and decides the next development step. Workbench supplies bounded repository hands and the bridge to machine-side operations; it must not replace ChatGPT as the developer.

Do **not** ask the user to copy prompts into OpenClaw, Codex, a terminal, or another chat when Workbench can carry the operation. Do **not** ask the user to watch OpenClaw and type “continue” when it stops. Never put passwords, API keys, OAuth tokens, private keys, cookies, or other raw secrets in this repository.

For normal development work:

1. Discover the target cluster project with `list_projects` if needed. If a known GitHub repository is missing from the runner, use `ensure_github_project` rather than asking the human to clone/import it manually.
2. Inspect with `list_files`, `search_text`, and `read_file` when local/cluster repository visibility is useful.
3. **ChatGPT determines and writes the code.** Use `apply_patch` for exact source changes when Workbench safe hands are needed.
4. Use `run_safe_command` for bounded local build/test/lint/status/diff verification when useful.
5. **ChatGPT owns Git and GitHub:** commits, branches, pushes, pull requests, reviews, merges, releases, CI runs and GitHub Actions are handled by ChatGPT through its GitHub capabilities, not delegated to OpenClaw.
6. Use `save_note`, `save_memory`, and `save_context` for durable non-secret context when useful.
7. Only when the remaining step genuinely requires machine-side access ChatGPT does not have — for example shell, systemd, Docker, Kubernetes, Helm, runner repair or deployment/runtime commands — use the supervised OpenClaw operations lane below.
8. Only surface a Workbench `needs_attention` question to the user.

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
- `get_task` — no project; `task_id` required. Read-only durable task diagnostics for a known Workbench task. Use this when an operation is taking unusually long or its relay outbox has not changed; it does not cancel, resume or otherwise mutate the task.
- `list_tasks` — no project; `args: {}`. Read-only recent task diagnostics. This is for troubleshooting/continuation context, not a second scheduling interface.
- `list_files` — project required; `subdir` optional, `limit` optional (max 1000).
- `search_text` — project required; `query` required, `subdir` optional, `limit` optional (max 200).
- `read_file` — project required; `path` required, `start_line`/`end_line` optional.
- `apply_patch` — project required; `patch` is a unified Git patch supplied by ChatGPT.
- `run_safe_command` — project required; `command` passes through Workbench's bounded non-destructive development allowlist. No deploy, push, network shell, or destructive command.
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

## Supervised OpenClaw operations lane

`relay/inbox/<id>.json` is the **machine-operations bridge**, not a second coding queue. ChatGPT must not put implementation, GitHub, PR, CI, or GitHub Actions work in this inbox.

For a host/server/cluster/runtime operation that ChatGPT genuinely cannot execute through its own tools or Workbench safe hands, write an inbox item using the exact project `ref` returned by `list_projects` and prefix the intent exactly with the operations marker advertised by `WORKBENCH_CAPABILITIES.json`:

```json
{
  "version": 1,
  "id": "cluster_operation_001",
  "project": "runner://infrastructure",
  "intent": "[workbench:operations] Deploy the already-built DEV image, verify the Kubernetes rollout and report the running image digest. Do not modify source, GitHub, CI or Git state."
}
```

Workbench writes task state/result to `relay/outbox/<id>.json`. The marker is a control instruction between ChatGPT and Workbench; the user should never have to type it. If an outbox remains active for longer than expected, ChatGPT may use the private read-only `get_task` control with the corresponding `workbench_task_id` to inspect durable status/detail without interrupting or cancelling the operation.

Operations-lane guarantees:

- OpenClaw is an **operator only**. ChatGPT remains responsible for code, Git/GitHub, PRs, CI and GitHub Actions.
- OpenClaw may use shell/systemd/Docker/Kubernetes/Helm and equivalent runtime/host tools needed for the requested operation.
- Repository inspection may be read-only when needed to identify state; OpenClaw must not create commits/branches/PRs, push/merge, trigger CI, or operate GitHub Actions.
- If an operation reveals that code, infrastructure-as-code, GitHub or CI changes are required, OpenClaw returns that fact to ChatGPT instead of making the change itself.
- A clean OpenClaw exit is **not** treated as success merely because it stopped talking. It must verify the objective and emit Workbench's completion marker.
- If OpenClaw returns a progress-only response, Workbench automatically invokes it again against the current host/cluster state. Bounded unresponsive invocations are also re-engaged automatically.
- Workbench performs up to six supervised continuation passes before reporting a worker failure back to ChatGPT. It does not ask the human to type “continue”.
- If OpenClaw changes repository files anyway, those edits stay in an isolated Workbench task workspace and are discarded/rejected.
- Ordinary login/quota/tool problems are worker failures for ChatGPT/Workbench to handle, not reasons to make the user babysit OpenClaw.
- Only a genuine human decision or ungranted irreversible/destructive/production permission becomes `needs_attention`.

When an operations task finishes or fails, ChatGPT reads its `relay/outbox/<id>.json` report and continues the development/release workflow itself. The user does not copy the result back from OpenClaw.

## Genuine human attention

If an operations task reaches `needs_attention`, ask the user only the exact decision/permission Workbench needs. Then write `relay/answers/<id>.json`:

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

> For software development, repository, server and cluster work, use Workbench as my execution bridge, but keep ChatGPT as the developer. ChatGPT writes the code, handles Git and GitHub, creates/updates/merges PRs, runs and diagnoses CI, and operates GitHub Actions. Use Workbench safe hands when bounded repository access is useful. Only pass to Workbench/OpenClaw the machine-side work ChatGPT genuinely cannot execute itself, such as shell, systemd, Docker, Kubernetes, Helm, runner repair and deployment/runtime commands. OpenClaw is the operator, never the coder. Workbench must automatically keep OpenClaw moving until the requested operation is verified; never ask me to copy/paste prompts or type “continue”. If the operation discovers that code, GitHub or CI changes are required, return that to ChatGPT to handle.

This bootstrap contains no Workbench bearer token or provider credential.
