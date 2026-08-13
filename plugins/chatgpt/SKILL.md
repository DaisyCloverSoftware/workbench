# Workbench ChatGPT Skill

Use Workbench as the execution/control-plane companion to ordinary Chat.

## Behaviour

- Treat conversation history as working context, not the project database.
- Start/resume with `get_workspace` and `get_context_pack` when available.
- Check `find_routines` before rebuilding a procedure or code pattern that may already exist.
- Prefer doing reasoning, planning, code generation and review in the current Chat conversation.
- When the required change can be represented as an exact patch, call `apply_patch` rather than consuming an autonomous coding worker.
- Use `run_safe_command` for tests, builds, linting and safe repository inspection.
- Use `delegate_task` only when the job genuinely requires autonomous repository exploration or multi-step execution. Workbench attaches relevant durable context automatically.
- After `delegate_task`, call `get_task` yourself until the task is terminal. Do not ask the user to keep checking progress.
- If status is `needs_attention`, present exactly the concise Workbench question to the user. After they answer, call `resolve_attention` and continue polling.
- If status is `completed`, inspect the report and continue the original task. Persist only genuinely durable lessons, decisions or routines rather than the whole transcript.
- Use `save_checkpoint` at meaningful milestones or before the working conversation becomes unwieldy. Include a compact summary, durable decisions, open loops and next actions.
- Use `save_routine` to update/store a proven reusable procedure or code template so similar work is not rebuilt repeatedly.
- Never request raw values from the Workbench vault. Model-facing memory/routine writes must not contain secrets.

## Personal-plan Git relay fallback

When direct Workbench write actions are unavailable but the GitHub app is connected, use the configured Workbench Git relay instead of making the human carry messages.

- Create `relay/inbox/<id>.json` with `version`, a unique `id`, the repository directory name in `project`, and the requested `intent`.
- Poll `relay/outbox/<id>.json` yourself until terminal.
- If the outbox reports `needs_attention`, ask the human only the returned question, then write `relay/answers/<id>.json` with the same `id` and their answer and resume polling.
- Use a private relay repository for real work. Public relay mode is status-only and is for harmless dogfood; never put private task intent into a public repository.
- Do not leak compact memory/checkpoint content through a public relay merely because direct memory writes are unavailable. Wait for the supported private memory-control transport.

## North-star UX

The human should be able to say “implement the next Workbench task”, leave, and return to a completed result or one genuine decision request — and a fresh conversation should be able to resume the project without replaying the old chat.
