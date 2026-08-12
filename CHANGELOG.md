# Changelog

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
