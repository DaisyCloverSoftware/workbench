# Workbench ChatGPT Skill

Use Workbench as the execution/control-plane companion to ordinary Chat.

## Behaviour

- Prefer doing reasoning, planning, code generation and review in the current Chat conversation.
- Retrieve the latest compact project context and relevant project/global memory before reconstructing prior work or building a similar routine/code path again.
- Save durable decisions, constraints, patterns, routines and reusable code references with the narrowest correct scope. Never promote project memory to global scope implicitly.
- Before the active conversation becomes unwieldy, save a compact continuation capsule containing only the current objective, verified state, decisions, constraints, references, open threads and next action.
- When the required change can be represented as an exact patch, call `apply_patch` rather than consuming an autonomous coding worker.
- Use `run_safe_command` for tests, builds, linting and safe repository inspection.
- Use `delegate_task` only when the job genuinely requires autonomous repository exploration or multi-step execution.
- After `delegate_task`, call `get_task` yourself until the task is terminal. Do not ask the user to keep checking progress.
- If status is `needs_attention`, present exactly the concise Workbench question to the user. After they answer, call `resolve_attention` and continue polling.
- If status is `completed`, inspect the report and continue the original task. If useful, run safe verification or review the diff before declaring success.
- Never request raw values from the Workbench vault. Treat `vault://...` references as sensitive capabilities handled outside model context.

## Personal-plan Git relay fallback

When direct Workbench write actions are unavailable but the GitHub app is connected, use the configured Workbench Git relay instead of making the human carry messages.

- Create `relay/inbox/<id>.json` with `version`, a unique `id`, the repository directory name in `project`, and the requested `intent`.
- Poll `relay/outbox/<id>.json` yourself until terminal.
- If the outbox reports `needs_attention`, ask the human only the returned question, then write `relay/answers/<id>.json` with the same `id` and their answer and resume polling.
- On a private relay, use `relay/control/<id>.json` for `save_memory`, `search_memory`, `save_context`, and `get_context`; poll the matching `relay/control-outbox/<id>.json` yourself. This is how a personal-plan chat persists/retrieves Workbench knowledge without pretending a read-only MCP tool is a write tool.
- Use a fresh control ID for each operation. Save context before a long conversation must be compacted, and restore it in the next conversation instead of replaying the transcript.
- Search memory before implementing a recurring problem; prefer a verified saved routine or reusable code reference over creating another equivalent implementation.
- Use a private relay repository for real work. Public relay mode is status-only, does not process memory/control requests, and is for harmless dogfood; never put private task intent or project memory into a public repository.

## North-star UX

The human should be able to say “implement the next Workbench task”, leave, and return to a completed result or one genuine decision request.
