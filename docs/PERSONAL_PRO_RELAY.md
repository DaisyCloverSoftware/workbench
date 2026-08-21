# Personal ChatGPT Pro relay

Workbench's private Git relay is a durable transport for ChatGPT plans/workspaces where direct custom-MCP mutation is unavailable or unsuitable. It is **not a second task engine** and it is not permission to bypass Workbench security/governance.

Canonical authority:

- project requirements/decisions: the target repository;
- Workbench trust boundaries: `SECURITY.md` and `docs/DECISIONS.md`;
- transport bootstrap details: private `WORKBENCH_CAPABILITIES.json` / `WORKBENCH_CHATGPT.md`.

## North-star loop

The relay supports two directions:

1. ChatGPT requests bounded repository/machine work or explicit durable continuation/autonomous work.
2. ChatGPT reads bounded results, durable task/dependency state and genuine attention questions.

The human should not carry prompts/results between ChatGPT, OpenClaw, terminals and Workbench when an authorised Workbench path exists.

## Safe-hands control: ChatGPT stays the brain

A private relay carries deterministic bounded requests in:

```text
relay/control/<control-id>.json
```

and results in:

```text
relay/control-outbox/<control-id>.json
```

Capabilities may include:

- privacy-minimal project discovery;
- bounded repository list/search/read;
- exact patch application;
- allowlisted safe commands;
- committed reviewed operations scripts;
- bounded machine/cluster controls;
- typed outbound Windows operations;
- durable context/memory controls;
- fixed Workbench maintenance actions.

The current machine-readable private capability manifest is the transport discovery source. Public documentation must not hard-code private host IDs, roots, topology or credentials.

Project discovery may span multiple authorised runner roots and return opaque `runner://...` references where a host filesystem path is unnecessary. Callers MUST use returned refs rather than infer private paths.

## Direct operations are not generic shell authority

Relay controls map onto Workbench's bounded execution policy. A control request is not permission to send arbitrary shell text.

- repository safe commands remain allowlisted;
- committed multi-step Bash operations must come from reviewed `scripts/ops/*.sh` at an exact authorised commit;
- direct machine operations use validated executable/argv policies;
- Windows host operations are separately outbound and typed/allowlisted;
- secrets/credential-like arguments or output are refused/withheld;
- unsupported actions fail closed.

OpenClaw availability is not a prerequisite for routine bounded operations.

## Autonomous work is a separate escalation channel

`relay/inbox` / `relay/outbox` may carry explicit autonomous work when genuinely needed. That is distinct from `relay/control` deterministic safe hands.

Direct ChatGPT development MUST NOT silently become OpenClaw/autonomous delegation merely because an autonomous transport exists.

The relay's internal compatibility mechanisms for trusted private continuation do not reopen implicit public/direct `delegate_task` authority.

## Authenticated durable continuation

Workbench can park work on a durable GitHub Actions dependency without holding an AI worker and later resume the original continuation automatically.

For a trusted private-relay development handoff, Workbench seals the continuation with an HMAC bound to:

- relay correlation ID;
- project;
- original continuation body.

When the external dependency becomes terminal, Workbench may append its exact owned dependency-update suffix after the proof. Validation accepts that Workbench-owned shape while arbitrary appended text fails closed.

Transport proof/correlation material is removed before the resumed worker receives the clean intent.

Evidence status is tracked in `docs/CURRENT_STATE.md`: automatic wake-up has live evidence; a clean post-validator proof of full productive completion remains unverified.

## Activity, presence and job state

The private relay is append-oriented historical transport. The runner therefore builds a bounded live view containing pending requests plus recent request/result pairs.

That bounded view prevents Dashboard cost from growing with the entire relay history. It does **not** make every recent file an executing job and does not define retention policy.

A runner activity record may include a bounded project/session `Active` lease. That is presence metadata. It MUST NOT by itself turn a terminal operation (`completed`/`failed`) into a running job. See `docs/operations-dashboard-contract.md`.

## Human attention

A genuine autonomous task attention boundary may use a matching answer envelope under the private relay. Workbench should consume each distinct answer once and resume the same durable task.

Normal implementation choices, transient provider outages and non-human dependency waits are not human-attention boundaries.

## Memory/context controls

Private relay controls may save/retrieve Workbench context/memory. These are advisory continuity aids only.

A material project decision is not canonically incorporated merely because it was stored in Workbench memory/context. Update the target repository's canonical documentation under `docs/GOVERNANCE.md`.

## Public versus private relay

A public relay is for deliberately harmless synthetic dogfood/status only. It MUST NOT contain private task intent, safe-hands control payloads, memory/context, credentials, deployment topology or private project content.

Real work requires a private authenticated relay transport. Relay transport cloning/authentication is separate from the Workbench public source tree and does not create/store a GitHub token in public source.

## Bootstrap

A fresh ChatGPT conversation using the private relay should:

1. locate the private Workbench relay repository through the authorised GitHub connection;
2. read `WORKBENCH_CAPABILITIES.json` for current machine-readable capabilities/version;
3. read `WORKBENCH_CHATGPT.md` for transport behaviour;
4. read the target project's canonical repository governance/requirements/decisions;
5. use a unique control/relay ID for each operation;
6. prefer deterministic bounded controls before autonomous delegation;
7. continue through non-human waits rather than asking the human to act as transport.

Private bootstrap documents describe **transport**, not product requirements. If they conflict with the target project's canonical requirements, fix the documentation/transport discrepancy rather than letting the private guide redefine the product.

## Retention

The bounded live projection is implemented; long-term historical transport retention/cleanup policy is not yet canonicalised at the 2026-08-21 baseline. Do not mass-delete private relay history until retention, rollback/audit needs and pending-request safety are explicitly defined.

## Why not automate the ChatGPT browser?

Workbench does not scrape ChatGPT output, inject DOM events or turn a consumer web session into an unofficial API. The relay uses supported GitHub transport plus Workbench's normal policy boundaries.
