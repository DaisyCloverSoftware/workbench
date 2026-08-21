# Roadmap

The roadmap is subordinate to current requirements/decisions in `docs/GOVERNANCE.md` and `docs/DECISIONS.md`. Items listed as candidates are not implementation authority. Current implementation/verification status lives in `docs/CURRENT_STATE.md`.

## Delivered foundations through v0.9

Workbench has delivered the following broad foundations across v0.5–v0.9:

- native Windows application plus Linux/cluster runner, loopback-authenticated MCP and private relay;
- provider-neutral capability/trust/cost routing with local/included routes preferred and scarce/metered routes protected;
- safe repository eyes/hands, durable task state, retries, attention boundaries, worktree isolation and controlled publication;
- persistent project/global knowledge, continuation capsules, reusable routines/assets and encrypted local secret storage;
- verified Windows updater and rollback-capable cluster maintenance;
- multi-project production workspace and privacy-minimal runner/project references;
- structured harness protocol and explicit adapter boundaries;
- scheduler-owned durable queueing, priority/FIFO ordering, truthful measured/stage/indeterminate progress and six Operations lanes;
- durable retry/dependency waits and authenticated private continuation;
- production Dashboard/Work/Settings Win32 surfaces with Windows responsiveness/screenshot evidence;
- bounded relay live projection so historical transport volume does not define current Operations state.

Detailed release history remains in `CHANGELOG.md`.

## v0.9.55 — Operations semantic correction

Source correction delivered in PR #232 and released through PR #233:

- terminal `completed`/`failed` relay history no longer becomes Running merely because the surrounding project/session presence lease is active;
- genuine remote running/queued/routing/waiting/attention state is projected from the event state itself;
- session presence remains separately labelled context;
- regression coverage includes completed+active, failed+active and the observed 100-terminal-history failure shape.

This is **implemented/tested/merged/released**, but the actual installed Windows 0.9.55 Dashboard still requires user-visible semantic inspection/sign-off before the corrections round is accepted.

## Initial Correction Sprint — must complete before feature development

1. **P0 — Windows Operations 0.9.55 acceptance closure.** Establish the actual Windows live 0.9.55 surface; inspect real Operations state; prove terminal history/session presence does not inflate live job counts; record and execute any observation-driven corrections; obtain user sign-off.
2. **P1 — Release publication reliability.** Remove the need for identical-tree/no-op `main` pushes as publication retriggers.
3. **P1 — Unattended continuation live acceptance.** Produce a clean end-to-end post-validator proof of `waiting_dependency → automatic resume → useful work → completed`.
4. **P1 — Private relay retention governance.** Define safe retention/compaction/cleanup separately from the bounded live Dashboard projection.
5. **P2 — Blender live acceptance.** Verify a current end-to-end headless GPU render through the typed Windows bridge and record privacy-safe backend/device evidence.
6. **P2 — Unreal startup investigation.** Re-verify the five-minute smoke and investigate the inherited `zen` classification without restoring the superseded 90-second test.

These are corrections/verification gaps, not new feature scope.

## Approved architecture roadmap after correction sign-off

### Authoritative cross-plane job model

Define how scheduler-native tasks, CI, direct server controls, typed Windows work and AI-worker jobs become one truthful Operations inventory without inferring execution from transport recency. Preserve the distinction between execution state, session/project presence and terminal history.

This is approved roadmap direction, but implementation must not begin until the current Correction Sprint has an inspectable result and user sign-off.

## Ideas / product decisions requiring fresh design or approval

- searchable decisions / knowledge graph capability: re-evaluate against current architecture; old PR #110 is based on 0.9.10 and must not be revived as-is;
- interactive drag/reorder beyond current persisted priority/FIFO;
- richer selected-job drawer controls/details beyond current explicit contracts;
- exact four-hour session-presence lease value;
- whether cross-process relay-state locking is required under the supported topology;
- automatic cross-model review policies for higher-risk changes;
- complete per-monitor DPI-aware desktop scaling;
- desktop/tablet/phone preview/test targets;
- richer screenshot/browser result surfaces where appropriate;
- human-interrupt adapters such as WhatsApp/Signal/Telegram/Slack;
- OS keychain support on macOS/Linux;
- signed installers and optional locally scheduled update checks.

Ideas are not requirements until canonically approved.

## Later possibilities

- Workbench Runner capable enough to replace a general-purpose external harness for many local/cluster workflows;
- distributed runner registry and capability advertisement;
- policy packs for production, regulated environments and team use;
- community adapter registry;
- mobile companion for approvals/read-only task status.
