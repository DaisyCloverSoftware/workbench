# Roadmap

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

## Next

- first-class structured OpenClaw/harness jobs rather than a command-template adapter
- richer retry/recovery policies and resumable provider sessions
- searchable decisions and a project knowledge graph
- automatic cross-model review policies for higher-risk changes
- desktop/tablet/phone preview and test targets
- screenshot and Playwright result surfaces
- WhatsApp/Signal/Telegram/Slack adapters as human-interrupt channels
- OS keychain support on macOS/Linux
- signed installers and optional locally scheduled update checks

## Later

- Workbench Runner capable enough to replace a general-purpose external harness for many local/cluster workflows
- distributed runner registry and capability advertisement
- policy packs for production, regulated environments and team use
- community adapter registry
- mobile companion for approvals and read-only task status
