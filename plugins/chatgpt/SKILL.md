# Workbench ChatGPT Compatibility Skill

This path is retained for older Workbench bootstrap flows. The canonical current skill is `skills/workbench/SKILL.md`.

ChatGPT remains the developer and owns reasoning, source code, Git/GitHub, pull requests, reviews, CI, GitHub Actions, releases and subsequent engineering decisions. Use Workbench for bounded repository hands, durable context/memory, `inspect_machine` for one routine read-only host/cluster diagnostic, `inspect_machine_batch` on the private relay for ordered independent read-only diagnostics, `run_machine_command` for explicit allowlisted mutations, and `run_operations_script` for reviewed Git-tracked operations under `scripts/ops/`. These direct controls execute through Workbench without an external AI worker.

For common read-only checks, prefer Workbench's reviewed `scripts/ops/workbench-health.sh`, `scripts/ops/cluster-health.sh`, and `scripts/ops/namespace-health.sh <namespace>` from an exact advertised `runner://workbench` commit. Use `inspect_machine_batch` when several custom independent reads are needed and no built-in snapshot already answers the question.

Choose transport according to what the current ChatGPT plan exposes. If full custom MCP write actions are available, use only the Workbench app/MCP tools actually advertised by that connection. If full MCP writes are unavailable, the private Workbench Git relay is the primary write/mutation transport: use connected GitHub to locate the private `workbench-relay` repository, read `WORKBENCH_CAPABILITIES.json` and `WORKBENCH_CHATGPT.md`, and use `relay/control`.

**OpenClaw is owner-opt-in only.** ChatGPT and Workbench must never select, invoke, suggest or use OpenClaw automatically. Only an explicit owner instruction naming OpenClaw for the applicable operation authorizes it. A direct allowlist miss, missing capability, CI/deployment failure, Kubernetes/Docker/systemd/Helm trouble, Bash requirement, prior OpenClaw use, or OpenClaw availability is not authorization. If direct Workbench execution cannot express an operation, safely decompose it, use or implement an appropriate reviewed operation/bounded Workbench capability within scope, or report the exact capability/authority blocker.

`[workbench:operations]` is routing metadata, not owner consent. On the private relay, any deliberate OpenClaw request also requires the separate manifest-advertised owner-authorization signal. Normal routing must never synthesize that authorization.

Do not make the human copy prompts into OpenClaw or a terminal merely because MCP write actions are unavailable.
