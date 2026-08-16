# Changelog

## 0.9.1 — 2026-08-16

- Added first-class `runner://<repo>` cluster projects so repositories can stay on the Workbench runner instead of being copied, mounted or duplicated on the Windows desktop.
- Added bounded runner project discovery under the authorised runner root and routed ChatGPT MCP safe eyes/hands (`list_files`, `search_text`, `read_file`, `apply_patch`, `run_safe_command`) through the fixed runner JSON/SSH protocol for cluster projects.
- Cluster-only autonomous coding now remains on the Workbench Runner execution host; local coding CLIs are not handed nonexistent remote paths and project-location refusal no longer poisons provider health/cooldowns.
- Add Project now offers configured cluster repositories as well as local folders and can invoke the runner's existing verified self-update path when an older runner protocol needs upgrading before discovery.
- Review & Publish policy for cluster projects is owned by the runner, with only a private desktop operator mirror retained for fast Settings display and verified GitHub review links; review retry remains runner-side.
- Fixed Dashboard provider health so ChatGPT readiness reflects the actual live local MCP listener. When the bridge is online, Dashboard now reports `ChatGPT Chat — Ready` instead of contradicting Settings.
- `waiting_dependency` now counts as active/unfinished work consistently in Dashboard and project-removal safety.
- Claude Code setup feedback now explains the local Windows CLI installation/login path instead of stopping at `not detected`; cluster projects continue to execute through the runner.
- Retained the v0.9 Win32 OS-thread ownership fix and 32-cycle responsiveness/visibility soak while adding the cluster-first workflow.

## 0.9.0 — 2026-08-16

- Added a coherent production dark Dashboard, Work and Settings experience backed by real Workbench state, with permanent navigation/top actions, truthful task/project/provider status and full-window Windows screenshot evidence.
- Added the versioned structured harness job/result protocol and private runner-host adapter configuration, keeping structured harnesses separate from local OpenClaw and eliminating shell-template coding commands from the eligible worker path.
- Added durable `waiting_retry` and `waiting_dependency` task states: transient provider cooldowns and GitHub Actions dependencies back off without holding a coding worker, survive Workbench restarts, remain cancellable, and automatically resume the original task when ready.
- GitHub Actions dependency watches use progressive polling/backoff instead of status hammering; connected AI guidance now tells clients to do other independent work during Workbench-owned waits and never claim to be monitoring unless a durable watch exists.
- Added provider-native Claude Code session continuation while keeping provider session identifiers private and outside task transport/model-facing state.
- Fixed the production Windows `Not Responding` failure by pinning the complete HWND create/message-pump/destroy lifetime to its owning OS thread with `runtime.LockOSThread()`.
- Added a real Windows responsiveness/visibility gate that soaks 32 Dashboard → Settings → Work cycles with an active Git project and fails if the message pump does not answer within two seconds or hidden page HWNDs remain visible.
- Kept Settings responsive by avoiding live Git/filesystem probing for policy reads, caching repeated provider/vault/policy materialisation, and persisting Windows short/long path aliases at policy-save time.
- Hardened production screenshot capture so Dashboard, Work and Settings are captured from fresh running processes with isolated fixture data, normal whole-window `PrintWindow` semantics and no prior-page backing-surface residue.
- Windows CI continues to package only `Workbench.exe` and `Workbench-Updater.exe`; the release workflow also publishes the matching Linux runner/server/relay cluster archive and SHA-256 checksums.

## 0.8.0 — 2026-08-15

- Replaced the remaining prototype/dogfood desktop split with one production `Workbench.exe`: a task-first multi-project workspace plus explicit Settings for workers, MCP, runner/harness configuration, review policy, encrypted vault and verified maintenance.
- Added a durable first-class project registry with pinned/recent ordering, per-project notes and task views, legacy-state migration, Windows filesystem-identity canonicalisation, and safeguards against project duplication through short-path/symlink aliases.
- Background delegation and cross-project notes no longer steal the human's active desktop project; MCP `get_workspace` exposes only privacy-minimal registered-project routing facts so connected chat can target `project_path` explicitly.
- Project removal now refuses to hide queued, routing, running or needs-attention work while preserving terminal task history after explicit removal.
- Hardened production desktop lifecycle ownership: the per-user Windows mutex is required before durable state opens or interrupted work recovers, eliminating a rare fail-open concurrent-state path.
- Bounded local worker stdout/stderr, custom-harness capture, Ollama replies, runner responses and SSH transport; durable reports are capped separately while preserving prefix and rolling tail so final `ATTENTION_REQUIRED` / `WORKER_UNAVAILABLE` control markers remain detectable.
- Durable runner transport now treats malformed or oversized submit/status replies as ambiguous transport state and stays attached to the same idempotent task ID rather than routing a second coder onto the repository.
- Added global human-attention navigation across projects and a native minimum window geometry so required answer/review controls cannot be resized off-screen.
- Removed `Workbench-Dogfood.exe` from production packaging and deleted the obsolete dogfood command; Windows CI now produces only the production app and verified updater.

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