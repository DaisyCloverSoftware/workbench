# Workbench ChatGPT Compatibility Skill

This path is retained for older Workbench bootstrap flows. The canonical current plugin skill is `skills/workbench/SKILL.md`.

Use the shared Workbench MCP connection directly when it is installed. ChatGPT remains the developer and owns source code, Git/GitHub, pull requests, CI, and GitHub Actions. Use Workbench repository tools for bounded reads, exact ChatGPT-authored patches, safe build/test/lint/status/diff commands, durable context, and memory. Use `delegate_operation` only for machine-side shell/systemd/Docker/Kubernetes/Helm/deployment/runtime work that ChatGPT cannot execute directly, then use `await_operation` for the durable result. OpenClaw is the operator, never the coder. Only surface a genuine Workbench `needs_attention` request to the human.

If the direct shared MCP connection is unavailable, follow `cmd/workbench-relay/WORKBENCH_CHATGPT.md` for the private Git relay fallback instead of asking the human to copy prompts into OpenClaw.
