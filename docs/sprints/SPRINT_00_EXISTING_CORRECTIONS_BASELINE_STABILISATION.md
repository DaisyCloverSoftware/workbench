# Sprint 00 — Existing Corrections & Baseline Stabilisation

Status: **IN DEVELOPMENT — BASELINE ESTABLISHMENT BLOCKED**

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

Product/source baseline at Sprint 0 start:

- repository: `DaisyCloverSoftware/workbench`;
- branch: `main`;
- audited product-source commit: `92666ad51f5407d511b89e0e516250a30af04adc`;
- audited commit time: 2026-08-21 12:45:25 UTC;
- stable source/application version: `0.9.55`;
- stable release/tag: `v0.9.55`;
- cluster live: 0.9.55 was previously deployment/version verified;
- Windows live: last directly observed desktop was 0.9.54; installed Windows 0.9.55 semantic inspection remained unverified at cutover.

Governance installation:

- mandatory sprint governance and Sprint 0 records were proposed in PR #240;
- `build`, `runner`, and `ui-responsiveness` checks passed on the docs-only PR head;
- PR #240 was merged to `main` at `c4873449c76ac525deace8f08023b142814d05ba`;
- that merge changes governance/documentation only and does not constitute product correction implementation or a new product release.

Current baseline-evidence branch:

- `sprint/0-baseline-evidence`;
- branched from `c4873449c76ac525deace8f08023b142814d05ba`.

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

## Current runtime verification

A bounded canonical read-only Workbench health operation was run against audited source commit `92666ad51f5407d511b89e0e516250a30af04adc` during Sprint 0 baseline establishment. It reported:

- Workbench runner/server/relay binaries present;
- MCP service active;
- relay service active;
- loopback MCP HTTP healthy;
- relay checkout clean;
- overall health OK.

The registered Windows host was also confirmed online through the approved host inventory. Public project documentation intentionally does not record private host identifiers or local paths.

## Initial observation target

Workbench has no conventional website-style DEV deployment. The intended first observable target remains:

**Windows live — installed Workbench desktop, Operations dashboard, exact installed version/build to be verified before the owner gate opens.**

The first baseline observation will focus on the existing Operations semantics corrected in 0.9.55 source/release but not yet inspected on the installed Windows UI:

- terminal completed/failed history must not inflate live Running counts merely because the project/session presence lease is active;
- genuine running work must remain visible;
- presence/session activity must remain separate from job execution state;
- no fabricated queue position or completion percentage may appear where Workbench lacks authority.

This initial observation is for finding existing problems, not for proposing new feature design.

### Observation-gate blocker

The owner observation gate is **NOT OPEN** yet.

The currently advertised approved direct Windows bridge can verify registered-host/tool state but does not expose the installed Workbench desktop executable/version/hash. A bounded read-only supervised operations fallback was requested solely to identify that build, with explicit instructions not to update, install, restart, close, or modify Workbench/source/Git/configuration/user data. At the time this record was updated, it remained running without returning verifiable build evidence.

Because the exact installed review build has not been independently identified, Sprint 0 MUST NOT ask the owner to inspect Windows live yet. This gap is registered as `S0-007` in the Baseline Correction Register rather than weakening the mandatory review-build-identification rule.

## Known starting correction set

The initial register was seeded from `docs/CURRENT_STATE.md` and the governance reset evidence and now contains:

- S0-001 — Windows Operations 0.9.55 installed-target acceptance closure;
- S0-002 — release publication reliability/no-op retrigger dependency;
- S0-003 — unattended continuation end-to-end acceptance;
- S0-004 — private relay retention/compaction governance;
- S0-005 — Blender headless GPU render acceptance;
- S0-006 — Unreal startup/`zen` investigation and acceptance;
- S0-007 — installed Windows Workbench review-build observability required to open the baseline owner gate.

The owner’s baseline observations may add further **existing corrections**. Genuine new requirements are deferred.

## Observation rounds

No owner observation round has yet been completed in this Sprint 0 record because the exact Windows-live build has not yet been independently verified.

## Implementation reference

No product correction implementation has started in Sprint 0.

Governance/sprint-record work began at `15573b831c7c18b8170c960e6832ba7aaf341c73` and became canonical through merged PR #240 at `c4873449c76ac525deace8f08023b142814d05ba`.

## Resulting protected behaviour

Pending Sprint 0 sign-off.

## Deferred issues

Future feature/roadmap work remains deferred until Sprint 0 is **SIGNED OFF**.
