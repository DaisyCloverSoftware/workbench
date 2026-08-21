# Roadmap

The roadmap is subordinate to current requirements/decisions in `docs/GOVERNANCE.md` and `docs/DECISIONS.md`. Items listed as candidates are not implementation authority.

## v0.5 — autonomous control-plane foundation

- Native standalone Windows application
- Headless runner and loopback-authenticated MCP service
- provider discovery and subscription-first / cheapest-eligible routing
- safe repository eyes plus exact patch and allowlisted command hands
- durable autonomous tasks, retries, attention boundaries and reports
- bidirectional Git relay with status-only public mode and report-capable private mode
- compact continuation context plus project/global durable memory
- automatic worker memory distillation and bounded memory injection into later workers
- versioned reusable routine/code assets with provenance and verification metadata
- cross-process knowledge-store locking and durable state snapshot hardening
- encrypted Windows vault whose raw values are not exposed through model-facing tools
- generic bootstrap/install paths and automated Windows/Linux CI

## v0.6 — isolated review execution and controlled publication

- durable Git worktree isolation per autonomous coding task
- restart recovery for queued/routing/running durable tasks
- bounded changeset inspection and stable fingerprints
- deterministic Workbench-owned review commits and local review branches
- safe publication of prepared review branches without worker push authority
- private local publication policy plus operator-only runner policy controls
- typed out-of-band runner policy synchronisation over SSH
- portable desktop-to-runner repository mapping with symlink-safe runner-root containment
- native Windows controls for prepare-only versus explicit review-branch publication
- isolated-worker memory correctly scoped to the real source repository
- first-class durable parked ideas
- Windows single-instance protection and additional model-safe command hardening
- release packaging for the Windows app and matching Linux runner/server/relay binaries

## v0.6.1 — trustworthy health checks and provider readiness

- deterministic runner self-test separated from external AI-worker availability probing
- valid committed live-worker test fixture with isolated review-commit verification
- filesystem-identity-safe task worktree validation on Windows and path aliases
- persistent host-local provider health telemetry for desktop and one-shot runner processes
- short-lived exponential cooldowns for retryable authentication, quota, permission, adapter and timeout failures
- safe categorical cooldown status in provider UI/runner doctor without persisting raw provider output

## v0.7 — locally initiated verified maintenance

- cache-backed private scratch worktrees instead of relying on system `/tmp` capacity
- official stable-release trust client with exact repository/tag/asset binding and dual SHA-256 verification
- operator-only cluster update check/apply commands kept outside MCP and model-safe hands
- exact cluster archive/ELF validation plus same-filesystem atomic binary staging
- rollback-capable cluster upgrades verified by the new runner selftest and existing Workbench systemd services
- standalone double-clickable Windows updater/installer for a sibling `Workbench.exe`
- Windows PE32+ AMD64 and post-swap checksum verification with rollback on launch failure
- release packaging contract for the Windows app, updater and Linux cluster binaries

## v0.8 — multi-project production workspace and unattended safety

- one production task-first Windows desktop; obsolete dogfood executable and command removed
- durable multi-project registry with pinned/recent ordering, per-project notes/tasks, legacy migration and Windows filesystem-identity path canonicalisation
- background delegation and cross-project notes preserve the human's active project selection
- privacy-minimal multi-project MCP workspace projection for explicit `project_path` targeting without project-selection authority
- cross-project note isolation and unfinished-project removal guard
- fail-closed Windows desktop ownership before durable state opens or interrupted work recovers
- bounded local/remote worker output and durable reports while preserving final attention/unavailable control markers
- durable runner transport stays attached to the same idempotent task across malformed or oversized submit/status responses
- global human-attention navigation and native minimum desktop geometry

## v0.9 — structured harnesses, production UI and resilient unattended routing

- versioned structured harness job/result protocol with bounded JSON stdin/stdout and strict task/version/result validation
- external harness adapters launch as one explicit executable with no coding command shell or `{project}`/`{prompt}` template expansion
- structured completed, human-attention, unavailable and failed states replace prose-marker control flow for compliant adapters
- legacy shell-template coding configuration is retained only as a disabled migration warning and cannot become an eligible worker
- native Windows settings configure a validated structured-adapter executable separately from local OpenClaw and Workbench Runner
- runner hosts keep their own adapter path in private atomic operator config; adapter paths never enter `RunnerRequest`, task state or MCP/model-facing data
- operator-only runner harness get/set/delete controls and safe doctor status without exposing the full host path
- production dark Dashboard, Work and Settings surfaces expose project/task/provider/review state with permanent navigation and top actions
- durable `waiting_retry` tasks automatically resume transient low-cost provider outages after cooldown without human supervision
- durable `waiting_dependency` watches own GitHub Actions waits with progressive backoff, no coding worker held, restart recovery and automatic continuation of the original task
- automatic retries/watches remain cancellable and active unfinished work rather than appearing failed or frozen
- connected Chat/skill guidance distinguishes active-worker checks from Workbench-owned waits so AIs do useful independent work instead of hammering dependency status
- provider-native Claude Code session continuation survives Workbench task retries without exposing private provider session identifiers in task transport
- production Win32 HWND creation/message pumping is pinned to one OS thread; Windows watchdog coverage proves Dashboard/Settings/Work remain responsive and page HWND visibility is correct
- Settings policy reads remain Git/filesystem-free after validated save, including Windows path aliases
- Windows CI captures real Dashboard, Work and Settings windows and packages only the production app plus verified updater; releases include matching Linux runner/server/relay binaries and checksums
- reversible terminal task-history archiving hides filed-away work from default Work/Dashboard views without deleting durable task records or rewriting execution chronology

## Governance freeze — v0.9.54 baseline

Ordinary feature development is frozen until the governance reset completion gate in `docs/GOVERNANCE_RESET_2026-08-21.md` passes.

The current release is not a claim that every v0.9 Operations semantic is accepted. In particular, 0.9.54's projection of session-active completed operations as running jobs is rejected by the canonical Operations contract.

## Post-reset priority backlog

This is an implementation backlog, not permission to start work while the freeze is active.

1. **P0 — Operations semantic correction.** Make actual job execution, project/session presence and terminal activity history distinct; remove the 0.9.54 false-running projection; add semantic acceptance coverage.
2. **P1 — Authoritative cross-plane job model.** Define how scheduler-native tasks, CI, direct server controls, typed Windows work and AI-worker jobs become one truthful inventory without inferring execution from transport recency.
3. **P1 — Unattended continuation live acceptance.** Record a clean end-to-end post-validator proof of `waiting_dependency → automatic resume → useful work → completed`.
4. **P1 — Release publication reliability.** Remove the need for identical-tree no-op `main` pushes as publication retriggers.
5. **P1 — Private relay retention governance.** Define safe retention/compaction/cleanup separately from the bounded live Dashboard projection.
6. **P2 — Blender live acceptance.** Verify a current end-to-end headless GPU render through the typed Windows bridge and record device/backend evidence without leaking private host detail.
7. **P2 — Unreal startup investigation.** Re-verify the bounded smoke and investigate the inherited five-minute `zen` classification without restoring the superseded 90-second test.
8. **P3 — Searchable decisions/knowledge graph.** Re-evaluate the capability against current 0.9.54+ architecture. PR #110 is based on 0.9.10 and MUST NOT be merged as-is merely because the roadmap historically listed the idea.

## Candidate future work

These are planning candidates and require normal decision/specification before implementation:

- automatic cross-model review policies for higher-risk changes;
- complete per-monitor DPI-aware desktop layout/font scaling;
- desktop/tablet/phone preview and test targets;
- richer screenshot/browser result surfaces where appropriate;
- WhatsApp/Signal/Telegram/Slack human-interrupt adapters;
- OS keychain support on macOS/Linux;
- signed installers and optional locally scheduled update checks.

## Later

- Workbench Runner capable enough to replace a general-purpose external harness for many local/cluster workflows;
- distributed runner registry and capability advertisement;
- policy packs for production, regulated environments and team use;
- community adapter registry;
- mobile companion for approvals and read-only task status.
