# Changelog

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
