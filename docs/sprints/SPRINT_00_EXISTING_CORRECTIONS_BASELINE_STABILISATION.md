# Sprint 00 — Existing Corrections & Baseline Stabilisation

Status: **DEPLOYING TO REVIEW TARGET**

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

Governance installation and continuation baseline:

- mandatory sprint governance and Sprint 0 records became canonical through merged PR #240 at `c4873449c76ac525deace8f08023b142814d05ba`;
- explicit sprint-continuity governance became canonical through merged PR #247 at `949dc578ba63cade772514054adb6b57836b8797`;
- during Sprint 0 branch setup, an accidental one-byte temporary marker was committed directly to `main` and immediately removed by repair commit `a13cb8be601752e281bd5145ddcfb2ab57b9f70a`; the repaired tree matched the pre-incident `949dc578ba63cade772514054adb6b57836b8797` tree and no product/runtime behaviour changed.

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

## S0-007 implementation and engineering verification

`S0-007` was implemented through PR #248 and merged at `12fa23cf662ac89ae7f57815e9a5472b2fac665e`.

The correction adds a bounded, path-free `workbench` capability to the existing outbound Windows heartbeat. The running desktop reports:

- canonical Workbench version;
- SHA-256 of the running executable;
- executable byte size.

The correction deliberately does **not** add Workbench executable-job authority. Unknown heartbeat capabilities remain filtered, and the host-job allowlist still rejects Workbench as an executable local tool.

Exact-candidate verification on PR #248 completed successfully:

- `build`: passed;
- `runner`: passed;
- `ui-responsiveness`: passed;
- production Windows build/page-capture stages passed on the exact candidate.

## Stable release 0.9.56

The coordinated release request prepared `v0.9.56`; release PR #249 was merged at `bf83b88851e854b572bec69143bb3b33e97a404f`.

The release candidate's `runner` workflow initially hit a transient relay-test worktree race during a second installer test pass. The first full `go test ./...` pass had succeeded, the failing job was rerun unchanged, and the rerun passed. `build` and `ui-responsiveness` also passed. No source workaround or no-op `main` retrigger was used to obtain those checks.

The stable `v0.9.56` tag subsequently became available. This is positive evidence for `S0-002`, but does not by itself close the known publication-reliability correction.

Current canonical `main` is `3e6c2d702c2f67872bb3c3d547a05fac4bdfbf8c`, which descends from the 0.9.56 release merge. Its subsequent PR #250 adds product-scoped operations scripts for another project and does not alter Workbench 0.9.56 version surfaces or the Sprint 0 desktop candidate.

## Cluster-live deployment evidence

The first private self-update attempt to move the Workbench cluster control plane to 0.9.56 failed closed before installation because the private relay checkout contained an unexpected local change.

Bounded read-only inspection established that the dirty set contained exactly one untracked relay control JSON and zero tracked changes. The same request existed canonically on relay `origin/main`, and its corresponding outbox was already completed. A one-time Sprint 0 operation then quarantined only that exact redundant canonical copy after verifying its blob matched `origin/main`; it failed closed if any other dirty state was present.

This deployment inconsistency is registered as `S0-008`. The operational remediation does not count as a durable source fix for recurrence.

After the guarded quarantine:

- the relay checkout verified clean;
- the existing `update_workbench` mechanism was retried unchanged;
- updater status reached `succeeded`;
- the private capability manifest independently advertised `workbench_version: 0.9.56`;
- a post-deploy `workbench-health.sh` run at then-current canonical `main` reported runner/server/relay binaries present, MCP active, relay active, loopback MCP HTTP healthy, relay checkout clean, and `overall=ok`.

Cluster live is therefore verified at Workbench 0.9.56 for the control-plane portion required by `S0-007`.

## Initial observation target

Workbench has no conventional website-style DEV deployment. The intended first observable target remains:

**Windows live — installed Workbench desktop, Operations dashboard, exact installed version/build verified before the owner observation gate opens.**

The first baseline observation will focus on the existing Operations semantics corrected in 0.9.55 and preserved in 0.9.56:

- terminal completed/failed history must not inflate live Running counts merely because the project/session presence lease is active;
- genuine running work must remain visible;
- presence/session activity must remain separate from job execution state;
- no fabricated queue position or completion percentage may appear where Workbench lacks authority.

This initial observation is for finding existing problems, not for proposing new feature design.

### Windows-live deployment state and human-only authority boundary

The owner observation gate is **NOT OPEN** yet.

After cluster live reached 0.9.56, the registered Windows host was online but its heartbeat still advertised only the existing Blender/Unreal capabilities. It did **not** advertise the new `workbench` identity capability. That is positive evidence that Windows live has not yet reached/restarted on the corrected 0.9.56 desktop candidate; Workbench therefore MUST NOT ask the owner to perform product observation yet.

The canonical Windows update entry point is the installed desktop's **Settings → Maintenance → `Check / install verified update`** control. It launches the separate verified updater. The updater downloads and checksum-validates the official stable Workbench release and Windows executable before presenting its explicit install/update confirmation. The update path then closes the owned Workbench window, transactionally replaces the executable, relaunches Workbench, and rolls back if the new executable cannot launch.

The current approved direct Windows bridge intentionally has no generic command or remote desktop-update action. Therefore initiating and confirming this desktop updater is a genuine human-only UI/authority boundary, permitted as an early owner interruption by sprint governance. It is not the owner observation gate and does not constitute Sprint 0 sign-off.

After the owner completes that verified updater interaction, Development must continue autonomously by re-reading the Windows heartbeat, verifying the reported Workbench version/SHA-256 against the released candidate, and only then moving to **READY FOR OWNER OBSERVATION / AWAITING OWNER OBSERVATION**.

## Known correction set

The Baseline Correction Register currently contains:

- S0-001 — Windows Operations installed-target acceptance closure;
- S0-002 — release publication reliability/no-op retrigger dependency;
- S0-003 — unattended continuation end-to-end acceptance;
- S0-004 — private relay retention/compaction governance;
- S0-005 — Blender headless GPU render acceptance;
- S0-006 — Unreal startup/`zen` investigation and acceptance;
- S0-007 — installed Windows Workbench review-build observability;
- S0-008 — private self-update deployment inconsistency caused by a redundant local canonical relay-control copy.

The owner’s baseline observations may add further **existing corrections**. Genuine new requirements are deferred.

## Observation rounds

No owner product-observation round has yet been completed in Sprint 0 because Windows live has not yet been independently verified as the corrected 0.9.56 review candidate.

## Implementation references

- S0-007 implementation PR #248 → merge `12fa23cf662ac89ae7f57815e9a5472b2fac665e`.
- Workbench 0.9.56 release PR #249 → merge `bf83b88851e854b572bec69143bb3b33e97a404f` → stable tag `v0.9.56`.
- Cluster-live 0.9.56 maintenance retry → terminal updater state `succeeded`, manifest version 0.9.56, post-deploy health `overall=ok`.
- S0-008 one-time operational remediation used bounded committed operations from a temporary Sprint 0 diagnostic branch and is not a substitute for a durable recurrence fix.

## Resulting protected behaviour

Pending Sprint 0 sign-off.

## Deferred issues

Future feature/roadmap work remains deferred until Sprint 0 is **SIGNED OFF**.
