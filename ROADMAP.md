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

## Next

- first-class structured OpenClaw/harness jobs rather than a command-template adapter
- richer retry/recovery policies and resumable provider sessions
- searchable decisions and a project knowledge graph
- automatic cross-model review policies for higher-risk changes
- task-history filtering/archiving without deleting durable task records
- complete per-monitor DPI-aware desktop layout/font scaling
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
