# Changelog

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
