# Workbench project governance

This document defines how Workbench requirements, decisions, implementation and development evidence are governed. It is normative for maintainers, AI agents and future development chats.

## Source-of-truth hierarchy

When sources disagree, use this hierarchy together with the decision history in `docs/DECISIONS.md`:

1. Current normative requirements and contracts in the repository, including this document, `VISION.md`, `ARCHITECTURE.md`, `SECURITY.md` and domain contracts under `docs/`.
2. Current decisions and explicit supersession records in `docs/DECISIONS.md`.
3. Current architecture, data, UX, security, operations and deployment documentation.
4. Tests and acceptance criteria, as executable evidence of an intended contract.
5. Implementation, as evidence of what currently exists.
6. Issues, pull requests, commits and release history, as development/audit evidence.
7. Conversations, handoffs and memory/context capsules, as historical working material only.

Implementation does not silently redefine a requirement. An old formal document does not override a later decision that has been canonically recorded. If two current canonical documents conflict, development MUST stop at that decision boundary and reconcile the documents before the conflicting product change is made.

## Conversations are not project authority

A decision made in chat is not durably incorporated until the appropriate repository documentation is updated. New chats and agents MUST bootstrap from the repository and verified current state, not from large historical conversation dumps.

When a user correction changes a material product, UX, architecture, data, security, deployment or behavioural requirement, the correction MUST propagate to canonical documentation. Fixing code alone is insufficient.

Workbench context capsules and project/global memory are useful continuity aids, but they are advisory. They MUST NOT override the canonical repository record.

## Decision change protocol

For every material changed decision:

1. update the current requirement/contract;
2. record the current decision and rationale in `docs/DECISIONS.md` when the decision is architecturally or behaviourally significant;
3. record the replaced decision as SUPERSEDED or REJECTED when forgetting it could cause regression;
4. add or update a do-not-reintroduce rule where appropriate;
5. update acceptance criteria/tests before or with implementation;
6. only then change product code.

Never erase a significant rejected behaviour so completely that old code, PRs or chats can make it look valid again.

## Negative requirements are first-class requirements

Explicitly rejected behaviours are requirements too. The do-not-reintroduce section of `docs/DECISIONS.md` is normative. Tests SHOULD enforce negative requirements where practical.

## Completion and evidence vocabulary

Use precise implementation states:

- **specified** — canonical requirement exists;
- **implemented** — code exists in a working branch/commit;
- **tested** — relevant automated/manual checks passed against that implementation;
- **merged** — implementation is on the default branch;
- **released** — an official release/tag contains it;
- **deployed** — a runtime has been updated to it;
- **verified** — the intended behaviour was observed in the target environment with appropriate semantic acceptance evidence.

Do not use **complete**, **fixed**, **done** or equivalent merely because code was written, CI was green or a release was published.

## Semantic acceptance beats superficial health

Build success, process health, screenshots and UI responsiveness are necessary evidence for some changes but are not substitutes for behavioural acceptance. A semantically false dashboard can remain responsive and pass build gates.

Acceptance criteria MUST test the user-visible meaning of material behaviour where that meaning can differ from rendering or process health.

## Workbench development stopping rule

Non-human waits are part of execution, not completion. CI queues, builds, runner availability, dependency watches, release publication and deployments MUST NOT by themselves end an active development workflow.

A development workflow may end only when one of these is true:

1. the in-scope work is genuinely complete and evidence-backed;
2. a newly pushed/deployed/installed result exists that is genuinely ready for the user to inspect; or
3. a genuine human-only decision, permission or authority boundary blocks further in-scope progress.

While waiting on a non-human dependency, continue other safe in-scope work where useful. Durable Workbench continuation SHOULD resume dependent work automatically when the dependency becomes terminal.

This execution rule does not waive change control. If a requirement is ambiguous or a governance reset/freeze is active, agents MUST respect that boundary rather than treating autonomy as permission to invent product intent.

## Development freeze and governance resets

A declared governance reset freezes ordinary feature work until its documented completion gate passes. Audit/documentation/cleanup changes are permitted; feature fixes and redesigns are not.

If audit coverage is incomplete, the reset MUST be marked INCOMPLETE rather than completed by assumption.

## Repository hygiene

At sensible milestones, review:

- canonical documentation accuracy;
- superseded/rejected decisions;
- stale branches and worktrees;
- obsolete generated/debug/temporary files;
- old release/proof branches;
- unresolved TODOs;
- test fixtures that encode superseded behaviour;
- dependency/configuration drift;
- conversation-pruning eligibility.

Do not delete historical material merely because it is old. Before deletion, preserve unique current requirements, rationale, operational knowledge and audit information; verify the material is not needed for migration or rollback; then rely on Git history for obsolete code instead of keeping it in the active tree.

## Privacy during governance work

Governance/audit material in this public repository MUST comply with `PUBLIC_SOURCE_POLICY.md`. Record generic capabilities and states only. Do not copy private relay payloads, host identifiers, private addresses/topology, local paths, credentials, account inventory or private project content into public documentation.

## Required handoff shape

A fresh development handoff MUST be generated from canonical repository state and include:

- current product definition and architecture;
- authoritative documents;
- exact repository/release/deployment baseline;
- verified versus merely implemented state;
- known defects and unverified claims;
- next priorities and acceptance criteria;
- do-not-reintroduce rules.

Historical chats are not required to continue normal development after a completed governance reset.
