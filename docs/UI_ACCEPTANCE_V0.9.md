# Workbench v0.9 production UI acceptance

The production Windows application must be one coherent product, not a development/control-plane utility.

This document is a UI/behavioural acceptance contract. A rendered screenshot or responsive HWND is supporting evidence; it is never sufficient proof that the data being displayed has the correct meaning.

## Visual acceptance target

- Dark professional desktop dashboard with a permanent left navigation rail and top action bar.
- Default Dashboard/Operations view with real Workbench operational state, projects, worker/capacity status and system status.
- Card hierarchy, spacing, typography and accent treatment comparable to the approved production direction.
- No invented progress percentages, fake health numbers, fake workers or synthetic operational state in the real application.
- Every visible action must work; no decorative dead buttons.
- Work view retains the real multi-project autonomous task workflow, human-attention answer path, review/open/retry actions, project notes and cancellation.
- Settings retains provider/runner/structured-harness routing, MCP, review policy, encrypted vault and verified updater controls.
- Waiting/retry work remains visible and cancellable.
- Genuine human-attention state remains visually prominent.
- Review-ready work presents the real structured PR/branch/commit state.

## Operations semantic acceptance

The Dashboard MUST implement `docs/operations-dashboard-contract.md`.

Semantic verification must distinguish:

- actual executing/queued/waiting jobs;
- project/ChatGPT session presence;
- recent terminal operation history.

Required regression cases include:

- actual running job contributes to Running;
- scheduler-native queued job contributes to Queued and only shows a queue position Workbench owns;
- dependency/retry wait contributes to Waiting;
- attention contributes to Needs You;
- completed operation + active session/presence lease does **not** become Running;
- failed operation + active session/presence lease does **not** become Running;
- completed/failed terminal history remains outside live job counts;
- real remote pending/running work remains visible with a large historical relay;
- measured/stage progress uses real source data and indeterminate work has no fabricated percentage.

### Historical regression and current status

Workbench 0.9.54's observed `Running 100` projection failed these requirements even though UI responsiveness passed. That remains permanent regression evidence.

Workbench 0.9.55 source corrects the terminal-history/session-presence projection and includes automated semantic regressions. At the 2026-08-21 cutover, the source correction is implemented/tested/merged/released and the cluster runtime is updated, but the actual installed Windows 0.9.55 Dashboard has not yet received user-visible semantic inspection/sign-off.

The initial Correction Sprint in the new Project must close that installed-target acceptance gap before new feature work.

## Responsiveness acceptance

- Dashboard, Work and Settings windows remain responsive under the Windows watchdog cycles used by CI.
- Page/window visibility and message pumping remain correct.
- Long-running operations must not make ordinary UI/status/control surfaces appear frozen merely because execution shares transport infrastructure.

Responsiveness passing does not waive semantic acceptance.

## Release acceptance

For a release that changes the production Windows UI/Operations projection:

- Windows and Linux test suites are green.
- Production `Workbench.exe` and `Workbench-Updater.exe` build successfully.
- Windows artifact contains only the intended production app/updater payload.
- Automated real Windows screenshot/window capture is attempted where supported; if hosted desktop capture is unsupported, report that explicitly rather than substituting concept art.
- State-projection semantic tests relevant to the change are green.
- The actual expected tag/release and downloadable artifact are verified separately from the merge.
- Where the change depends on live cluster/Windows data, target-environment verification is recorded separately from CI.
- Do not call the change complete merely because build, screenshot or responsiveness gates passed.
