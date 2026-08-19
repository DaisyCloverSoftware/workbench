# Workbench ChatGPT Compatibility Skill

This path is retained for older Workbench bootstrap flows. The canonical current skill is `skills/workbench/SKILL.md`.

ChatGPT remains the developer and owns source code, Git/GitHub, pull requests, CI, and GitHub Actions. Use Workbench for bounded repository hands, durable context/memory, `inspect_machine` for routine read-only host/cluster diagnostics, `run_machine_command` for explicit allowlisted mutations, and `run_operations_script` for reviewed Git-tracked operations under `scripts/ops/`. These direct controls execute through Workbench without an external AI worker. OpenClaw is optional operator capacity only for outcomes outside the direct structured/committed operations surface.

For common read-only checks, prefer Workbench's reviewed `scripts/ops/workbench-health.sh` and `scripts/ops/namespace-health.sh <namespace>` from an exact advertised `runner://workbench` commit. They provide compact bridge/namespace health evidence without exposing credentials or creating a burst of separate relay requests.

Choose transport according to what the current ChatGPT plan exposes. If full custom MCP write actions are available, the Workbench app/MCP connection can be used directly. If full MCP writes are unavailable, the private Workbench Git relay is the primary write/mutation transport: use connected GitHub to locate the private `workbench-relay` repository, read `WORKBENCH_CAPABILITIES.json` and `WORKBENCH_CHATGPT.md`, and use `relay/control`. Do not make the human copy prompts into OpenClaw or a terminal merely because MCP write actions are unavailable.
