# OpenClaw operations CLI contract

Workbench uses OpenClaw only for delegated machine-side/autonomous operations that ChatGPT cannot reasonably complete through Workbench's bounded direct control plane.

**OpenClaw is optional operator/autonomous capacity. It is not Workbench's default coder and is not required for routine bounded repository, server/cluster or typed Windows operations.**

The supported scripted invocation for the OpenClaw operations adapter is:

```text
openclaw agent --message <prompt>
```

`openclaw agent` is already a non-interactive/scripted command. Workbench must not append the browser-only `--headless` flag. If OpenClaw changes this contract, update the operations adapter and its regression tests together.

This does not change the ownership boundary:

- ChatGPT owns primary reasoning, source code, Git/GitHub, pull requests, CI, GitHub Actions and release orchestration;
- Workbench owns durable task state, bounded machine/repository controls, scheduling/continuation and the security/attention boundary;
- OpenClaw owns only the explicit delegated operation while that capacity is intentionally selected.

Direct ChatGPT development MUST NOT silently become OpenClaw delegation. Authenticated private-relay continuation is a separate trusted Workbench path and does not reopen implicit delegation.

A delegated OpenClaw operation does not redefine project requirements. Target-project canonical documentation remains authoritative under `docs/GOVERNANCE.md` and `docs/DECISIONS.md`.
