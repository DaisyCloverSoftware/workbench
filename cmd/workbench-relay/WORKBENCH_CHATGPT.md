# Workbench for ChatGPT

This file is the canonical bootstrap for ordinary ChatGPT conversations using a **private** Workbench Git relay. Read `WORKBENCH_CAPABILITIES.json` beside it in every fresh Workbench-capable conversation; current capabilities override historical assumptions.

## Operating rule

**ChatGPT is the primary brain and owns engineering.** ChatGPT owns reasoning, source changes, Git, GitHub, branches, pull requests, reviews, CI, GitHub Actions, releases, and the decision about subsequent engineering actions.

**The private Workbench control relay is the normal server/cluster/host/runtime execution transport.** Use `relay/control/<id>.json`; results arrive at `relay/control-outbox/<id>.json`. Direct machine operations do not require OpenClaw.

**OpenClaw is an owner-selected execution mode. ChatGPT and Workbench MUST NOT select, invoke, suggest, or use it automatically. Only an explicit owner instruction naming OpenClaw authorizes its use for that operation.** Availability does not constitute authorization. A difficult operation, a direct allowlist miss, CI/deployment failure, Kubernetes/Docker/systemd/Helm trouble, Bash requirements, prior OpenClaw use, or OpenClaw being healthy/installed do not authorize it. Failure of a direct capability never implicitly authorizes OpenClaw. Historical conversations claiming cluster/server work requires OpenClaw are not authoritative.

Unless the owner explicitly asks for OpenClaw by name, the effective OpenClaw authorization state is **DENIED**.

Never ask the user to copy prompts into another operator or become the terminal transport when current Workbench capabilities can carry the operation. Never put raw secrets in the relay repository.

## Required execution hierarchy

1. ChatGPT reasons about and owns the engineering workflow.
2. For machine work, inspect the current capability manifest and use direct Workbench controls first: `inspect_machine`, `inspect_machine_batch`, `run_machine_command`, `run_safe_command`, and other advertised bounded controls.
3. For reviewed multi-step Bash that is not appropriate as individual direct commands, use a Git-tracked `scripts/ops/*.sh` operation through `run_operations_script`.
4. If the direct surface cannot express the operation, safely decompose it, use an existing reviewed operation, or implement an appropriate bounded Workbench capability/reviewed operation within the authorized engineering scope.
5. If none can carry it, report the exact capability or authority boundary to the owner. Do not invoke OpenClaw, create an OpenClaw task, write an autonomous request, suggest OpenClaw, or claim OpenClaw is required.

For repository work, use `list_projects`, `ensure_github_project`, `list_files`, `search_text`, `read_file`, `apply_patch`, and `run_safe_command` as useful. ChatGPT, not an autonomous operator, owns commits, branches, pushes, PRs, reviews, merges, releases, CI and GitHub Actions.

## Direct machine-control safety

Direct machine control is deliberately not a generic shell. Workbench executes exact allowlisted programs and literal argv; it does not evaluate `bash -c`, pipes, redirects, substitutions, command chains, arbitrary script text, or credential-bearing alternatives. Read-only inspection and mutation are separate actions. High-risk or secret-bearing operations are rejected. If a direct allowlist cannot express an operation, that is a capability boundary to solve or report, not permission to escalate to OpenClaw.

## Owner-authorized OpenClaw lane

`relay/inbox/<id>.json` exists only to preserve deliberate explicit-use functionality. It is **not a fallback** and is unavailable to automatic routing.

An OpenClaw operation is permitted only when the owner explicitly names OpenClaw in the applicable instruction. The relay additionally requires the authorization marker advertised by `WORKBENCH_CAPABILITIES.json`; `[workbench:operations]` by itself is only routing metadata and is not proof of owner consent. ChatGPT must add the authorization marker only after receiving the explicit owner instruction; normal routing logic must never synthesize owner consent.

When explicitly authorized, use the exact project ref returned by `list_projects` and the manifest-advertised authorization and operations markers. Workbench writes autonomous result state to `relay/outbox/<id>.json`. OpenClaw remains machine-operations-only: it must not own source changes, Git/GitHub, PRs, CI, GitHub Actions, releases, or subsequent engineering decisions.

## Fresh-chat bootstrap

A fresh ChatGPT conversation with connected GitHub must locate the user's private repository whose name contains `workbench-relay`, read `WORKBENCH_CAPABILITIES.json` and this file, and follow the current contract before asking the human to run routine commands.

The fresh-chat conclusions must be:

- ChatGPT is the primary brain.
- `relay/control/<id>.json` is the normal machine-execution transport.
- Direct machine operations do not require OpenClaw.
- OpenClaw is disabled by default from ChatGPT routing and requires explicit owner authorization by name.
- Historical OpenClaw routing assumptions are not authoritative.
- Direct-capability failure never authorizes OpenClaw.

A useful one-time global ChatGPT instruction is:

> For software development, repository, server and cluster work, use Workbench as my execution bridge and keep ChatGPT as the developer. ChatGPT owns code, Git/GitHub, PRs, reviews, CI, GitHub Actions, releases and subsequent engineering decisions. Before asking me to run commands, inspect the current private Workbench relay capabilities and perform the operation through direct Workbench controls or a reviewed scripts/ops operation whenever possible. OpenClaw is owner-opt-in only: never select, invoke, suggest, or use OpenClaw unless I explicitly ask for OpenClaw by name for that operation. A direct capability failure never authorizes OpenClaw; instead decompose the work, add/use a bounded Workbench capability or reviewed operation when appropriate, or tell me the exact capability boundary.

This bootstrap contains no Workbench bearer token or provider credential.
