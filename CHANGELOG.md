# Changelog

## 0.7.0 — 2026-08-15

- Moved Workbench-owned review and relay scratch worktrees out of system `/tmp` and into a private cache-backed scratch area, with `WORKBENCH_SCRATCH_ROOT` available for a deliberately chosen larger volume.
- Added a strict official-release trust client bound to `DaisyCloverSoftware/workbench`: stable-tag validation, exact asset URL binding, bounded downloads, GitHub-declared SHA-256 verification and published checksum verification.
- Added operator-only `workbench-runner update check` and transactional Linux/amd64 `update apply`; update commands remain outside model-safe hands and MCP.
- Cluster self-update accepts only the exact runner/server/relay archive set, verifies ELF64 x86-64, stages replacements on the install filesystem, keeps rollback backups, and verifies the new runner before restarting only previously-active Workbench systemd user services.
- Cluster update failures during new-runner verification or service restart restore the previous binary set rather than leaving a partial upgrade.
- Added a double-clickable `Workbench-Updater.exe` for Windows that can bootstrap or update a sibling `Workbench.exe` from the verified official stable release.
- The Windows updater validates PE32+ AMD64 and SHA-256 before and after the atomic swap, retains the previous executable until the new app successfully launches, and never kills a running Workbench process.
- Windows CI now builds the app and updater together; GitHub releases publish the updater and its checksum alongside the existing Windows app and Linux cluster package.

## 0.6.1 — 2026-08-14

- Split runner verification into a deterministic `selftest` for Workbench control-plane health and an explicit `live-selftest` for external AI-worker availability.
- Fixed the live worker self-test to use a committed Git baseline and verify successful work in a Workbench-owned review commit while the source checkout remains clean.
- Fixed Windows task-worktree validation and cleanup to compare filesystem identity instead of case-sensitive/canonical path strings.
- Added host-local provider health telemetry with short-lived exponential cooldowns so recently unavailable workers are routed around instead of retried on every task.
- Provider cooldown state stores only safe categorical reasons and timestamps; raw worker output, task content, credentials and account identifiers are not persisted.
- Successful provider runs clear their cooldown immediately, non-retryable project failures do not poison provider health, and explicit local Rescan clears cooldowns after operator remediation.

## 0.6.0 — 2026-08-14

- Autonomous coding now runs in durable Workbench-owned Git worktrees, keeping the user's source checkout and active index untouched.
- Added bounded changeset inspection/snapshots, deterministic Workbench review commits, Git-semantic file-mode handling and protected/binary/secret publication guards.
- Added safe publication of prepared `workbench/...` review branches without giving coding workers direct push authority.
- Added private per-project publication policy with operator-only runner configuration, portable desktop-to-runner project mapping and explicit native Windows review-publication settings.
- Publication targets remain outside task state, `RunnerRequest`, MCP/model-facing state, worker prompts and worker memory; optional runner sync uses a separate typed operator SSH channel.
- Isolated workers now read and write durable memory under the real source project rather than ephemeral worktree paths.
- Added restart recovery for interrupted durable tasks, first-class parked ideas, Windows single-instance protection and additional safe-command shell hardening.
- Hardened runner-root containment against symlink escape and SSH-host option/whitespace injection.
- Fixed the asynchronous router-test persistence teardown race exposed by repeated cluster installer/smoke validation.
- GitHub releases now include both the native Windows app and the matching Linux runner/server/relay cluster package with SHA-256 checksums.

## 0.5.0

- Added the bidirectional private Git relay for request/result and genuine-attention flows.
- Added durable project and global memory, compact continuation context, reusable routine/code knowledge, and private Personal Pro memory control.
- Autonomous workers now retrieve relevant memory before work and can distil validated project-scoped knowledge after successful tasks.
- Hardened unattended routing around worker-local permission/setup failures and added safe verification allowances.
- Added concurrency protection for the shared knowledge store and strengthened public-source privacy guards.

## 0.3.1 — 2026-08-12

First native dogfood foundation.

- Added first-class SSH configuration/test path for a remote OpenClaw host, without storing SSH secrets in the app.

- Replaced the browser-hosted prototype with a standalone Win32 desktop application.
- Added cost-aware AI provider discovery and routing.
- Added “protect scarce Work/Codex” default policy.
- Added MCP hands for patch application, safe commands, autonomous task delegation and polling.
- Added persistent task lifecycle and genuine-attention boundary.
- Added Windows DPAPI encrypted vault.
- Added OpenClaw/other-harness command adapter and external human-interrupt hook.
- Added tests proving safe-command policy, secret detection, MCP token protection and lower-cost worker routing before Codex.

### Current provider compatibility note

Google's individual-account terminal path moved from Gemini CLI to Antigravity CLI in 2026. Workbench therefore detects `agy` as the current Google worker and keeps legacy `gemini` as an opt-in enterprise/API compatibility adapter rather than pretending an old free route still exists.
