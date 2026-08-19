# Changelog

## 0.9.29 — 2026-08-19

- Make Workbench transport plan-aware so project chats use the private Git relay as the preferred write/mutation path whenever full custom MCP write actions are unavailable.
- Publish machine-readable preferred write transport, MCP role, zero-external-model-credit and fresh-chat bootstrap metadata in WORKBENCH_CAPABILITIES.json.
- Update shared integration and ChatGPT skills so Secure MCP Tunnel remains optional for personal-plan writes while full-MCP-capable workspaces can use the same Workbench tools directly.

## 0.9.28 — 2026-08-19

- Make direct Kubernetes access work on the K3s control host while preserving the existing root-owned kubeconfig boundary.
- After Workbench validates an allowlisted kubectl program-plus-argv request, retry only the specific K3s kubeconfig permission failure through the cluster's existing non-interactive sudo -n k3s kubectl path.
- Keep sudo completely outside the public Workbench command surface and add regression proof that generic kubectl failures never trigger elevation and that the validated argv is preserved exactly.

## 0.9.27 — 2026-08-19

- Make routine host and cluster operations a direct Workbench execution path owned by ChatGPT, with no OpenClaw, Claude, Codex, Work-credit or other external AI-worker requirement.
- Add separate inspect_machine and run_machine_command tools using structured program-plus-argv execution with no shell, bounded time/output, explicit MCP authority annotations and matching private-relay controls.
- Harden direct machine access with executable/subcommand allowlists, credential and alternate-target blocking, Kubernetes Secret protection, bounded logging/watch behavior, destructive primitive refusal and secret-like output withholding.
- Keep repository run_safe_command isolated from deployment authority and demote OpenClaw to optional autonomous fallback capacity only when the direct structured command surface cannot express the remaining machine-side outcome.
- Fix Secure MCP Tunnel client installation to resolve OpenAI's current stable release metadata, download the published versioned archive from the persistent public artifact bucket, verify SHA-256 checksums and reject placeholder tunnel IDs before network work.

## 0.9.26 — 2026-08-19

- Allow OpenClaw quota/context-capacity failures reported through WorkerUnavailable to continue through Workbench's model fallback chain while preserving genuine worker-local unavailability as authoritative.
- Add the shared ChatGPT Workbench integration package so ordinary project chats can use one reusable MCP connection for bounded repository hands and supervised machine-side operations.
- Update Secure MCP Tunnel installation to the current named-profile flow, keep runtime credentials in local file references, and automatically refresh obsolete tunnel-client binaries.
- Add POSIX and Windows personal plugin packagers that inject the workspace-specific plugin_asdk_app binding only on the user's machine, with CI coverage for both packaging paths.

## 0.9.25 — 2026-08-19

- Treat OpenClaw model context overflow and prompt-too-large failures as model-route capacity problems so Workbench can fail over instead of cooling and repeating the same unusable route.
- When an OpenAI/Codex operations route is exhausted or too small for the job context, prefer an available Claude model in the same job-scoped OpenClaw conversation before falling back to local Ollama.
- Add end-to-end regression coverage proving context overflow can move to Claude Sonnet, preserve the same Workbench job conversation and reach verified completion without human nudging.

## 0.9.24 — 2026-08-19

- Give every durable Workbench machine-operation job its own fresh OpenClaw conversation, reuse that conversation across supervised continuation, retries, attention resumes and model failover, and best-effort archive it after verified completion.
- When an OpenAI/Codex operations route reaches a genuine usage, quota, rate-limit or capacity ceiling, prefer an available Claude model (Sonnet first) in the same job conversation before falling back to local Ollama.
- Keep project chats independent of OpenClaw state: different Workbench jobs get different conversations and never inherit the long-lived interactive main session's stale bindings or history.

## 0.9.23 — 2026-08-18

- Expose private read-only get_task and list_tasks controls so ChatGPT can diagnose unusually long supervised machine operations without asking the human to watch or intervene.
- Keep task diagnostics strictly non-mutating: they accept no project scope, cannot cancel/resume/schedule work, and their results remain protected by the private relay secret-like result guard.
- Publish the diagnostic controls in Workbench capabilities and the canonical ChatGPT bootstrap so fresh chats can use them as troubleshooting/status tools while ChatGPT remains the developer.

## 0.9.22 — 2026-08-18

- Keep machine-side operations available when OpenClaw's primary cloud model reaches a usage, quota, rate-limit or capacity ceiling by retrying through a suitable local Ollama model when one is available.
- Prefer an explicitly configured WORKBENCH_OPENCLAW_OPERATION_FALLBACK_MODEL, otherwise detect the OpenClaw host's local Ollama inventory and prefer Qwen Coder-class models without exposing credentials or model files.
- Preserve strict failure boundaries: authentication, genuine human attention and explicit worker-local unavailability are never hidden by model fallback, and ChatGPT remains the owner of source code, Git/GitHub and CI.

## 0.9.21 — 2026-08-18

- Add an app-first await_operation bridge so ChatGPT can hand off machine-side work and let Workbench own the bounded wait instead of polling or asking the human to keep watching.
- Keep durable operations running across wait expiry, return immediately for completion/failure/attention, and reject development tasks so OpenClaw cannot become a second coding path.
- Mark machine-side delegation as permission-worthy open-world work while preserving ChatGPT ownership of source code, Git/GitHub, pull requests, CI and GitHub Actions.

## 0.9.20 — 2026-08-18

- Target the canonical main OpenClaw agent explicitly for scripted machine-side operations instead of relying on an interactive session target.
- Fix the real end-to-end bridge failure where OpenClaw required --to, --session-id or --agent before accepting Workbench operations.
- Preserve ChatGPT ownership of source code, Git/GitHub, pull requests, CI and GitHub Actions; OpenClaw remains machine-side operations only.

## 0.9.19 — 2026-08-18

- Allow machine-side operations to run even when the canonical repository has unrelated local edits by using a disposable worktree pinned to committed HEAD.
- Keep uncommitted user edits out of the OpenClaw workspace while preserving the existing rejection/discard boundary for any source changes OpenClaw attempts.
- Keep autonomous coding workspaces fail-closed on dirty repositories; the relaxed isolation path applies only to machine operations.

## 0.9.18 — 2026-08-18

- Make machine-operation failures self-diagnosing for ChatGPT by surfacing bounded non-secret OpenClaw failure output instead of generic worker-failed messages.
- Keep operations routing isolated to OpenClaw or the Workbench runner so machine-side work cannot fall through to coding workers.
- Preserve ChatGPT ownership of source code, Git/GitHub, pull requests, CI and GitHub Actions while improving the OpenClaw bridge.

## 0.9.17 — 2026-08-18

- Make supervised OpenClaw machine operations runnable from the Workbench systemd service environment by restoring bounded user-level executable and Node interpreter directories to PATH.
- Support common npm, pnpm, NVM, FNM, Volta, Bun and local-bin OpenClaw installations without depending on an interactive shell PATH.
- Preserve the ownership boundary: ChatGPT owns code, Git/GitHub, pull requests, CI and GitHub Actions; OpenClaw remains machine-side operations only.

## 0.9.16 — 2026-08-18

- Fix the supervised OpenClaw operations handoff to use the current scripted `openclaw agent --message` CLI contract.
- Remove the invalid browser-only `--headless` flag that caused machine-side operations to fail immediately after reaching OpenClaw.
- Preserve the strict ownership boundary: ChatGPT owns code, Git/GitHub, pull requests, CI and GitHub Actions; OpenClaw remains the machine-side operator only.

## 0.9.15 — 2026-08-18

- Repair the machine-operations handoff by discovering OpenClaw from common user-level install locations when systemd does not inherit the interactive shell PATH.
- Keep ChatGPT as the developer while making OpenClaw available to Workbench only as the machine-side operations harness.
- Add regression coverage for service-safe OpenClaw discovery and provider scanning.

## 0.9.14 — 2026-08-18

- Make ChatGPT own the software-development loop: source code, Git/GitHub, pull requests, CI, GitHub Actions and release orchestration stay with ChatGPT.
- Restrict Workbench/OpenClaw handoff to machine-side operations ChatGPT cannot execute itself, including shell, systemd, Docker, Kubernetes, Helm, runner-host repair and deployment/runtime operations.
- Automatically re-engage OpenClaw when a machine operation stops with progress rather than verified completion, while rejecting source/IaC/GitHub/CI work and preserving genuine human approval boundaries.

## 0.9.13 — 2026-08-18

- Show the active workload by recency, keep the full bounded active set, and explicitly indicate when additional active tasks are hidden from the compact dashboard panel.
- Retry temporary GitHub DNS and network failures during update discovery and report a non-destructive update-check-unavailable warning when the current installation has not changed.

## 0.9.12 — 2026-08-18

- Compute ChatGPT active-session state on the runner clock and return it explicitly to the desktop, eliminating desktop clock/skew from active-task visibility.
- Keep Recent activity and Active tasks consistent by showing runner-active ChatGPT events as Working while preserving persisted autonomous task state.

## 0.9.11 — 2026-08-18

- Keep ordinary ChatGPT Workbench sessions visible as active across unattended work blocks instead of expiring after 45 minutes.
- Use persisted autonomous task state for delegated work so completed jobs do not remain falsely active while long-running jobs stay visible.

## 0.9.10 — 2026-08-18

- Add reversible terminal task-history archiving without deleting durable task records or rewriting execution chronology.
- Hide archived work from the default Work task list, Dashboard totals, recent activity, provider activity and project summaries so filed-away history does not bury live work.
- Add explicit Show archived and Archive/Restore controls on the production Work page, with archived rows clearly labelled when revealed.
- Fail closed when archiving active work: queued, routing, running, retry, dependency and human-attention tasks cannot be hidden.
- Keep archived completed reviews out of the global Review & Publish shortcut until the task is explicitly restored.

## 0.9.9 — 2026-08-18

- Add a one-click Copy ChatGPT bootstrap action to production Settings so fresh conversations can discover and use the private Workbench relay without the user reconstructing setup instructions.
- Keep the ChatGPT bootstrap credential-free and separate from Copy MCP connection, whose clipboard payload can contain a local bearer credential.
- Tell fresh ChatGPT conversations to read WORKBENCH_CAPABILITIES.json and WORKBENCH_CHATGPT.md, keep ChatGPT as the primary brain/coder, prefer safe hands, protect scarce Codex/Work and leave metered fallback opt-in.
- Emit literal <id> relay protocol paths in WORKBENCH_CAPABILITIES.json instead of JSON HTML escapes while preserving valid machine-readable JSON.
- Verify the production Settings control is created, owner-drawn, page-scoped and wired to the safe bootstrap clipboard action.

## 0.9.8 — 2026-08-18

- Redact implicit runner host filesystem paths from private ChatGPT control results while preserving opaque project refs and useful source, memory and context output.
- Publish WORKBENCH_CAPABILITIES.json beside the private relay guide so fresh ChatGPT conversations can discover the current Workbench protocol deterministically.
- Keep the human-readable ChatGPT bootstrap aligned with machine-readable capabilities and exact multi-root runner project references.
- Pin Workbench GitHub Actions to immutable commit SHAs and enforce that supply-chain contract in repository tests.
- Retain the 0.9.7 multi-root cluster project discovery path across conventional src and projects roots with fail-closed duplicate-name scoping.

## 0.9.7 — 2026-08-17

- Expanded cluster repository discovery and execution across multiple authorised runner roots while preserving legacy WORKBENCH_RUNNER_ROOT and adding WORKBENCH_RUNNER_ROOTS for explicit multi-root hosts.
- Recognised conventional ~/src and ~/projects roots by default when present, and removed the old MCP/relay systemd installer pin that forced live services back to ~/src.
- Kept unique repositories on backwards-compatible runner://name references while duplicate directory names fail closed and receive stable opaque runner://rN/name refs without exposing host paths.
- Preserved scoped runner refs as distinct desktop project identities and kept absolute-path and symlink containment checks fail-closed across every authorised root.
- Extended the private ChatGPT safe-hands and autonomous relay paths to accept exact discovered runner refs, and updated the self-describing guide to tell ChatGPT to reuse those refs instead of guessing duplicate repositories.
- Added cross-platform tests for multi-root resolution/discovery, stable optional-root slot numbering, scoped registry identity, relay resolution, symlink escape, and installer contracts; exact-head Windows build, Linux runner/MCP/relay, and UI-responsiveness gates passed before merge.

## 0.9.6 — 2026-08-17

- Pinned ChatGPT's PRIMARY role outside the scrollable Settings worker inventory so asynchronous runner/model refreshes can never make Workbench look autonomous-worker-first again.
- Made the private Git relay self-describing with a canonical non-secret WORKBENCH_CHATGPT.md bootstrap that ordinary ChatGPT conversations can read to use safe repository eyes/hands, memory/context, autonomous escalation and genuine-attention flows without manual prompt shuttling.
- Made fresh/open-chat bootstrap deterministic by directing connected ChatGPT sessions to repository-search the private workbench-relay repository instead of depending on private code-search indexing.
- Dogfooded the self-describing private loop end to end: live maintenance update succeeded, the private bootstrap file matched the canonical public source exactly, and post-update list_projects safe hands returned all configured cluster repositories.
- Added tested one-shot release-request automation so future versions coordinate every binary/plugin version advertisement and CHANGELOG entry without fragile manual multi-file edits; the workflow retains existing guarded merge/release authority and pins its third-party actions to immutable SHAs.

## 0.9.5 — 2026-08-17

- Restored the product's Chat-first architecture in the native UI: ordinary ChatGPT is the explicit primary Workbench brain, autonomous coding workers are escalation capacity, and scarce Codex/Work is visibly last-resort rather than the default coding route.
- Extended the verified private Git relay with bounded safe hands for Personal-plan workflows: privacy-safe repository discovery plus `list_files`, `search_text`, `read_file`, exact `apply_patch`, allowlisted `run_safe_command`, and non-secret `save_note`, all using the same Workbench policy boundaries as direct MCP.
- Kept autonomous `delegate_task` and genuine human-attention answers on separate relay channels, with no generic shell, arbitrary SSH, direct push/deploy, or new publication authority exposed to Chat.
- Hardened private safe-hands project confinement by canonicalising repository roots and rejecting symlink escape or aliasing outside the authorised runner root.
- Reworked private-loop maintenance to use an app-owned disposable update checkout instead of resetting, cleaning, switching, or fast-forwarding the developer project checkout; existing local development work is left untouched.
- Added privacy-minimal `update_status` state (`scheduled`, `running`, `succeeded`, `failed`) so Chat can verify maintenance completion without asking the human to inspect systemd or journal output.
- Fixed live-host self-update reliability by isolating update-source tests from the operator HOME and allowing up to 30 seconds for the restarted MCP service to become healthy before authenticated initialize verification.
- Dogfooded the complete private Chat-first loop before release: update status, project discovery, model-safe repository read, allowlisted command execution, and a full self-update all completed through the private relay while the developer checkout retained the same pre-existing working-tree changes.

## 0.9.4 — 2026-08-17

- Fixed Windows runner connection setup so Workbench opens a real persistent `cmd.exe /K` console in its own Windows console process instead of relying on an intermediate `START` hand-off that could disappear before SSH/Tailscale output was readable.
- Added a second-stage OpenClaw cloud-model router without changing Workbench's existing outer provider hierarchy: local/cheap workers and the established provider order still get first refusal, and cloud model selection only begins if routing reaches runner-host OpenClaw.
- Added dynamic discovery of OpenClaw's currently allowed/available OpenAI and Anthropic models, including safe provider, input/image, context, default and short-lived cooldown metadata where available; future OAuth catalogue changes do not require a fixed Workbench model list.
- Routine OpenClaw cloud work follows OpenClaw's resolved global default, so a plentiful model such as GPT-5.3 Codex Spark can be preferred without hard-coding it into Workbench. Difficult/high-risk cloud work can start higher in the live ladder, including GPT-5.6 Sol or capable Claude models when available.
- Added bounded cross-model and cross-provider fallback plus per-model cooldowns, so a rate-limited/unavailable cloud model does not immediately poison the entire OpenClaw provider.
- Added native Settings rows for discovered OpenClaw cloud models. Selecting one exposes **Set model default**, validates it against the live catalogue, invokes only the fixed operator-side runner action, and verifies OpenClaw's resolved default after the change.
- Added optional per-task `cloud_model` overrides through the Workbench task/MCP contract. The override is passed only to the OpenClaw runner shim, never inserted into the worker prompt, never selects OpenClaw itself, and cannot bypass the existing local/cheaper/provider routing hierarchy.
- Stale, removed or cooling per-task model overrides fail safely back to the live automatic cloud ladder instead of making an otherwise runnable task fail.
- Kept cloud model discovery optional and bounded so a slow/unhealthy OpenClaw catalogue cannot make ordinary runner worker inventory disappear from Settings.
- Preserved privacy and command-safety boundaries: model inventory excludes account/auth/token/raw usage details, cloud default changes remain operator-only, arbitrary OpenClaw flags are rejected, and cloud controls remain outside model-safe command execution.

## 0.9.3 — 2026-08-16

- Made durable `waiting_dependency` tasks behave like real active work in the native Work page: they are auto-selected and the Cancel button remains enabled, matching the engine's existing cancellable durable-wait semantics.
- Replaced all-or-nothing cluster-project import with a native cluster repository chooser. Add Project can now discover runner repositories and let the user choose **Add selected**, **Add all**, double-click one repository, or Cancel.
- Kept cluster discovery, runner probing and verified runner updates off the Win32 UI thread; only the chooser/update confirmation is posted back to the owning UI thread, preserving the v0.9 thread-affinity fix.
- Prevented overlapping Add Project discovery sessions by disabling the action until its asynchronous result returns.
- Cluster projects still remain on the runner as `runner://<repo>` references; choosing a project never copies or mounts the repository on Windows.
- Preserved the host-aware 0.9.2 worker model, privacy-minimal runner inventory, allowlisted runner provider login and the 32-cycle Dashboard → Settings → Work responsiveness/visibility soak.

## 0.9.2 — 2026-08-16

- Made coding-worker status host-aware: Settings now distinguishes `This PC` workers from `Runner` workers instead of presenting local CLIs, the cluster control plane and the Chat bridge as one ambiguous inventory.
- Added privacy-minimal runner worker discovery with only worker ID/name/capability/status/cost and categorical install/auth/readiness state; command paths, account identifiers, raw authentication output and runner filesystem details are not exposed to the desktop or model-facing state.
- Runner worker inventory refreshes asynchronously so opening Settings never blocks the Win32 message thread. Explicit Rescan may use Workbench's existing verified cluster self-update when an older runner lacks the current inventory protocol; passive Settings display never mutates the runner.
- Added an operator-only, allowlisted runner provider-login path. Connect selected can open the provider's own human login flow on the execution host in an interactive SSH console, without accepting arbitrary remote command text and without exposing the operation through MCP.
- Separated the local Chat/MCP bridge from coding capacity. Dashboard provider counts no longer treat ChatGPT bridge readiness as an executable coding worker, while System status continues to report whether the local bridge listener is online.
- Clarified bridge language throughout Settings: a listening local MCP endpoint means the Workbench bridge is ready; it does not claim that a specific ChatGPT conversation is attached.
- Expanded the native bridge-status area so that connection warning is actually visible instead of being clipped in the production Settings layout.
- Preserved the v0.9 Win32 thread-affinity fix and 32-cycle Dashboard → Settings → Work responsiveness/visibility soak while adding asynchronous runner inventory.

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
- GitHub Actions dependency watches use progressive polling/backoff instead of status hammering; connected AI guidance now tells clients to do other independent useful work during Workbench-owned waits and never claim to be monitoring unless a durable watch exists.
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
