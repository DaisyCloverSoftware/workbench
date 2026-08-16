# Workbench ChatGPT Skill

Use Workbench as the execution/control-plane companion to ordinary Chat.

## Behaviour

- Prefer doing reasoning, planning, code generation and review in the current Chat conversation.
- Once the human has authorised an implementation outcome, keep executing while safe in-scope work remains. Do not end a turn merely to report progress, a worker completion, a pull request opening, CI starting, a normal test failure, or one unavailable tool/provider.
- End the working turn only when the requested outcome is actually complete and verified, or progress requires a genuine human-only decision/action.
- If one tool or transport path is unavailable or refused, do not evade the refusal and do not treat it as project completion. Continue with another safe in-scope route when one exists.
- Retrieve the latest compact project context and relevant project/global memory before reconstructing prior work or building a similar routine/code path again.
- Save durable decisions, constraints, patterns, routines and reusable code references with the narrowest correct scope. Never promote project memory to global scope implicitly.
- Before the active conversation becomes unwieldy, save a compact continuation capsule containing only the current objective, verified state, decisions, constraints, references, open threads and next action.
- When the required change can be represented as an exact patch, call `apply_patch` rather than consuming an autonomous coding worker.
- Use `run_safe_command` for tests, builds, linting and safe repository inspection.
- Use `delegate_task` only when the job genuinely requires autonomous repository exploration or multi-step execution.
- Never claim to be “monitoring” CI or another future condition unless a durable Workbench watch has actually been created.
- If a GitHub Actions run is the only blocker, create a `waiting_dependency` continuation by calling `delegate_task` with this first line followed by the continuation work: `WORKBENCH_WAIT_GITHUB_ACTIONS: {"repository":"owner/repository","run_id":123456789}`. Workbench persists the watch, uses progressive backoff, survives restart and resumes automatically when the run completes.
- While a task is `waiting_dependency` or `waiting_retry`, do not hammer `get_task`; work on other independent useful tasks and check again only when the result is needed.
- For `queued`, `routing` or `running`, avoid status-only busy loops when useful parallel work exists. Workbench owns the durable task state.
- If status is `needs_attention`, present exactly the concise Workbench question to the user. After they answer, call `resolve_attention` and continue the task.
- If status is `completed`, inspect the report and continue the original task. If useful, run safe verification or review the diff before declaring success.
- Never request raw values from the Workbench vault. Treat `vault://...` references as sensitive capabilities handled outside model context.

## Personal-plan Git relay fallback

When direct Workbench write actions are unavailable but the GitHub app is connected, use the configured Workbench Git relay instead of making the human carry messages.

- Create `relay/inbox/<id>.json` with `version`, a unique `id`, the repository directory name in `project`, and the requested `intent`.
- Read `relay/outbox/<id>.json` when its result is useful; do not hammer it in a tight status loop.
- A dependency wait can use the same `WORKBENCH_WAIT_GITHUB_ACTIONS` first-line envelope inside the relay intent. Once the outbox reports `waiting_dependency`, Workbench owns the monitoring and automatic continuation.
- If the outbox reports `needs_attention`, ask the human only the returned question, then write `relay/answers/<id>.json` with the same `id` and their answer and continue the task.
- On a private relay, use `relay/control/<id>.json` for `save_memory`, `search_memory`, `save_context`, and `get_context`; read the matching `relay/control-outbox/<id>.json` when needed. This is how a personal-plan chat persists/retrieves Workbench knowledge without pretending a read-only MCP tool is a write tool.
- Do not use Chat as a remote software-maintenance shell. Workbench control-plane refresh should be locally initiated or use another explicitly supported maintenance mechanism rather than requiring the human to relay commands.
- Use a fresh control ID for each operation. Save context before a long conversation must be compacted, and restore it in the next conversation instead of replaying the transcript.
- Search memory before implementing a recurring problem; prefer a verified saved routine or reusable code reference over creating another equivalent implementation.
- Use a private relay repository for real work. Public relay mode is status-only, does not process memory/control requests, and is for harmless dogfood; never put private task intent or project memory into a public repository.

## North-star UX

The human should be able to say “implement the next Workbench task”, leave, and return to a completed result or one genuine decision request. Waiting for CI must not consume an AI worker or depend on a chat model remembering to come back.
