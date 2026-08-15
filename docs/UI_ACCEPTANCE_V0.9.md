# Workbench v0.9 production UI acceptance

The production Windows application must be one coherent product, not a development/control-plane utility.

## Visual acceptance target

- Dark professional desktop dashboard with a permanent left navigation rail and top action bar.
- Default Dashboard view with real Workbench metrics, recent activity, active work, projects, provider/worker health, and system status.
- Card hierarchy, spacing, typography, and accent treatment comparable to the approved dashboard concept.
- No invented progress percentages, fake health numbers, fake workers, or synthetic operational state in the real application.
- Every visible action must work; no decorative dead buttons.
- Work view retains the real multi-project autonomous task workflow, human-attention answer path, review/open/retry actions, project notes, and cancellation.
- Settings retains provider/runner/structured-harness routing, MCP, review policy, encrypted vault, and verified updater controls.
- Waiting-retry work remains visible and cancellable.
- Genuine human-attention state remains visually prominent.
- Review-ready work presents the real structured PR/branch/commit state.

## Release acceptance

- Windows and Linux test suites green.
- Production `Workbench.exe` and `Workbench-Updater.exe` build successfully.
- Windows artifact contains only the production app and updater.
- Attempt an automated actual Windows screenshot capture in CI; if hosted desktop capture is unsupported, report that explicitly rather than substituting concept art.
- `v0.9.0` is not released until the production shell is visually coherent and all existing safety/control-plane tests still pass.
