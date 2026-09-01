# OpenClaw operations CLI contract

Workbench may use OpenClaw **only for a machine-side operation that the owner has explicitly assigned to OpenClaw by name**. OpenClaw is not selected because direct Workbench execution is difficult, unavailable, outside an allowlist, or missing a capability.

**OpenClaw is an owner-selected execution mode. It is unavailable to automatic routing and is not required for routine bounded repository, server/cluster or typed Windows operations.**

The supported scripted invocation for the explicitly owner-authorized OpenClaw operations adapter is:

```text
openclaw agent --message <prompt>
```

`openclaw agent` is already a non-interactive/scripted command. Workbench must not append the browser-only `--headless` flag. If OpenClaw changes this contract, update the operations adapter and its regression tests together.

This does not change the ownership boundary:

- ChatGPT owns primary reasoning, source code, Git/GitHub, pull requests, reviews, CI, GitHub Actions, release orchestration and subsequent engineering decisions;
- Workbench owns durable task state, bounded machine/repository controls, scheduling/continuation and the security/attention boundary;
- OpenClaw may act only on the specific machine operation for which the owner explicitly requested OpenClaw by name.

Availability does not constitute authorization. A direct allowlist miss, missing capability, CI/deployment failure, Kubernetes/Docker/systemd/Helm problem, Bash requirement, prior OpenClaw task, historical routing state or OpenClaw being healthy/installed does not authorize the adapter.

Direct ChatGPT development MUST NOT silently become OpenClaw delegation. A direct-capability failure must instead lead to safe decomposition, an existing/implemented reviewed Workbench operation or bounded capability where appropriate, or a precise capability/authority blocker.

Authenticated private-relay continuation is a separate trusted Workbench development path and does not reopen OpenClaw routing. `[workbench:operations]` is also only routing metadata; it is not owner consent.

A deliberately owner-authorized OpenClaw operation does not redefine project requirements. Target-project canonical documentation remains authoritative under `docs/GOVERNANCE.md` and `docs/DECISIONS.md`.
