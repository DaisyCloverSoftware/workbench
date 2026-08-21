# Sprint 00 — Existing Corrections & Baseline Stabilisation

Status: **IN DEVELOPMENT — BASELINE ESTABLISHMENT**

## Sprint objective

Establish the exact observable post-reset Workbench baseline, identify and durably register all existing corrections that must be resolved before new feature development, correct the agreed baseline against current canonical requirements, and obtain explicit owner sign-off on the corrected product.

## Scope

- verify the exact current canonical source/release/runtime baseline;
- establish the actual inspectable Workbench target rather than assuming a conventional DEV surface;
- present the existing product for owner baseline observation before forward feature work;
- maintain the Baseline Correction Register;
- resolve agreed existing corrections only;
- run broad regression verification appropriate to baseline stabilisation;
- install/deploy the corrected result to the applicable observable target;
- conduct owner observation and correction rounds until explicit Sprint 0 sign-off;
- establish the signed-off result as the protected development baseline.

## Explicit non-scope

- no roadmap feature work;
- no Sprint 1 work;
- no new authoritative cross-plane job model;
- no knowledge-graph/searchable-decisions feature work;
- no richer selected-job drawer/reorder/control design beyond current requirements;
- no unrelated dashboard redesign;
- no implementation of genuinely new ideas unless the owner explicitly reclassifies them as baseline corrections.

## Starting baseline

Canonical source at Sprint 0 start:

- repository: `DaisyCloverSoftware/workbench`;
- branch: `main`;
- commit: `92666ad51f5407d511b89e0e516250a30af04adc`;
- commit time: 2026-08-21 12:45:25 UTC;
- stable source version: `0.9.55`;
- stable release/tag: `v0.9.55`;
- cluster live: 0.9.55 was previously deployment/version verified;
- Windows live: last directly observed desktop was 0.9.54; installed Windows 0.9.55 semantic inspection remained unverified at cutover.

Sprint working branch:

- `sprint/0-baseline-stabilisation`;
- branched from `92666ad51f5407d511b89e0e516250a30af04adc`.

## Acceptance criteria

Sprint 0 may be signed off only when all of the following are true:

1. The actual review target and exact build/version/commit are verified rather than assumed.
2. The owner has had an initial opportunity to inspect the current product specifically to identify existing defects.
3. All agreed existing corrections are recorded durably in `docs/BASELINE_CORRECTION_REGISTER.md`.
4. New ideas are separated from existing corrections and are not silently implemented in Sprint 0.
5. Every agreed Sprint 0 correction is implemented against current canonical requirements, with canonical documentation/negative requirements strengthened where needed.
6. Applicable broad regression checks pass against the exact corrected candidate.
7. The corrected candidate is installed/deployed to the agreed observable target and that target is verified to be running the intended build.
8. Owner correction rounds are recorded and resolved.
9. The owner explicitly signs off Sprint 0 or explicitly releases the gate.
10. The resulting protected baseline and resulting protected behaviours are durably recorded.

## Initial observation target

Workbench has no conventional website-style DEV deployment. The first required observable target is therefore:

**Windows live — installed Workbench desktop, Operations dashboard, version 0.9.55 or a later explicitly identified Sprint 0 candidate derived from the canonical baseline.**

The first baseline observation focuses on the existing Operations semantics that were corrected in 0.9.55 source/release but not yet inspected on the installed Windows UI:

- terminal completed/failed history must not inflate live Running counts merely because the project/session presence lease is active;
- genuine running work must remain visible;
- presence/session activity must remain separate from job execution state;
- no fabricated queue position or completion percentage may appear where Workbench lacks authority.

This initial observation is for finding existing problems, not for proposing new feature design.

## Known starting correction set

The initial register is seeded from `docs/CURRENT_STATE.md` and the governance reset evidence. It currently includes:

- Windows Operations 0.9.55 installed-target acceptance closure;
- release publication reliability/no-op retrigger dependency;
- unattended continuation end-to-end acceptance;
- private relay retention/compaction governance;
- Blender headless GPU render acceptance;
- Unreal startup/`zen` investigation and acceptance.

The owner’s baseline observations may add further **existing corrections**. Genuine new requirements are deferred.

## Observation rounds

No owner observation round has yet been completed in this Sprint 0 record.

## Implementation reference

No product correction implementation has started in Sprint 0.

Governance/sprint-record commit on the Sprint 0 branch begins at `15573b831c7c18b8170c960e6832ba7aaf341c73`.

## Resulting protected behaviour

Pending Sprint 0 sign-off.

## Deferred issues

Future feature/roadmap work remains deferred until Sprint 0 is **SIGNED OFF**.
