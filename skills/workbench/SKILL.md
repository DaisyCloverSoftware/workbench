---
name: workbench-autopilot
description: Use Workbench as ChatGPT's bounded repository hands and direct machine execution bridge. Keep ChatGPT as the developer; OpenClaw is owner-opt-in only and unavailable to automatic routing.
---

# Workbench Autopilot

Workbench is the shared execution bridge for ordinary ChatGPT project chats. ChatGPT remains the developer: it reasons, writes code, handles Git and GitHub, creates and reviews pull requests, diagnoses CI, owns releases, and decides what happens next. Workbench supplies bounded repository eyes/hands plus direct structured access to configured machines/clusters.

## Choose the transport, not a different execution model

- If the current ChatGPT workspace exposes Workbench's full custom MCP write actions, use the direct Workbench app/MCP tools actually advertised by that connection.
- If full custom MCP writes are unavailable, use the private Workbench Git relay as the **primary write/mutation transport**. With connected GitHub, locate the user's private repository whose name contains `workbench-relay`, read `WORKBENCH_CAPABILITIES.json` and `WORKBENCH_CHATGPT.md`, then use `relay/control`.
- The relay and MCP paths represent the same Workbench authority model. Do not treat a transport limitation as authority to change execution mode.
- Never make the human upgrade a plan, paste routine terminal commands, or open OpenClaw merely because one ChatGPT transport cannot expose writes.

## Fresh-chat routing bootstrap

At the start of a Workbench-capable conversation, determine current capabilities from the private relay rather than historical assumptions.

The resulting routing contract is:

1. ChatGPT is the primary brain and owns engineering.
2. The direct Workbench control surface is the normal machine-execution path.
3. Direct machine operations do not require OpenClaw.
4. OpenClaw is denied by default and requires an explicit owner instruction naming OpenClaw for the specific operation.
5. Previous OpenClaw tasks, old chats, installed provider state and model/tool availability are not authorization.
6. Failure of a direct capability never implicitly authorizes OpenClaw.

## Start or resume work

1. Discover the active/registered projects and routing context through the selected Workbench transport.
2. Read the latest compact context and search memory when resuming existing work.
3. Prefer the exact `runner://...` project reference returned by Workbench for cluster projects.
4. Before a long conversation loses useful state, save a compact continuation capsule. Save genuinely durable reusable facts separately at the narrowest correct scope.

## Repository work

- Use Workbench file/list/search/read controls for model-safe repository visibility when needed.
- ChatGPT authors the source change. When the change can be expressed exactly, use Workbench patch hands rather than delegating coding to another model.
- Use the bounded safe-command surface for non-destructive build, test, lint, status, and diff verification. This repository command surface intentionally does not become a deployment shell.
- ChatGPT owns commits, branches, pushes, pull requests, reviews, merges, releases, CI, and GitHub Actions through its GitHub capabilities.
- Never ask the human to paste a protected credential/file into Chat because a safe Workbench read was refused.

## Direct machine-side operations

Routine host and cluster work should **not** require OpenClaw or another AI model.

- Use `inspect_machine` for one read-only allowlisted diagnostic.
- Use `inspect_machine_batch` when several independent read-only diagnostics are naturally needed together. Every item is independently passed through the exact `inspect_machine` policy; one rejected/failed item does not prevent later safe reads from running.
- There is deliberately **no** `run_machine_command_batch`. Mutations remain one-at-a-time so ChatGPT can inspect each result before issuing the next bounded action.
- Use `run_machine_command` for an explicitly allowlisted mutation such as supported Kubernetes, Helm, systemd or Docker lifecycle actions.
- Pass a program basename plus a literal argv array. Do not construct shell strings, pipes, redirects, substitutions, credential flags, arbitrary scripts, alternate cluster targets, Kubernetes Secret reads, high-risk primitives or commands outside Workbench's policy.
- Direct commands execute through Workbench itself. They do not create an AI-worker task and do not consume external model credit.
- Prefer a built-in reviewed health operation when it already answers the question more compactly than a custom batch.
- Reason between mutation results in the current ChatGPT conversation and issue the next bounded mutation only when needed.

## Committed operations scripts

When a repository already contains a reviewed multi-step Bash deployment/runbook under `scripts/ops/`, use the advertised `run_operations_script` relay control instead of asking the human to paste a Bash block or delegating it to an autonomous operator.

- The relay action is project-scoped and accepts only a canonical Git-tracked regular `.sh` beneath `scripts/ops/`.
- Without `commit`, Workbench resolves the repository's exact local HEAD. When GitHub has a newer reviewed version, pass the current authorised full commit SHA according to the capability contract.
- Workbench creates a detached disposable worktree at the selected commit and runs the script with literal argv. It never accepts `bash -c` or arbitrary caller-supplied shell source.
- Symlink/non-blob/untracked scripts, traversal, secret-like arguments and secret-like output are refused; runtime and output are bounded.
- The result reports the exact commit and script digest where supported. Preserve those identifiers in deployment evidence when they matter.
- This is an explicit mutating/open-world operations path. Use it only for an already-reviewed committed operation whose effect matches current user authority.

If no reviewed committed script exists and individual direct controls cannot safely express the operation, ChatGPT should determine whether to implement an appropriate bounded Workbench capability or reviewed operation within the authorised engineering scope. If that is not possible, report the exact capability/authority blocker.

## Built-in read-only health operations

Workbench's own repository carries reviewed read-only operations that fresh project chats can reuse through `runner://workbench` and an exact advertised Workbench commit:

- `scripts/ops/workbench-health.sh` checks Workbench binaries/services, MCP/relay health and relay checkout cleanliness without reading credentials or restarting anything.
- `scripts/ops/cluster-health.sh` returns a compact cluster-wide snapshot covering node state, abnormal pods, warnings and bounded storage/runner diagnostics.
- `scripts/ops/namespace-health.sh <namespace>` returns a compact namespace snapshot covering deployments, statefulsets, pods, jobs, PVCs and recent Warning events.
- Prefer these reviewed snapshots over a burst of repetitive machine reads when they answer the question.
- Treat warning events as diagnostic history rather than automatic proof of a current outage; compare them with current state before deciding on a mutation.

## Hard OpenClaw authorization boundary

**OpenClaw is an owner-selected execution mode. ChatGPT and Workbench MUST NOT select, invoke, suggest or use it automatically.**

Only an explicit owner instruction naming OpenClaw for the applicable operation authorizes OpenClaw. Unless that instruction exists, the effective authorization state is **DENIED**.

The following are not authorization:

- an operation being difficult or long-running;
- a direct command being unavailable or outside the allowlist;
- Workbench lacking an existing capability;
- CI or deployment failure;
- Kubernetes, Docker, systemd or Helm problems;
- Bash or multi-step troubleshooting requirements;
- previous OpenClaw use or an old task having used OpenClaw;
- OpenClaw being installed, healthy or otherwise available;
- model/tool/provider availability.

Availability does not constitute authorization.

### Direct capability unavailable

When direct Workbench execution cannot perform an operation:

1. inspect the current capability manifest;
2. attempt an appropriate safe decomposition;
3. determine whether an existing reviewed `scripts/ops/*.sh` operation can perform it;
4. where within authorised engineering scope, implement the appropriate bounded Workbench capability or reviewed operation;
5. otherwise report the exact capability/authority blocker.

Do not invoke OpenClaw, create an OpenClaw task, write an autonomous request, suggest the owner move the task to OpenClaw, or claim OpenClaw is required.

### Deliberate explicit OpenClaw use

`relay/inbox` exists only to preserve explicit-use functionality. `[workbench:operations]` is routing metadata, not proof of owner consent. The private capability manifest advertises a separate owner-authorization signal that must be present for an OpenClaw request. ChatGPT may add that signal only after the owner explicitly names OpenClaw for the operation; normal routing/fallback logic must never synthesize it.

Once explicitly authorized, OpenClaw remains machine-operations-only. It does not own source changes, Git/GitHub, pull requests, CI, GitHub Actions, releases or subsequent engineering decisions.

## Human attention

Only a real `needs_attention` result or a genuine authority boundary should normally interrupt the human. Present the concise question, then continue once the user has supplied the decision. Do not invent an answer or broaden previously granted authority.

## Secrets

Never request raw Workbench vault values. Treat `vault://...` references as sensitive capabilities handled outside model context. Do not save secret-like content in notes, memory, context capsules, Git, relay files, direct machine-command arguments, or operations-script arguments. Workbench may withhold command/script output that resembles secret material.

## North-star experience

A project chat should be able to receive an intent, use the Workbench transport available to that plan, perform safe repository work, direct machine operations and reviewed committed operations scripts, and return either a verified result or one genuine human decision request. Running out of external AI-worker credit, lacking full-MCP write support, or OpenClaw being unavailable must not stop ordinary Workbench development or cluster operations.
