# ChatGPT → Workbench bootstrap

Workbench is designed so ordinary ChatGPT remains the primary reasoning/coding brain and uses Workbench for bounded repository eyes/hands and routine machine execution. **OpenClaw is not part of the normal execution path. It is owner-opt-in only.**

For personal-plan workflows where direct MCP write actions are unavailable, the private Workbench Git relay is self-describing. The relay publishes `WORKBENCH_CAPABILITIES.json` and `WORKBENCH_CHATGPT.md` into the private transport repository. A ChatGPT conversation with access to the user's connected GitHub account should use repository search to find the user's private repository whose name contains `workbench-relay`, then read those two files from its root.

## Mandatory fresh-chat capability bootstrap

Before telling the user to run Bash, PowerShell, `git`, build/test commands, repository inspection, cluster commands, or other routine machine operations, ChatGPT **MUST first bootstrap the private Workbench relay and check the capabilities that exist now**.

The bootstrap conclusions are mandatory:

1. ChatGPT is the primary brain and owns engineering: reasoning, source changes, Git/GitHub, branches, pull requests, reviews, CI, GitHub Actions, releases, and subsequent engineering decisions.
2. The private Workbench control relay is the normal server/cluster/host/runtime execution transport.
3. Direct machine operations do not require OpenClaw.
4. OpenClaw is disabled by default from ChatGPT routing and requires an explicit owner instruction naming OpenClaw for the specific operation.
5. Historical conversations or prior OpenClaw tasks are not authority for current routing.
6. Failure of a direct capability never implicitly authorizes OpenClaw.

Historical limitations are not a reason to delegate a command to the human or to an autonomous operator. If the operation can be performed through `run_safe_command`, `inspect_machine`, `inspect_machine_batch`, `run_machine_command`, `run_operations_script`, a current bounded Windows bridge action, or another advertised Workbench control, ChatGPT performs it itself and reasons from the result.

When a direct operation is unavailable, ChatGPT must determine whether the work can be safely decomposed, performed by an existing reviewed `scripts/ops/*.sh` operation, or implemented as an additional bounded Workbench capability/reviewed operation within the authorised engineering scope. If none can carry it, report the exact capability or authority boundary.

**Do not invoke, select, suggest, or create an OpenClaw operation merely because direct Workbench execution cannot express the task.** Availability, difficulty, CI/deployment failure, Kubernetes/Docker/systemd/Helm trouble, Bash requirements, prior OpenClaw use, or an allowlist miss are not authorization.

The human is not a command transport layer. Ask the user to execute a local command only when the required input or machine is genuinely outside the currently advertised Workbench reach, or when a real permission/authority boundary requires human action. State the exact boundary.

Recommended one-time global ChatGPT instruction:

> For software-development, repository, server and cluster work, use Workbench as my default execution layer. Use my connected GitHub app to find my private repository whose name contains `workbench-relay`, read `WORKBENCH_CAPABILITIES.json` and `WORKBENCH_CHATGPT.md`, and follow the current contract. ChatGPT is the primary brain/developer and owns code, Git/GitHub, PRs, reviews, CI, GitHub Actions, releases and subsequent engineering decisions. Before asking me to run commands, first use current Workbench direct controls or a reviewed `scripts/ops` operation whenever possible. OpenClaw is owner-opt-in only: never select, invoke, suggest or use OpenClaw unless I explicitly ask for OpenClaw by name for that operation. A direct-capability failure never authorizes OpenClaw; decompose the work, add/use a bounded Workbench capability or reviewed operation when appropriate, or tell me the exact capability boundary.

For ChatGPT workspaces/plans that support a custom MCP app, Workbench can instead be connected through OpenAI Secure MCP Tunnel so the private/local MCP server does not need a public inbound endpoint. The Git relay remains a transport/bootstrap option; this transport choice does not change the OpenClaw authorization boundary.
