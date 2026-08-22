# Contributing

Workbench is deliberately early. Contributions are welcome, especially when they make the system more provider-neutral, safer or less annoying to supervise.

## Read the canonical project record first

Before implementing a material change, read:

- `docs/GOVERNANCE.md`;
- `docs/DECISIONS.md`;
- `docs/CURRENT_STATE.md`;
- `ARCHITECTURE.md`;
- `SECURITY.md`;
- the domain contract relevant to the change.

Conversations, old PR descriptions and historical code are not project authority.

If a requirement changes, update the canonical documentation/decision record **before or with** the implementation. If the new decision supersedes/rejects earlier behaviour, record that explicitly and add a do-not-reintroduce rule/test when regression risk warrants it.

Do not make implementation convenience silently redefine the product.

## Evidence and completion

Use precise states: specified, implemented, tested, merged, released, deployed and verified.

A green build, screenshot or responsiveness check does not prove semantic correctness. Add/perform acceptance evidence for the meaning changed by the PR. In particular, Operations changes must satisfy `docs/operations-dashboard-contract.md` and `docs/UI_ACCEPTANCE_V0.9.md`.

Non-human waits such as CI/build/release/publication/deployment queues are execution latency, not owner handoff points. During an authorised sprint, re-check asynchronous engineering operations and continue through the applicable verification and inspectable-delivery stages without requiring owner keepalive prompts such as `continue`, `carry on` or `check again`. If an in-scope automated check fails, investigate it, make an already-authorised in-scope correction, rerun the applicable verification, and continue.

For normal sprint work, return control to the owner at the observation gate only after the exact candidate is genuinely ready and the agreed target has been verified to be running/serving it, or earlier for a genuine human-only decision, permission or authority boundary. See `docs/SPRINT_GOVERNANCE.md` and WB-DEC-010. This continuity rule does not widen scope or bypass publication, security, explicit-approval, semantic-acceptance or sign-off gates.

## Implementation guidance

Keep provider-specific assumptions inside adapters. Core routing should reason about capabilities, trust and cost classes rather than brands.

Use bounded typed operations instead of widening remote command authority. Do not add a generic Windows shell merely to avoid defining a typed operation.

Direct ChatGPT development must not silently route through OpenClaw/autonomous delegation. Autonomous workers are explicit optional capacity.

Do not add browser automation for consumer AI chat products as a shortcut around unsupported integration paths.

## Before opening a pull request

```bash
gofmt -w .
go test ./...
```

Also run/record any relevant platform-specific, semantic or deployment acceptance required by the changed contract.

Public changes must comply with `PUBLIC_SOURCE_POLICY.md`; never copy private deployment/relay details into source, tests, PR text or screenshots.
