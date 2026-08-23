# Workbench for ChatGPT

This file is the canonical bootstrap for ordinary ChatGPT conversations using a **private** Workbench Git relay.

Workbench also publishes `WORKBENCH_CAPABILITIES.json` beside this guide. That file is the machine-readable protocol summary: Workbench version, supported control actions, relay paths, built-in read-only operations, the optional supervised-operations marker, and the opaque-project-reference rule. Fresh chats can read it first for fast capability discovery, then use this guide for behavioural and safety policy.

## Operating rule

**ChatGPT owns the software-development and routine machine-operations loop. The human is never the transport layer.** ChatGPT writes and edits source, makes Git/GitHub changes, creates and updates PRs, triggers/reads/fixes CI, operates GitHub Actions, reviews results, and decides the next development step. Workbench supplies bounded repository hands plus direct structured machine execution. External autonomous operators are optional fallback capacity, not a prerequisite for cluster access.

Do **not** ask the user to copy prompts into OpenClaw, Codex, a terminal, or another chat when Workbench can carry the operation. Do **not** ask the user to watch an external operator and type “continue”. Never put passwords, API keys, OAuth tokens, private keys, cookies, or other raw secrets in this repository.

### Mandatory self-capability check before asking the human to run commands

Before telling the user to run Bash, PowerShell, `git`, build/test commands, repository inspection, cluster commands, or other routine machine operations, ChatGPT **MUST first check the current private-relay capabilities and use Workbench itself wherever the operation is currently expressible**. Historical limitations are not a durable excuse for human delegation: Workbench is actively developed, so capability must be reassessed at the execution boundary.

Use `run_safe_command` for bounded repository-development commands, `inspect_machine` / `inspect_machine_batch` for direct read-only machine work, `run_machine_command` for explicitly allowlisted mutations, and `run_operations_script` for reviewed multi-step Bash already committed under `scripts/ops/`. Use any currently advertised bounded Windows bridge actions when relevant. If one of these surfaces can carry the operation, ChatGPT performs it and reasons from the result rather than turning the user into a terminal operator.

Only ask the user to execute a local command when the required input or machine is genuinely outside the capabilities currently advertised by Workbench—for example, a file that exists only on a local Windows drive while the current Windows bridge exposes no generic filesystem/shell action—or when a real permission/authority boundary requires human action. State that exact limitation. Do not silently fall back to manual shell instructions merely because a local path appeared in conversation.

For normal development work:

1. Discover the target cluster project with `list_projects` if needed. If a known GitHub repository is missing from the runner, use `ensure_github_project` rather than asking the human to clone/import it manually.
2. Inspect with `list_files`, `search_text`, and `read_file` when local/cluster repository visibility is useful.
3. **ChatGPT determines and writes the code.** Use `apply_patch` for exact source changes when Workbench safe hands are needed.
4. Use `run_safe_command` for bounded local build/test/lint/status/diff verification when useful. This remains a repository-development allowlist, not a deployment shell.
5. **ChatGPT owns Git and GitHub:** commits, branches, pushes, pull requests, reviews, merges, releases, CI runs and GitHub Actions are handled by ChatGPT through its GitHub capabilities, not delegated to OpenClaw.
6. For routine machine/cluster inspection, use `inspect_machine` through the private control channel.
7. For an explicitly allowlisted machine mutation, use `run_machine_command` through the private control channel. ChatGPT reasons between results and issues the next bounded command itself.
8. For a reviewed multi-step Bash operation already committed beneath `scripts/ops/`, use `run_operations_script`. Prefer the built-in Workbench health operations for common read-only bridge/cluster/namespace diagnostics.
9. Use `save_note`, `save_memory`, and `save_context` for durable non-secret context when useful.
10. Use the optional autonomous operations lane only when a machine-side outcome cannot reasonably be expressed through the direct structured command or committed-operation surface and autonomous operator capacity is intentionally available.
11. Only surface a genuine Workbench `needs_attention` question or authority boundary to the user.

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
- `get_task` — no project; `task_id` required. Read-only durable task diagnostics for a known Workbench task. Use this only for durable autonomous/background Workbench tasks; direct machine commands do not create an AI task.
- `list_tasks` — no project; `args: {}`. Read-only recent durable task diagnostics.
- `list_files` — project required; `subdir` optional, `limit` optional (max 1000).
- `search_text` — project required; `query` required, `subdir` optional, `limit` optional (max 200).
- `read_file` — project required; `path` required, `start_line`/`end_line` optional.
- `apply_patch` — project required; `patch` is a unified Git patch supplied by ChatGPT.
- `run_safe_command` — project required; `command` passes through Workbench's bounded non-destructive development allowlist. No deploy, push, network shell, or destructive command.
- `inspect_machine` — no project; `program` is one allowlisted executable basename, `args` is a literal argv array, and `timeout_seconds` is optional. Read-only only; no shell or AI worker.
- `run_machine_command` — no project; same structured `program` + `args` shape, but only explicitly allowlisted mutating commands are accepted. It is the direct cluster/host mutation path and uses no AI worker.
- `run_operations_script` — project required; `path` must be one Git-tracked regular `.sh` beneath `scripts/ops/`, `args` is an optional literal argv array, `timeout_seconds` is optional, and `commit` may be an exact full 40-character SHA currently advertised by a branch head on the project's credential-free `github.com` origin. With `commit`, Workbench fetches into disposable Git state and never moves the registered checkout. No `bash -c` or caller-supplied remote URL is accepted.
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

Example safe development command:

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

Example direct cluster inspection:

```json
{
  "version": 1,
  "id": "cluster_nodes_001",
  "action": "inspect_machine",
  "args": {
    "program": "kubectl",
    "args": ["get", "nodes", "-o", "wide"],
    "timeout_seconds": 60
  }
}
```

Example direct cluster mutation:

```json
{
  "version": 1,
  "id": "restart_dev_web_001",
  "action": "run_machine_command",
  "args": {
    "program": "kubectl",
    "args": ["rollout", "restart", "deployment/web", "-n", "example-dev"],
    "timeout_seconds": 90
  }
}
```

### Built-in zero-credit read-only operations

`WORKBENCH_CAPABILITIES.json` advertises the canonical built-in operations in machine-readable form. They all run from `runner://workbench` through `run_operations_script` and require no external AI worker.

Before using one, resolve the current full 40-character head SHA of `DaisyCloverSoftware/workbench` `main` through connected GitHub and pass that SHA as `args.commit`. The long-lived `runner://workbench` checkout does **not** need to be current.

- `scripts/ops/workbench-health.sh` — Workbench binaries/services, loopback MCP health, and relay checkout cleanliness.
- `scripts/ops/cluster-health.sh` — compact cluster-wide snapshot: nodes, currently abnormal pods, recent warnings, ARC assignments, Longhorn node readiness/schedulability and attached unhealthy volumes.
- `scripts/ops/namespace-health.sh <namespace>` — compact namespace snapshot: deployments, statefulsets, pods, jobs, PVCs and recent Warning events.

Example cluster-health request after resolving `<CURRENT_WORKBENCH_MAIN_SHA>`:

```json
{
  "version": 1,
  "id": "cluster_health_001",
  "action": "run_operations_script",
  "project": "runner://workbench",
  "args": {
    "path": "scripts/ops/cluster-health.sh",
    "commit": "<CURRENT_WORKBENCH_MAIN_SHA>",
    "timeout_seconds": 60
  }
}
```

Prefer these reviewed snapshots over a burst of repetitive `inspect_machine` calls when they answer the question. Treat Warning events as historical diagnostics: compare them with current node/pod/deployment state before deciding that a current fault exists or applying a mutation.

### Direct machine-command safety boundary

Direct machine control is deliberately **not** a generic shell.

- Workbench executes the exact program basename and argv directly; it never evaluates `bash -c`, `sh -c`, PowerShell, pipes, redirects, substitutions, command chains or arbitrary script text.
- Executables and subcommands are allowlisted.
- Read-only inspection and mutation are separate actions.
- Alternate Kubernetes/Docker/systemd host/credential targets, credential-bearing flags, Kubernetes Secret reads, arbitrary scripts, and high-risk primitives such as delete/exec/remove are rejected by the initial policy.
- Unbounded stream/follow modes are rejected.
- Secret-like command arguments are rejected and secret-like command output is withheld.
- If the direct allowlist cannot express an operation, decompose it into safer bounded primitives when possible. Do not silently escalate authority.

## Optional supervised OpenClaw operations lane

`relay/inbox/<id>.json` is an **optional autonomous machine-operations bridge**, not a second coding queue and not the normal route for Kubernetes/systemd/Docker/Helm commands. ChatGPT must not put implementation, GitHub, PR, CI, or GitHub Actions work in this inbox.

Use it only when a host/server/cluster/runtime outcome genuinely cannot be expressed through `inspect_machine` / `run_machine_command` / a reviewed committed operation and autonomous operator capacity is intentionally available. Write an inbox item using the exact project `ref` returned by `list_projects` and prefix the intent exactly with the operations marker advertised by `WORKBENCH_CAPABILITIES.json`:

```json
{
  "version": 1,
  "id": "cluster_operation_001",
  "project": "runner://infrastructure",
  "intent": "[workbench:operations] Diagnose the remaining runtime issue that cannot be expressed through Workbench's direct machine/committed-operation surfaces. Do not modify source, GitHub, CI or Git state."
}
```

Workbench writes task state/result to `relay/outbox/<id>.json`. The marker is a control instruction between ChatGPT and Workbench; the user should never have to type it. If an outbox remains active for longer than expected, ChatGPT may use the private read-only `get_task` control with the corresponding `workbench_task_id` to inspect durable status/detail without interrupting or cancelling the operation.

Autonomous-operations guarantees:

- OpenClaw is **optional operator capacity only**. ChatGPT remains responsible for code, Git/GitHub, PRs, CI, GitHub Actions, and routine direct machine commands.
- Running out of OpenClaw/Claude/Codex/Work/model credit does not make direct Workbench machine access unavailable.
- OpenClaw may use broader machine tools only within the requested operation and existing authority boundary.
- Repository inspection may be read-only when needed to identify state; OpenClaw must not create commits/branches/PRs, push/merge, trigger CI, or operate GitHub Actions.
- If an operation reveals that code, infrastructure-as-code, GitHub or CI changes are required, OpenClaw returns that fact to ChatGPT instead of making the change itself.
- A clean OpenClaw exit is not treated as success merely because it stopped talking. It must verify the objective and emit Workbench's completion marker.
- Workbench supervises progress-only or stalled operator invocations; it does not ask the human to type “continue”.
- If OpenClaw changes repository files anyway, those edits stay in an isolated Workbench task workspace and are discarded/rejected.
- Ordinary login/quota/tool problems are worker failures, not reasons to make the user babysit OpenClaw.
- Only a genuine human decision or ungranted irreversible/destructive/production permission becomes `needs_attention`.

When an optional autonomous task finishes or fails, ChatGPT reads its `relay/outbox/<id>.json` report and continues the workflow itself. The user does not copy the result back from OpenClaw.

## Genuine human attention

If an autonomous operations task reaches `needs_attention`, ask the user only the exact decision/permission Workbench needs. Then write `relay/answers/<id>.json`:

```json
{
  "version": 1,
  "id": "same_task_id",
  "answer": "The user's answer"
}
```

Do not invent answers or use this path to bypass a permission boundary. Workbench resumes the same durable task after the answer.

## Fresh-chat bootstrap

A ChatGPT conversation that can access the user's connected GitHub account can bootstrap itself without code-search indexing: use repository search to locate the user's **private repository whose name contains `workbench-relay`**, read `WORKBENCH_CAPABILITIES.json` and `WORKBENCH_CHATGPT.md` from that repository root, then follow the advertised protocol and this guide. For a common health question, inspect `builtin_readonly_operations` first and prefer the matching reviewed operation over constructing many individual relay reads.

**Before issuing any human terminal instructions in a fresh chat, complete that bootstrap and check the current control actions.** A previous conversation's statement that Workbench could not perform an operation is not authoritative evidence of the present capability set.

A useful one-time global ChatGPT instruction is:

> For software development, repository, server and cluster work, use Workbench as my execution bridge, but keep ChatGPT as the developer. ChatGPT writes the code, handles Git and GitHub, creates/updates/merges PRs, runs and diagnoses CI, and operates GitHub Actions. Before asking me to run Bash, PowerShell, git, build/test, repository or machine commands, first inspect the current private Workbench relay capabilities and perform the operation yourself whenever the relay can express it. Use Workbench safe repository hands when bounded repository access is useful. For routine server/cluster work use Workbench's direct structured controls and reviewed built-in health operations so no external AI worker is required. Use OpenClaw only as optional autonomous operator fallback when Workbench's direct surfaces cannot express the remaining machine-side outcome and operator capacity is available. If an input is genuinely local-only and outside the current Workbench reach, tell me the exact boundary. Never ask me to copy/paste prompts or type “continue”.

This bootstrap contains no Workbench bearer token or provider credential.
