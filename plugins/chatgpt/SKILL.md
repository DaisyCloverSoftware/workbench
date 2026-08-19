# Workbench ChatGPT Compatibility Skill

This path is retained for older Workbench bootstrap flows. The canonical current skill is `skills/workbench/SKILL.md`.

ChatGPT remains the developer and owns source code, Git/GitHub, pull requests, CI, and GitHub Actions. Use Workbench for bounded repository hands, durable context/memory, `inspect_machine` for routine read-only host/cluster diagnostics, and `run_machine_command` for explicit allowlisted mutations. These direct machine controls execute through Workbench without an external AI worker. OpenClaw is optional operator capacity only for outcomes outside the direct structured command surface.

Choose transport according to what the current ChatGPT plan exposes. If full custom MCP write actions are available, the Workbench app/MCP connection can be used directly. If full MCP writes are unavailable, the private Workbench Git relay is the primary write/mutation transport: use connected GitHub to locate the private `workbench-relay` repository, read `WORKBENCH_CAPABILITIES.json` and `WORKBENCH_CHATGPT.md`, and use `relay/control`. Do not make the human copy prompts into OpenClaw or a terminal merely because MCP write actions are unavailable.
