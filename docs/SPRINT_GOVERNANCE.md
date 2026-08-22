# Workbench sprint governance

Status: **NORMATIVE / MANDATORY**

This document is permanent project governance. It supplements `docs/GOVERNANCE.md` and governs all normal development after the 2026-08-21 governance reset.

## Core development loop

Normal development follows:

**SPECIFY → BUILD → TEST → DEPLOY/INSTALL TO AN OBSERVABLE TARGET → OWNER OBSERVES → RECORD OBSERVATIONS → CORRECT → REDEPLOY/REINSTALL → OWNER SIGNS OFF → PROTECT THE APPROVED BASELINE → NEXT SPRINT**

Never substitute a long open-ended development stream or a batch of completed sprints followed by one late owner review.

## Sprint structure

Work MUST be divided into small, coherent, independently auditable sprints. Prefer vertical slices that produce one meaningful observable product increment. Sprint size is optimised for how easily the project owner can determine whether the increment is correct, not for code volume.

Before implementation, each sprint MUST have a durable written record containing:

- sprint ID/name;
- objective;
- scope;
- explicit non-scope;
- acceptance criteria;
- starting baseline;
- observation target;
- implementation reference;
- review build/deployment reference;
- observation rounds;
- corrections;
- final sign-off;
- resulting protected behaviour;
- deferred issues.

Conversation text alone is not sufficient.

## Sprint boundary

Implement only the current sprint. Do not silently redesign adjacent behaviour, begin future sprint work for convenience, or remove approved behaviour unless that change is explicitly in scope. Newly discovered non-blocking problems are recorded for later.

## Engineering-complete before presentation

Before a sprint is presented for owner observation, complete all applicable implementation, tests, linting, type checking, migrations, documentation and decision updates, CI, deployment/installation, smoke testing, and regression verification. Do not present a knowingly broken or partially deployed sprint as ready.

## Observation target

Every sprint MUST end with something the owner can directly inspect. Prefer the actual functioning product rather than a description of code or screenshots when a live/installed product can be inspected. Where no meaningful visual interface exists, use an equally auditable artefact such as test output, API behaviour, generated reports, logs, benchmarks, command output, schema inspection, or CI evidence.

For Workbench, there is no conventional website-style DEV surface. The target may therefore be **Windows live**, **cluster live**, a PR/preview build, or another explicitly named inspectable surface. Never invent a DEV URL that does not exist.

Every review build/target MUST be identifiable with the applicable commit SHA, version/build, release/deployment revision, digest, state/schema version, feature flags, and timestamp. Never ask the owner to review a target without first verifying it is serving/running the intended build.

## Sprint review

At the observation gate provide a concise Sprint Review containing:

- sprint name/number;
- what changed;
- what was deliberately not changed;
- exact build/version/commit/deployment being reviewed;
- exact observation target;
- specific things worth checking.

Engineering status alone is not acceptance.

## Observation gate and correction rounds

Once a sprint is presented for owner observation, STOP starting new product work. The sprint enters **AWAITING OWNER OBSERVATION**. Maintenance needed to keep the review environment operational is allowed only if it does not silently alter the reviewed behaviour.

Every owner-reported issue becomes a durable **Sprint Observation Record**. Classify observations where useful as:

- BUG;
- REQUIREMENT MISUNDERSTANDING;
- UX CORRECTION;
- VISUAL CORRECTION;
- MISSING BEHAVIOUR;
- REGRESSION;
- DATA ISSUE;
- PERFORMANCE ISSUE;
- DOCUMENTATION/REQUIREMENT CORRECTION;
- NEW IDEA — FUTURE SPRINT.

If an observation shows canonical requirements are wrong or incomplete, update the canonical documentation. Rejected behaviour must be recorded where necessary so it cannot silently return.

Fix all in-sprint corrections, repeat applicable tests/CI/deployment/smoke verification, and return the same concrete target for re-observation. Use explicit round identifiers, for example **Sprint 04 — Review 1**, **Sprint 04 — Correction Round 1**, **Sprint 04 — Review 2**.

## Sign-off hard gate

A sprint is not complete because code exists, tests or CI pass, an agent believes it is correct, or the build is deployed. A normal user-facing sprint is complete only after:

1. implementation is complete;
2. engineering verification passes;
3. it is available at the agreed observation target;
4. the owner has had the opportunity to inspect it;
5. requested corrections have been addressed;
6. the owner explicitly signs it off, approves it, instructs development to proceed, or explicitly waives further review.

Do not infer approval because the conversation moves to another topic.

## Protected baseline

Signed-off behaviour becomes part of the protected project baseline. Later sprints MUST preserve it unless a later explicitly agreed requirement changes it. A future sprint that needs to change signed-off behaviour MUST identify **PREVIOUSLY SIGNED-OFF BEHAVIOUR TO BE CHANGED** in its plan and explain why.

Tests and canonical documentation SHOULD protect significant signed-off behaviour. Visual and interaction characteristics are functional behaviour when reviewed and approved, including position, order, labels, controls, available actions, hierarchy, interaction sequence, navigation, responsive behaviour, visibility, visual state, content and user feedback.

## Fixed sprint states

Use these states consistently:

**PLANNED**

↓

**IN DEVELOPMENT**

↓

**ENGINEERING VERIFICATION**

↓

**DEPLOYING TO REVIEW TARGET**

↓

**READY FOR OWNER OBSERVATION**

↓

**AWAITING OWNER OBSERVATION**

If corrections exist:

**CORRECTIONS REQUIRED**

↓

**CORRECTIONS IN DEVELOPMENT**

↓

**READY FOR OWNER RE-OBSERVATION**

Repeat as needed, then:

**SIGNED OFF**

Only **SIGNED OFF** permits normal progression to the next sprint.

## Autonomous sprint continuity and owner-return boundary

Once an authorised sprint enters **IN DEVELOPMENT**, Development owns autonomous progression through the applicable normal engineering stages:

**IN DEVELOPMENT → ENGINEERING VERIFICATION → DEPLOYING TO REVIEW TARGET → READY FOR OWNER OBSERVATION → AWAITING OWNER OBSERVATION**

Ordinary engineering latency is part of execution. It MUST NOT return control to the owner or require keepalive prompts such as `continue`, `carry on`, `check again`, or equivalent. Development MUST inspect and re-check asynchronous engineering operations itself and continue when they become terminal. This includes ordinary waiting/settling for:

- CI and GitHub checks;
- automated tests;
- builds;
- PR/preview artifacts;
- release, image or package publication when in scope;
- deployment or installation;
- rollout, readiness and smoke checks;
- runner operations and other asynchronous engineering dependencies.

If an in-scope automated check fails, Development MUST investigate the failure, make an in-scope correction where already authorised, rerun the applicable verification, and continue the sprint without requiring a fresh owner prompt. A failure that exposes a genuine owner-only decision or authority boundary remains an exception.

Correction rounds use the same continuity rule. Once the owner has required corrections and the sprint enters **CORRECTIONS IN DEVELOPMENT**, Development owns progression through the applicable verification and deployment/installation stages until the corrected result reaches **READY FOR OWNER RE-OBSERVATION**. No additional owner `continue` prompt is required between those stages.

For Workbench, the normal point to return control to the owner is the actual observation gate: the exact candidate has passed applicable engineering verification, reached the agreed inspectable target, Development has verified that target is serving/running the intended candidate, and the Sprint Review is ready. Development may then enter **AWAITING OWNER OBSERVATION** and return control for genuine product observation.

Development may return control earlier only for a genuine human-only boundary that prevents safe in-scope continuation, such as:

- an unresolved product or architecture decision;
- a conflict or ambiguity in canonical requirements that cannot safely be resolved;
- a destructive, irreversible, security-sensitive or explicitly approval-gated action;
- permissions or credentials unavailable through authorised tooling;
- an external failure that genuinely prevents further in-scope progress;
- another material human-only authority decision.

Ordinary CI/build/publication/deployment latency is not such a boundary.

Autonomous continuation is orchestration continuity only. It does not widen sprint scope, bypass an approval or security gate, transfer publication/deployment authority to a coding worker, authorise external/scarce model-credit use, waive semantic acceptance, waive owner observation/sign-off, or permit the next sprint to begin. Only **SIGNED OFF** permits normal progression to the next sprint.

## No batched owner reviews

Do not implement Sprint 1, Sprint 2, Sprint 3 and Sprint 4 and then ask for one combined review. Each sprint must pass its own implementation → verification → observable-target → owner-observation → corrections → sign-off loop before the next sprint starts.

## Completion language

Distinguish **implementation complete**, **ready for owner observation**, **sprint signed off**, and **project complete**. Do not use ambiguous “done” language for a build merely pushed/deployed awaiting review.

## Emergency fixes

A production emergency may interrupt the normal sequence only when explicitly declared as a hotfix with limited scope. Fix and verify it, present an observable result where applicable, record the change, update the protected baseline, then return to the interrupted sprint. Emergency work is not permission for uncontrolled development.

# Mandatory Sprint 0 — Existing Corrections & Baseline Stabilisation

Before any new roadmap or feature sprint begins after the governance reset, Workbench MUST complete **Sprint 0 — Existing Corrections & Baseline Stabilisation**. Sprint 0 is a hard gate. No Sprint 1 or forward feature development may begin before Sprint 0 is explicitly signed off by the owner.

## Purpose and scope

Sprint 0 exists to fix and sign off what already exists before building what comes next. It covers existing known defects and discrepancies, including:

- known bugs and regressions;
- broken workflows;
- incorrect UX or visual behaviour;
- missing behaviour that was already required;
- previously approved behaviour that disappeared;
- existing requirement misunderstandings;
- deployment inconsistencies and target discrepancies;
- stale/incorrect data behaviour;
- existing test failures;
- governance-audit findings where implementation does not match canonical specification.

It is not a feature-development sprint.

## Establish the current observable baseline

Before owner baseline observation, verify exactly what the applicable current review target is serving/running. Record where applicable commit SHA, image digest, application version/build, deployment revision, state/schema version, relevant feature flags, date/time, and the exact observation target. Do not assume a runtime reflects repository HEAD.

For Workbench specifically, do not assume a website-style DEV environment exists. Use the canonical deployment terminology and the actual inspectable target, such as Windows live or cluster live.

## Initial owner baseline observation

The current product MUST be presented to the owner for the express purpose of finding what is already wrong before forward development. Existing behaviour that should already work and is identified as wrong becomes a Sprint 0 correction, not an optional improvement.

## Baseline Correction Register

Maintain a durable **Baseline Correction Register**. Each correction record MUST include:

- ID;
- description;
- affected area;
- expected behaviour;
- current incorrect behaviour/evidence;
- source requirement/decision where applicable;
- severity;
- whether it is a regression;
- whether it restores previously signed-off behaviour;
- implementation status;
- verification status;
- owner observation status.

The register combines known pre-reset defects, governance-audit findings, specification/implementation mismatches, failing regression tests, owner baseline observations, and unresolved previously reported bugs.

## Corrections versus new ideas

Classify Sprint 0 observations as either:

**EXISTING CORRECTION** — the existing product already should do this or previously did it correctly; it belongs in Sprint 0.

**NEW FEATURE / NEW REQUIREMENT** — genuinely new behaviour not previously required; record it for the future backlog and do not implement it during Sprint 0 unless the owner explicitly reclassifies it as baseline scope.

## Correct the baseline

Work through all agreed Sprint 0 corrections against current canonical requirements. Do not restore historical behaviour merely because it once existed. Restore only current intended behaviour. When a correction changes canonical documentation, update it with the implementation. Where superseded or rejected behaviour returned, strengthen requirements, regression tests, decision records and/or do-not-reintroduce rules so recurrence is harder.

## Regression verification

Sprint 0 requires broader regression coverage than an ordinary sprint. Run all applicable unit, integration, browser/end-to-end, API, schema, lint/type, deployment, security and visual/manual checks. Pay particular attention to historically approved behaviour and ensure one correction does not silently remove another behaviour.

## Corrected baseline review

After agreed corrections are implemented and engineering verification passes, deploy/install the corrected baseline to the normal observation target and verify the target is actually running the corrected build. Then report **SPRINT 0 — READY FOR OWNER OBSERVATION** with the exact target.

Owner correction rounds continue as **Sprint 0 — Correction Round N** until the owner explicitly approves the baseline.

## Sprint 0 sign-off

When the owner explicitly signs off Sprint 0, record **SPRINT 0 — SIGNED OFF**. The resulting corrected application becomes the **PROTECTED DEVELOPMENT BASELINE**. Only then may Sprint 1 begin.

Permanent rule: **FIX AND SIGN OFF WHAT ALREADY EXISTS BEFORE BUILDING WHAT COMES NEXT.**
