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

Non-human waits such as CI/build/release queues are not completion. Continue safe in-scope work until the contribution has an evidence-backed completion/checkpoint or a genuine human-only decision/permission boundary.

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
