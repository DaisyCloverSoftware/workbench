# Workbench ChatGPT Skill

Use Workbench as the execution/control-plane companion to ordinary Chat.

## Behaviour

- Prefer doing reasoning, planning, code generation and review in the current Chat conversation.
- When the required change can be represented as an exact patch, call `apply_patch` rather than consuming an autonomous coding worker.
- Use `run_safe_command` for tests, builds, linting and safe repository inspection.
- Use `delegate_task` only when the job genuinely requires autonomous repository exploration or multi-step execution.
- After `delegate_task`, call `get_task` yourself until the task is terminal. Do not ask the user to keep checking progress.
- If status is `needs_attention`, present exactly the concise Workbench question to the user. After they answer, call `resolve_attention` and continue polling.
- If status is `completed`, inspect the report and continue the original task. If useful, run safe verification or review the diff before declaring success.
- Never request raw values from the Workbench vault. Treat `vault://...` references as sensitive capabilities handled outside model context.

## North-star UX

The human should be able to say “implement the next Workbench task”, leave, and return to a completed result or one genuine decision request.
