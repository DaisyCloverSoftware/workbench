# Personal ChatGPT Pro relay

Workbench's private Git relay is a durable transport for ChatGPT plans/workspaces where direct custom-MCP mutation is unavailable or unsuitable. It is **not a second task engine** and it is not permission to bypass Workbench security/governance.

Canonical authority:

- project requirements/decisions: the target repository;
- Workbench trust boundaries: `SECURITY.md` and `docs/DECISIONS.md`;
- transport bootstrap details: private `WORKBENCH_CAPABILITIES.json` / `WORKBENCH_CHATGPT.md`.

Private bootstrap documents describe transport/capabilities. They do not override the target project's canonical requirements.

## North-star loop

Workbench needs two directions:

1. ChatGPT can **request** bounded repository/machine work or explicit autonomous/durable continuation work.
2. ChatGPT can **receive** bounded tool results, durable task/dependency state, final results and genuine attention questions.

The human should not carry prompts/results between ChatGPT, OpenClaw, terminals and Workbench when an authorised Workbench path exists.

## Transport shape

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

Only when autonomous execution is genuinely needed:

```text
ordinary ChatGPT
  -> private relay/inbox/<id>.json
  -> workbench-relay
  -> explicit autonomous Workbench path
  -> durable Workbench task / optional worker
  -> private relay/outbox/<id>.json
  -> ordinary ChatGPT
```

This remains one Workbench execution/security model with two transports, not two competing task systems.

## Safe-hands control: ChatGPT stays the brain

A **private** relay carries deterministic bounded Workbench controls through:

```text
relay/control/<control-id>.json
```

Results appear at:

```text
relay/control-outbox/<control-id>.json
```

Each control ID is one-shot/idempotent. Prefer deterministic controls before autonomous delegation.

Current capability discovery comes from the private machine-readable manifest rather than a stale hard-coded action list. Typical capabilities include:

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

Use the exact project reference discovered by Workbench:

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

### Exact patch example

```json
{
  "version": 1,
  "id": "patch_example_001",
  "action": "apply_patch",
  "project": "runner://example-project",
  "args": {
    "patch": "diff --git a/example.go b/example.go\n..."
  }
}
```

## Committed operations scripts

Multi-step server/cluster operations that exceed one direct allowlisted command can use `run_operations_script` without becoming generic shell authority.

The operation must:

- resolve to a Git-tracked regular `.sh` below `scripts/ops/`;
- execute at the registered checkout HEAD, or at an explicit full authorised Workbench commit according to current relay policy;
- receive literal argv rather than shell-composed command text;
- return bounded result metadata including the exact commit/script digest where supported.

Use reviewed committed operations scripts for repeatable operational workflows rather than pasting arbitrary shell programs through transport.

## Direct machine controls

Direct machine execution remains a bounded program + argv policy, split into read-only inspection and explicit mutation.

- `inspect_machine` is one read-only policy-checked command.
- `inspect_machine_batch` is a bounded sequential group of read-only commands; one item failing does not silently authorise a mutation or stop policy enforcement on later items.
- `run_machine_command` is an explicit one-at-a-time mutation path.
- There is deliberately no generic mutation batch.
- Secret-like arguments/output, alternate credential targets, unbounded streams and unsupported/high-risk operations are refused by policy.

OpenClaw availability is not a prerequisite for routine bounded machine work.

## Outbound Windows host controls

Windows access is a separate outbound typed bridge. There is no inbound generic Windows listener/shell.

The private capability manifest currently advertises typed operations including:

- host discovery;
- bounded Blender version check;
- bounded Unreal startup smoke;
- bounded Windows host-job status.

Reviewed committed operations scripts may submit other exact typed jobs when that operation exists in Workbench source. For example, the current Blender smoke-render wrapper submits the bounded headless Blender render operation through the host bridge at an authorised Workbench commit.

Every Windows job is checked again by the Windows-side allowlist. Do not replace typed operations with generic Windows command authority for convenience.

A generic host-job flow is:

```text
list_windows_hosts
  -> choose returned opaque host ID
  -> submit exact typed Windows operation
  -> receive job ID
  -> get_windows_host_job until terminal
```

Do not publish real host IDs or machine metadata in the public repository.

## Autonomous request envelope

Use the autonomous inbox only when ordinary ChatGPT cannot sensibly complete the outcome with deterministic safe hands and the task genuinely benefits from autonomous exploration/multi-step execution.

A relay request is one bounded UTF-8 file:

```text
relay/inbox/<relay-id>.json
```

Generic shape:

```json
{
  "version": 1,
  "id": "wb_example_001",
  "project": "<project-reference-required-by-current-private-contract>",
  "intent": "Implement the requested outcome and verify it."
}
```

The current private bootstrap contract is authoritative for the exact project-reference/envelope rules. The relay bounds IDs and intent, correlates the request to the durable Workbench task, and does not execute arbitrary repository content merely because it arrived through Git.

Direct ChatGPT development MUST NOT silently become OpenClaw/autonomous delegation merely because this inbox exists.

## Autonomous result envelope

The relay publishes durable state to:

```text
relay/outbox/<relay-id>.json
```

Public-safe transports expose only deliberately harmless status. A verified private transport may include the bounded worker report/error/attention information allowed by its result mode.

Example private result shape:

```json
{
  "version": 1,
  "id": "wb_example_001",
  "workbench_task_id": "task_123",
  "status": "needs_attention",
  "attention": "Choose A or B?",
  "updated_at": "2026-08-12T20:16:00Z"
}
```

Outbox publication is idempotent: unchanged generated envelopes are not repeatedly rewritten, and bounded retry protects concurrent Git writers.

## Authenticated durable continuation

Workbench can park work on a durable external dependency without holding an AI worker and later resume the original continuation automatically.

For trusted private-relay development continuation, Workbench seals the continuation with an HMAC bound to:

- relay correlation ID;
- project;
- original continuation body.

When the dependency becomes terminal, Workbench may append its exact owned dependency-result suffix after the proof. Validation accepts that Workbench-owned shape while arbitrary appended content fails closed.

Transport proof/correlation material is removed before the resumed worker receives the clean intent.

Evidence status is tracked in `docs/CURRENT_STATE.md`: automatic dependency wake-up has live evidence; a clean post-validator proof of the full `waiting_dependency → automatic resume → useful work → completed` sequence remains unverified at the 2026-08-21 baseline.

## Human answer envelope

When an autonomous task genuinely needs human input, the private relay may use:

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

Workbench consumes each distinct answer once and resumes the same durable task through the current attention-resolution path.

Normal implementation choices, transient provider outages and non-human dependency waits are not human-attention boundaries.

## Memory/context control

The private control channel can save/retrieve bounded Workbench memory/context without pretending a read-only MCP operation is a mutation.

Example compact-context shape:

```json
{
  "version": 1,
  "id": "ctxsave_example_001",
  "action": "save_context",
  "project": "runner://example-project",
  "args": {
    "objective": "Finish the current feature",
    "state": "Implementation is complete and tests are green.",
    "decisions": ["Re-check material decisions against canonical repository docs."],
    "constraints": ["Do not expose private deployment data."],
    "references": ["task-or-memory-id"],
    "open_threads": ["Prepare the next verified checkpoint."],
    "next_action": "Continue from canonical project state."
  }
}
```

Example reusable global routine shape:

```json
{
  "version": 1,
  "id": "memsave_example_001",
  "action": "save_memory",
  "args": {
    "scope": "global",
    "kind": "routine",
    "title": "Verification before completion",
    "content": "Run the repository's relevant tests and inspect the final diff before reporting completion.",
    "tags": ["verification", "routine"]
  }
}
```

Memory/context is advisory continuity data. A material project decision is not canonically incorporated merely because it was saved there; update the target repository under `docs/GOVERNANCE.md`.

## Activity, presence and job state

The private relay is append-oriented historical transport. The runner builds a bounded live view containing pending requests plus recent request/result counterparts so Dashboard cost does not grow with all history.

That projection is not a job scheduler and does not define retention.

A runner activity record may carry a bounded project/session `Active` lease. That is **presence metadata**. It MUST NOT by itself turn a terminal operation (`completed`/`failed`) into a running job. See `docs/operations-dashboard-contract.md`.

## Public versus private relay repositories

A public relay is for **deliberately harmless synthetic dogfood/status only**. Public mode must not process private safe-hands control, private project intent, memory/context, credentials or deployment details.

For real work, use a dedicated private relay transport. The Workbench source checkout and relay transport clone are separate concerns. The relay uses the operator's existing Git authentication; public source must not contain the credential.

When private/report mode is requested, installation/runtime checks should fail closed unless privacy can be established under the current transport policy. Do not downgrade that check to make setup easier.

## Install

The Workbench MCP/control service required by the relay must already be available according to the deployment model.

For harmless status-only dogfood using the source checkout transport where supported:

```bash
bash scripts/install-github-relay.sh
```

For real work, point the daemon at a separate private transport clone:

```bash
WORKBENCH_RELAY_REPO_DIR="$HOME/path/to/private-relay-clone" \
WORKBENCH_RELAY_PRIVATE=1 \
bash scripts/install-github-relay.sh
```

The private relay repository needs an existing target branch and non-interactive read/write Git authentication. It may otherwise remain a minimal transport repository.

Current relay configuration variables include:

- `WORKBENCH_RELAY_REPO_DIR` — transport clone; source checkout may be used only for deliberately harmless dogfood where supported;
- `WORKBENCH_RELAY_REMOTE` — Git remote, default `origin`;
- `WORKBENCH_RELAY_BRANCH` — target branch, default `main`;
- `WORKBENCH_RELAY_INTERVAL` — poll interval, default `10s` at this baseline;
- `WORKBENCH_RELAY_PRIVATE` — private-transport mode;
- `WORKBENCH_RELAY_RESULT_MODE` — `status` or `report`; private mode can use report mode;
- `WORKBENCH_RELAY_ASSUME_PRIVATE` — explicit override only for a genuinely private non-GitHub transport that cannot be verified automatically;
- `WORKBENCH_MCP_URL` — loopback Workbench MCP endpoint override.

Exact installer/runtime semantics should be verified against current script/source before changing deployment; this runbook does not make old environment defaults immutable product requirements.

## Fresh-chat bootstrap

A fresh ChatGPT conversation using the private relay should:

1. locate the authorised private Workbench relay repository through the connected GitHub account;
2. read `WORKBENCH_CAPABILITIES.json` for current machine-readable version/capabilities;
3. read `WORKBENCH_CHATGPT.md` for transport behaviour;
4. read the target project's canonical governance/requirements/decision documents;
5. discover project/host references through Workbench instead of inferring private paths/identifiers;
6. use a unique control/relay ID for each operation;
7. prefer deterministic bounded controls before autonomous delegation;
8. continue through non-human waits rather than making the human act as transport.

If a private bootstrap guide and current Workbench source disagree, record/fix the transport-contract drift. Do not let the private guide silently redefine product or security requirements.

## Retention

The bounded live projection is implemented. Long-term historical transport retention/compaction/cleanup policy is **not yet canonicalised** at the 2026-08-21 baseline.

Do not mass-delete relay history until pending-request safety, rollback/audit needs, retained operational evidence and compaction semantics are explicitly defined.

## Why not automate the ChatGPT browser?

Workbench does not scrape ChatGPT output, inject DOM events or turn a consumer web session into an unofficial API. The relay uses supported GitHub transport plus Workbench's normal policy boundaries.
