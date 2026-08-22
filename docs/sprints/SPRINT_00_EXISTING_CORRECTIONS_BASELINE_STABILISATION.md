# Sprint 00 — Existing Corrections & Baseline Stabilisation

Status: **IN DEVELOPMENT — S0-007 BASELINE OBSERVABILITY CORRECTION**

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

Governance installation and current continuation baseline:

- mandatory sprint governance and Sprint 0 records became canonical through merged PR #240 at `c4873449c76ac525deace8f08023b142814d05ba`;
- explicit sprint-continuity governance became canonical through merged PR #247 at `949dc578ba63cade772514054adb6b57836b8797`;
- during Sprint 0 branch setup, an accidental one-byte temporary marker was committed directly to `main` and immediately removed by repair commit `a13cb8be601752e281bd5145ddcfb2ab57b9f70a`; the repaired tree matches the pre-incident `949dc578ba63cade772514054adb6b57836b8797` tree and no product/runtime behaviour changed;
- current correction branch: `sprint/0-windows-build-identity`, branched from repaired canonical `main` at `a13cb8be601752e281bd5145ddcfb2ab57b9f70a`.

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

### Observation-gate blocker and S0-007 correction

The owner observation gate is **NOT OPEN** yet.

The approved direct Windows bridge can verify registered-host/tool state but existing installed builds do not expose the running Workbench desktop executable identity. The bounded read-only supervised fallback previously requested for this purpose has now reached a terminal failure without producing build evidence. Retrying that unsuitable worker path is not accepted as baseline proof.

`S0-007` is therefore in development on `sprint/0-windows-build-identity`. The correction adds a bounded, path-free Workbench capability to the existing outbound Windows heartbeat. It reports the canonical product version plus SHA-256 and byte size of the running Workbench executable. Unknown heartbeat capabilities remain filtered, and the Workbench identity capability is not accepted by the executable host-job allowlist, so this observability correction does not add generic command or publication/deployment authority.

The gate remains closed until the correction passes exact-candidate engineering verification, is released/installed through the normal Workbench path, and the Windows-live heartbeat independently identifies the running candidate with release-equivalent evidence.

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

Sprint 0 product correction implementation has started with `S0-007` on branch `sprint/0-windows-build-identity` from canonical `main` repair commit `a13cb8be601752e281bd5145ddcfb2ab57b9f70a`.

The implementation is deliberately limited to baseline observability: a bounded Workbench version+SHA-256 identity in the existing outbound Windows heartbeat plus regression tests preserving the typed/allowlisted security boundary. No Sprint 1 or unrelated Sprint 0 correction is included in this changeset.

## Resulting protected behaviour

Pending Sprint 0 sign-off.

## Deferred issues

Future feature/roadmap work remains deferred until Sprint 0 is **SIGNED OFF**.
