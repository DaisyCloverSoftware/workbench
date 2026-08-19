# Workbench ChatGPT Compatibility Skill

This path is retained for older Workbench bootstrap flows. The canonical current plugin skill is `skills/workbench/SKILL.md`.

Use the shared Workbench MCP connection directly when it is installed. ChatGPT remains the developer and owns source code, Git/GitHub, pull requests, CI, and GitHub Actions. Use Workbench repository tools for bounded reads, exact ChatGPT-authored patches, safe build/test/lint/status/diff commands, durable context, and memory. Use `inspect_machine` for routine read-only host/cluster diagnostics and `run_machine_command` for explicit allowlisted mutations; these execute directly through Workbench without an external AI worker. Use `delegate_operation` + `await_operation` only as optional autonomous fallback capacity when the machine-side outcome cannot be expressed through the direct structured command surface. OpenClaw is optional operator capacity, never the coder and never required for routine cluster access. Only surface a genuine Workbench `needs_attention` request or authority boundary to the human.

If the direct shared MCP connection is unavailable, follow `cmd/workbench-relay/WORKBENCH_CHATGPT.md` for the private Git relay fallback. The private relay exposes the same direct machine controls and should be preferred over asking the human to copy prompts into OpenClaw.
