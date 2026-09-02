# Personal ChatGPT Pro relay

Workbench's private Git relay is a durable transport for ChatGPT plans/workspaces where direct custom-MCP mutation is unavailable or unsuitable. It is **not a second task engine** and it is not permission to bypass Workbench security/governance.

Canonical authority:

- project requirements/decisions: the target repository;
- Workbench trust boundaries: `SECURITY.md`, `docs/GOVERNANCE.md` and `docs/DECISIONS.md`;
- transport bootstrap details: private `WORKBENCH_CAPABILITIES.json` / `WORKBENCH_CHATGPT.md`.

Private bootstrap documents describe transport/capabilities. They do not override the target project's canonical requirements.

## North-star loop

Workbench needs two directions:

1. ChatGPT can request bounded repository/machine work through the direct control relay and can deliberately create an owner-authorized OpenClaw operation only when the owner explicitly asks for OpenClaw by name.
2. ChatGPT can receive bounded tool results, durable task/dependency state, final results and genuine attention questions.

The human should not carry prompts/results between ChatGPT, terminals and Workbench when an authorised Workbench path exists.

## Normal transport shape

When a ChatGPT workspace supports the required direct MCP actions, Workbench's loopback MCP surface may be exposed through the approved outbound secure tunnel/app path.

When direct custom-MCP mutation is unavailable or unsuitable, the private Git relay carries the request/result loop:

```text
ordinary ChatGPT
  -> authorised GitHub connection
  -> private relay/control/<id>.json
  -> workbench-relay
  -> Workbench bounded safe hands / machine controls
  -> private relay/control-outbox/<id>.json
  -> ordinary ChatGPT
```

This direct control path is the normal server/cluster/host/runtime execution route. It does not require OpenClaw.

## Safe-hands control: ChatGPT stays the brain

A **private** relay carries deterministic bounded Workbench controls through:

```text
relay/control/<control-id>.json
```

Results appear at:

```text
relay/control-outbox/<control-id>.json
```

Each control ID is one-shot/idempotent. Current capability discovery comes from the private machine-readable manifest rather than a stale hard-coded action list. Typical capabilities include:

- privacy-minimal project discovery;
- bounded repository list/search/read;
- exact patch application;
- allowlisted safe build/test/status commands;
- bounded direct machine inspection/mutation;
- read-only batched machine inspection;
- reviewed committed multi-step operations scripts at an exact authorised Workbench commit;
- outbound typed/allowlisted Windows operations;
- durable task/status reads;
- context/memory controls;
- fixed Workbench maintenance actions.

The relay validates request shape, enforces the relevant Workbench policy, caps results and withholds probable secret material. `run_safe_command`, machine controls and Windows operations are not generic shell escapes. `apply_patch` does not grant push/deploy authority.

Project discovery may span multiple authorised roots and return opaque `runner://...` references. Use the exact project reference returned by Workbench; do not infer host filesystem paths.

### Project discovery example

```json
{
  "version": 1,
  "id": "projects_example_001",
  "action": "list_projects",
  "args": {}
}
```

### Safe repository test example

```json
{
  "version": 1,
  "id": "safe_example_001",
  "action": "run_safe_command",
  "project": "runner://example-project",
  "args": {
    "command": "go test ./..."
  }
}
```

## Committed operations scripts

Multi-step server/cluster operations that exceed one direct allowlisted command can use `run_operations_script` without becoming generic shell authority.

The operation must:

- resolve to a Git-tracked regular `.sh` below `scripts/ops/`;
- execute at the registered checkout HEAD, or at an explicit full authorised commit according to current relay policy;
- receive literal argv rather than shell-composed command text;
- return bounded result metadata including the exact commit/script digest where supported.

Use reviewed committed operations scripts for repeatable operational workflows rather than pasting arbitrary shell programs through transport.

If no current direct capability or reviewed operation can express an operation, ChatGPT should safely decompose it, implement an appropriate bounded Workbench capability/reviewed operation within authorised engineering scope, or report the exact capability/authority blocker. A missing direct capability is **not** authorization to use OpenClaw.

## Direct machine controls

Direct machine execution remains a bounded program + argv policy, split into read-only inspection and explicit mutation.

- `inspect_machine` is one read-only policy-checked command.
- `inspect_machine_batch` is a bounded sequential group of read-only commands; one item failing does not silently authorise a mutation or stop policy enforcement on later items.
- `run_machine_command` is an explicit one-at-a-time mutation path.
- There is deliberately no generic mutation batch.
- Secret-like arguments/output, alternate credential targets, unbounded streams and unsupported/high-risk operations are refused by policy.

OpenClaw availability is not a prerequisite for routine bounded machine work and never changes the authority of these controls.

## Outbound Windows host controls

Windows access is a separate outbound typed bridge. There is no inbound generic Windows listener/shell.

The private capability manifest advertises the currently supported typed operations. Every Windows job is checked again by the Windows-side allowlist. Do not replace typed operations with generic Windows command authority for convenience.

A generic host-job flow is:

```text
list_windows_hosts
  -> choose returned opaque host ID
  -> submit exact typed Windows operation
  -> receive job ID
  -> get_windows_host_job until terminal
```

Do not publish real host IDs or machine metadata in the public repository.

## Hard OpenClaw authorization boundary

OpenClaw is an **owner-selected execution mode**, not a fallback. ChatGPT and Workbench MUST NOT select, invoke, suggest or use OpenClaw automatically.

Only an explicit owner instruction naming OpenClaw for the applicable operation authorizes use. The following are not authorization:

- an operation being difficult or long-running;
- a direct command being unavailable or outside an allowlist;
- Workbench lacking a capability;
- CI or deployment failure;
- Kubernetes, Docker, systemd or Helm problems;
- Bash or multi-step troubleshooting requirements;
- previous OpenClaw use or old task history;
- OpenClaw being installed, healthy or immediately available.

Availability does not constitute authorization. Unless the owner explicitly asks for OpenClaw by name, the effective OpenClaw authorization state is denied.

### Explicit-use autonomous request envelope

`relay/inbox/<relay-id>.json` exists only to preserve deliberate explicit-use functionality. It is unavailable to automatic routing.

The exact current private bootstrap contract is authoritative for the request envelope. In addition to the `[workbench:operations]` routing marker, an OpenClaw request must carry the separate manifest-advertised owner-authorization signal. The operations marker by itself is machine routing metadata and is not proof of owner consent. Normal routing logic must never synthesize owner authorization.

Conceptual shape after an owner explicitly names OpenClaw:

```json
{
  "version": 1,
  "id": "wb_openclaw_example_001",
  "project": "runner://example-project",
  "intent": "<manifest owner-authorization marker> [workbench:operations] Investigate the explicitly owner-authorized machine operation."
}
```

The relay rejects ordinary operations-lane metadata without the owner-authorization signal. A direct capability failure must therefore produce safe decomposition, a reviewed operation/capability path, or a precise unsupported-capability result rather than an OpenClaw task.

When an explicitly owner-authorized OpenClaw task exists, it remains machine-operations-only. It must not own source changes, Git/GitHub, PRs, CI, GitHub Actions, releases or subsequent engineering decisions.

## Autonomous result envelope

For a deliberately authorized OpenClaw task, the relay publishes durable state to:

```text
relay/outbox/<relay-id>.json
```

Public-safe transports expose only deliberately harmless status. A verified private transport may include the bounded worker report/error/attention information allowed by its result mode.

Outbox publication is idempotent: unchanged generated envelopes are not repeatedly rewritten, and bounded retry protects concurrent Git writers.

## Authenticated durable continuation

Workbench can park development work on a durable external dependency without holding an AI worker and later resume the original continuation automatically.

For trusted private-relay development continuation, Workbench seals the continuation with an HMAC bound to:

- relay correlation ID;
- project;
- original continuation body.

When the dependency becomes terminal, Workbench may append its exact owned dependency-result suffix after the proof. Validation accepts that Workbench-owned shape while arbitrary appended content fails closed. Transport proof/correlation material is removed before the resumed worker receives the clean intent.

Authenticated development continuation is distinct from OpenClaw authorization and must not be interpreted as permission to enter the OpenClaw operations lane.

## Human answer envelope

When an explicitly owner-authorized autonomous task genuinely needs human input, the private relay may use:

```text
relay/answers/<relay-id>.json
```

Generic shape:

```json
{
  "version": 1,
  "id": "wb_example_001",
  "answer": "Choose A."
}
```

Workbench consumes each distinct answer once and resumes the same durable task through the current attention-resolution path. Normal implementation choices, transient provider outages and non-human dependency waits are not human-attention boundaries.

## Memory/context control

The private control channel can save/retrieve bounded Workbench memory/context without pretending a read-only MCP operation is a mutation. Memory/context is advisory continuity data. A material project decision is not canonically incorporated merely because it was saved there; update the target repository under `docs/GOVERNANCE.md`.

## Activity, presence and job state

The private relay is append-oriented historical transport. The runner builds a bounded live view containing pending requests plus recent request/result counterparts so Dashboard cost does not grow with all history.

That projection is not a job scheduler and does not define retention. Project/session `Active` lease data is presence metadata and MUST NOT by itself turn a terminal operation into a running job. See `docs/operations-dashboard-contract.md`.

## Public versus private relay repositories

A public relay is for **deliberately harmless synthetic dogfood/status only**. Public mode must not process private safe-hands control, private project intent, memory/context, credentials or deployment details.

For real work, use a dedicated private relay transport. The Workbench source checkout and relay transport clone are separate concerns. The relay uses the operator's existing Git authentication; public source must not contain the credential.

When private/report mode is requested, installation/runtime checks should fail closed unless privacy can be established under the current transport policy. Do not downgrade that check to make setup easier.

## Fresh-chat bootstrap

A fresh ChatGPT conversation using the private relay should:

1. locate the authorised private Workbench relay repository through the connected GitHub account;
2. read `WORKBENCH_CAPABILITIES.json` for current machine-readable version/capabilities;
3. read `WORKBENCH_CHATGPT.md` for transport behaviour;
4. read the target project's canonical governance/requirements/decision documents;
5. discover project/host references through Workbench instead of inferring private paths/identifiers;
6. use a unique control/relay ID for each operation;
7. use deterministic bounded controls and reviewed operations for normal machine work;
8. treat OpenClaw as denied unless the current owner instruction explicitly names OpenClaw;
9. continue through non-human waits rather than making the human act as transport.

Historical conversations claiming cluster/server work requires OpenClaw are not authoritative. Direct-capability failure never authorizes OpenClaw.

## Retention

The bounded live projection is implemented. Long-term historical transport retention/compaction/cleanup policy remains a separate governance question. Do not mass-delete relay history until pending-request safety, rollback/audit needs, retained operational evidence and compaction semantics are explicitly defined.

## Why not automate the ChatGPT browser?

Workbench does not scrape ChatGPT output, inject DOM events or turn a consumer web session into an unofficial API. The relay uses supported GitHub transport plus Workbench's normal policy boundaries.
