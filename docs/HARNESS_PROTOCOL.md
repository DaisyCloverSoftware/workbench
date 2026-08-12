# Harness Adapter Protocol — v0.4 draft

Workbench is harness-agnostic. OpenClaw is an adapter, not the architecture.

## Preferred path: Workbench Runner over SSH

The first structured runner transport uses the developer's existing SSH trust rather than introducing another public daemon immediately.

Desktop Workbench sends a JSON `RunnerRequest` on stdin to:

```text
ssh <host> $HOME/.local/bin/workbench-runner run
```

The request contains the task plus routing policy (`avoid_work_usage`, `allow_metered_api`). The cluster-side runner resolves the repository under its configured runner root, scans the workers actually installed on that machine, and applies Workbench's cost-aware routing policy again.

The response is structured JSON containing:

- worker result/output;
- any genuine `ATTENTION_REQUIRED` question;
- provider ID/name;
- provider cost class;
- attempt history;
- a concise error when every eligible worker fails.

Prompts are transported through stdin rather than shell arguments so large tasks and arbitrary source text do not depend on fragile shell escaping.

## Repository mapping

The default runner root is:

```text
~/src
```

Override it with `WORKBENCH_RUNNER_ROOT`.

If a task arrives with a desktop path such as:

```text
C:\workspace\workbench
```

and that exact path does not exist on the Linux runner, the runner maps the final directory name to:

```text
~/src/workbench
```

The resolved project must remain inside the runner root. This prevents a remote task from escaping into arbitrary host directories.

## Legacy command-template adapter

v0.3 introduced a deliberately small bridge where the user may configure a command template containing:

- `{project}` — project path
- `{prompt}` — Workbench worker instructions

A harness can request human attention by ending its report with:

```text
ATTENTION_REQUIRED: <one concise question>
```

This remains as an advanced compatibility adapter, but the structured Workbench Runner is now the preferred path.

## Next protocol steps

The SSH request/response schema is transport-neutral. Planned additions include durable remote task IDs, cancellation, progress events, artefact transfer, resumable human input, HTTPS/MCP transport and worker capability advertisement.
