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

Historical conversations are not a required dependency for ordinary decisions after a completed governance reset. A historical statement that resurfaces in context is evidence only; it cannot silently override current canonical state. If historical material is deliberately reintroduced and accepted, the resulting decision must be written into the current canonical record before it becomes binding.

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

## Corrections-first, visually auditable sprint workflow

After a governance reset, product development MUST begin with a **corrections round** for known existing defects before new feature work starts.

After that, work is divided into bounded, visually/behaviourally auditable sprints. Each sprint follows this mandatory cycle:

1. **Scope and acceptance** — define the bounded user-visible outcome and observable acceptance criteria from canonical requirements before implementation.
2. **Implement** — complete the sprint without silently removing, shrinking or reinterpreting previously approved behaviour.
3. **Automated/technical verification** — pass the relevant tests, build, security, runner and semantic gates on the exact candidate head.
4. **Inspectable delivery** — deploy/install/publish the result to the appropriate inspectable surface. For Workbench this may be a Windows live build, cluster live surface or another explicitly named target; do not invent a conventional DEV surface where none exists.
5. **User observation** — give the user something concrete to inspect. Engineering status alone is not the sprint checkpoint.
6. **Corrections pass** — record issues observed from that inspection and execute the resulting corrections against the same sprint scope.
7. **Sign-off** — do not advance to the next product sprint until the inspectable result has been signed off by the user or a canonical decision explicitly changes the workflow.

A sprint is not accepted because a PR merged or CI passed. The checkpoint is the inspectable product behaviour.

If a correction reveals a material requirement change, apply the decision-change protocol before or with the code change so the correction cannot exist only in conversation.

## Workbench development stopping rule

Non-human waits are part of execution, not completion. CI queues, GitHub checks, automated tests, builds, PR/preview artifacts, runner operations, release/image/package publication where in scope, deployments/installations, rollouts, readiness checks and smoke checks MUST NOT by themselves end an active development workflow or return control to the owner.

Once an authorised sprint enters **IN DEVELOPMENT**, Development owns autonomous progression through the applicable engineering stages defined in `docs/SPRINT_GOVERNANCE.md`:

**IN DEVELOPMENT → ENGINEERING VERIFICATION → DEPLOYING TO REVIEW TARGET → READY FOR OWNER OBSERVATION → AWAITING OWNER OBSERVATION**

Development MUST inspect and re-check asynchronous engineering operations itself and continue when they become terminal. Repeated owner keepalive prompts such as `continue`, `carry on`, `check again`, or equivalent are not required merely because an ordinary engineering operation is still settling.

If an in-scope automated check fails, Development MUST investigate the failure, make an in-scope correction where already authorised, rerun the applicable verification, and continue the sprint without requiring a fresh owner prompt. The same continuity rule applies during correction rounds from **CORRECTIONS IN DEVELOPMENT** through the applicable verification/deployment stages to **READY FOR OWNER RE-OBSERVATION**.

For Workbench, the normal owner-return point is the actual observation gate: the exact candidate has passed applicable engineering verification, reached the agreed inspectable target, Development has verified that target is serving/running the intended candidate, and the Sprint Review is ready. Development may then enter **AWAITING OWNER OBSERVATION** and return control for genuine product observation.

A development workflow may return control earlier only when a genuine human-only decision, permission or authority boundary prevents further safe in-scope progress. Examples include an unresolved product/architecture decision; an irreconcilable ambiguity or conflict in canonical requirements; a destructive, irreversible, security-sensitive or explicitly approval-gated action; permissions/credentials unavailable through authorised tooling; an external failure that genuinely prevents further in-scope progress; or another material human-only authority decision.

While waiting on a non-human dependency, continue other safe in-scope work where useful. Durable Workbench continuation SHOULD resume dependent work automatically when the dependency becomes terminal.

This execution rule does not waive change control. Autonomous continuation is orchestration continuity, not permission to widen sprint scope, bypass publication/security/approval gates, transfer publication/deployment authority to a coding worker, consume external/scarce model credit without explicit authorisation, waive semantic acceptance or owner sign-off, or begin a later sprint. Only **SIGNED OFF** permits normal progression to the next sprint.

## Development freeze and governance resets

A declared governance reset freezes ordinary feature work until its documented completion gate passes. Audit/documentation/cleanup changes are permitted; feature fixes and redesigns are not.

If audit coverage is incomplete, the reset MUST be marked INCOMPLETE rather than completed by assumption, unless the documented completion gate explicitly allows a conscious human residual-risk acceptance and that acceptance is durably recorded.

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
- do-not-reintroduce rules;
- current corrections/sprint checkpoint and what is awaiting user inspection/sign-off.

Historical chats are not required to continue normal development after a completed governance reset.
