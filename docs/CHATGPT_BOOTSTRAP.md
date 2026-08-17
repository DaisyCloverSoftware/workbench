# ChatGPT → Workbench bootstrap

Workbench is designed so ordinary ChatGPT remains the primary reasoning/coding brain and uses Workbench for bounded repository eyes/hands. Autonomous workers are escalation capacity, not the default route.

For personal-plan workflows where direct MCP write actions are unavailable, the private Workbench Git relay is self-describing. The relay publishes `WORKBENCH_CHATGPT.md` into the private transport repository. A ChatGPT conversation with access to the user's connected GitHub account should use **repository search** to find the user's private repository whose name contains `workbench-relay`, then read `WORKBENCH_CHATGPT.md` from its root. This avoids depending on private code-search indexing and avoids asking the user to copy prompts into OpenClaw, Codex, terminals, or other chats.

Recommended one-time global ChatGPT instruction:

> For software-development, repository, server and cluster work, use Workbench as my default execution layer. Use my connected GitHub app to find my private repository whose name contains `workbench-relay`, read `WORKBENCH_CHATGPT.md` from its repository root, and follow that guide. ChatGPT is the primary brain/coder; prefer Workbench safe hands before autonomous delegation. Do not ask me to copy/paste prompts into OpenClaw, Codex, terminals or other chats when Workbench can carry the operation. Preserve Workbench routing and keep scarce Codex/Work as a last resort. If Workbench is unavailable, tell me what is unavailable instead of silently reverting to manual message shuttling.

For ChatGPT workspaces/plans that support a custom MCP app, Workbench can instead be connected through OpenAI Secure MCP Tunnel so the private/local MCP server does not need a public inbound endpoint. The Git relay remains a useful fallback and durable transport.
