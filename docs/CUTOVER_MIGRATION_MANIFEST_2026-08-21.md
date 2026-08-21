# Workbench pre-cutover migration manifest — 2026-08-21

**Cutover assessment: READY, subject only to recording the final `main` SHA after this manifest/handoff reconciliation.**

This manifest records the final engineering state used to retire the legacy ChatGPT Project as an active development environment. The new ChatGPT Development Project is expected to use **Project-only memory** and bootstrap from this repository rather than historical conversation context.

## 1. Project identity

- **Project:** Workbench
- **Repository:** `DaisyCloverSoftware/workbench`
- **Continuation branch:** `main`
- **Current source version:** `0.9.55`
- **Related private operational transport:** a separate private Workbench relay repository; private deployment data is deliberately not copied into this public manifest

### Deployment/environment terminology

Workbench does **not** have a canonical website-style DEV instance.

Use the definitions in `docs/DECISIONS.md`:

- **development source** — normal branches / `main` source
- **PR/preview build** — CI artifact for a PR/commit; not DEV
- **release candidate/request** — coordinated release-request/version-bump state
- **stable release** — official GitHub version tag/release artifacts
- **cluster live** — installed runner/server/relay/MCP runtime
- **Windows live** — installed desktop application version

## 2. Current product state

Workbench is a ChatGPT-first developer/operations control plane and durable execution bridge.

Current engineering model:

- ChatGPT owns primary reasoning/coding, Git/GitHub, PR/CI/release orchestration and normal bounded machine-operation decisions.
- Workbench provides durable project/task state, safe repository eyes/hands, scheduling/execution infrastructure, bounded machine controls, private relay transport, an outbound typed Windows bridge and continuity across waits/chat boundaries.
- OpenClaw and other autonomous harnesses are optional operator/worker capacity, not the default coder or a mandatory trust hop.
- The human is an authority/decision boundary, not a clipboard/message bus.
- Routing is provider-neutral and capability/trust/cost based. Scarce/metered use is policy protected; external model-credit tests/probes require explicit user authorisation.

Current delivered foundations include multi-project durable tasks, scheduler-owned queueing, retries/dependency waits, persistent memory/context, safe repository hands, controlled publication, provider/harness adapters, Windows updater, cluster self-update, bounded private relay controls, six-lane Operations UI and typed outbound Windows application operations.

## 3. Current source/release/runtime baseline

### Source and stable release

- P0 Operations correction PR #232 merged as `cf97e8c1dab987782d91743df491e54f99a85103`.
- Its validated correction head was `3d07e0336c7e79eba552e17c443638e7adb188b8`.
- Release PR #233 merged as `5d08829a1924d6445d3578de9821bd3cae4dd823`.
- Source version is `0.9.55`.
- `CHANGELOG.md` records the 0.9.55 semantic correction and regression coverage.
- Stable tag/release is `v0.9.55`.

### Cluster live

The 0.9.55 Workbench maintenance update reached terminal `succeeded`, and the private capabilities manifest advertises `workbench_version: 0.9.55`.

Status: **deployed and runtime-version verified**.

### Windows live

The last directly observed installed desktop before the correction was 0.9.54. This cutover audit has not established a fresh installed Windows 0.9.55 semantic inspection.

Status:

- 0.9.55 Windows source/build/release: **implemented/tested/merged/released**
- installed Windows 0.9.55: **not yet verified in this cutover**
- corrected Operations semantics on the actual installed Windows UI: **awaiting inspection/user sign-off**

This is intentionally captured as the first new-Project Correction Sprint acceptance item rather than being hidden or misclassified as feature work.

## 4. Canonical documentation index

### Product definition / governance

- `README.md` — current product summary, build commands and governance bootstrap
- `VISION.md` — current product vision and responsibility principles
- `docs/GOVERNANCE.md` — source-of-truth hierarchy, change protocol, corrections-first sprint workflow and evidence language
- `docs/DECISIONS.md` — current decisions, superseded/rejected behaviour, negative requirements and unresolved decisions
- `docs/CURRENT_STATE.md` — verified implementation/release/deployment state and current correction backlog

### Architecture / data / execution

- `ARCHITECTURE.md` — intended current system boundaries, scheduler/control-plane split, JSON state model and deployment surfaces
- `docs/operations-dashboard-contract.md` — normative Operations/job/presence/history semantics
- `docs/HARNESS_PROTOCOL.md` — structured harness execution contract
- `docs/KNOWLEDGE_SYSTEM.md` — durable memory/context/reusable knowledge model
- `docs/TASK_WORKSPACES.md` — isolated task workspace model
- `docs/CHANGESET_PUBLISHING.md` — controlled changeset/review publication model
- `docs/COMMITTED_OPERATIONS_SCRIPTS.md` — reviewed committed operations-script boundary

### Security / privacy

- `SECURITY.md` — current trust boundaries and secret-handling rules
- `PUBLIC_SOURCE_POLICY.md` — public-source privacy requirements
- `docs/PRIVACY_GUARD.md` — additional privacy guardrail

### Testing / acceptance

- `CONTRIBUTING.md` — baseline test/format and contribution evidence rules
- `docs/UI_ACCEPTANCE_V0.9.md` — production Windows UI and semantic acceptance contract
- `.github/workflows/build.yml`
- `.github/workflows/runner.yml`
- `.github/workflows/ui-responsiveness.yml`

### Release / deployment / updates

- `.github/workflows/prepare-release-request.yml`
- `.github/workflows/release.yml`
- `docs/PRIVATE_SELF_UPDATE.md`
- `docs/PERSONAL_PRO_RELAY.md`
- `docs/CHATGPT_SHARED_INTEGRATION.md`

### Audit/history only — not current requirement authority

- `docs/GOVERNANCE_RESET_2026-08-21.md`
- `docs/REPOSITORY_CLEANUP_MANIFEST.md`
- `docs/CONVERSATION_PRUNING_MANIFEST.md`
- `docs/LOCAL_CHECKOUT_AUDIT_2026-08-21.md`
- `docs/POST_RESET_HANDOFF.md` — superseded by this manifest and `docs/NEW_PROJECT_HANDOFF.md`
- archival tag `archive/pre-governance-reset-20260821` — preservation only, never a development baseline

## 5. Documentation/source audit classification

| Material | Classification at cutover | Resolution |
| --- | --- | --- |
| `README.md` | CURRENT | Product/build/governance bootstrap consistent with current architecture. |
| `VISION.md` | CURRENT | ChatGPT-first / user-not-message-bus principles retained. |
| `docs/GOVERNANCE.md` | CURRENT | Normative authority and sprint/change-control rules. |
| `docs/DECISIONS.md` | CURRENT | Current/superseded/rejected/unresolved decisions durable. |
| `ARCHITECTURE.md` | CURRENT | Reflects scheduler, relay, typed Windows boundary and JSON state model. |
| `SECURITY.md` | CURRENT | Reflects current trust/secret/private-relay/Windows boundaries. |
| `CHANGELOG.md` | CURRENT | 0.9.55 release semantics recorded. |
| `docs/CURRENT_STATE.md` | PARTIALLY CURRENT before cutover | Reconciled to 0.9.55 source/release/runtime and actual acceptance gap. |
| `docs/operations-dashboard-contract.md` | CURRENT contract / stale status before cutover | Reconciled to 0.9.55 source correction + outstanding installed-Windows acceptance. |
| `docs/UI_ACCEPTANCE_V0.9.md` | CURRENT contract / stale status before cutover | Reconciled to 0.9.55 source correction + outstanding installed-target sign-off. |
| `ROADMAP.md` | PARTIALLY CURRENT before cutover | Reconciled so completed P0 code is not re-listed as work; correction/roadmap/ideas separated. |
| `docs/POST_RESET_HANDOFF.md` | SUPERSEDED | Replaced by `docs/NEW_PROJECT_HANDOFF.md`; retained only as dated historical reset artifact. |
| Reset/cleanup/pruning/local-audit docs | HISTORICAL/AUDIT | Retained for traceability, explicitly below current canonical state. |
| Old Workbench release patch uploads | DUPLICATED/HISTORICAL | Current Git/CHANGELOG/source supersede them; no migration dependency. |
| Old Workbench installer uploads | HISTORICAL | Current release/updater source and docs govern installation/update behaviour. |
| Original governance-reset conversation handoff | HISTORICAL EVIDENCE | Its enduring decisions are represented in canonical repository docs. |
| Separate monolithic SRS | MISSING BY DESIGN, NOT A GAP | Current requirements are intentionally distributed across vision, decisions, architecture, security and domain acceptance contracts; no competing SRS is required to continue engineering. |

No unresolved contradiction was found that requires historical chat authority to resolve ordinary future development.

## 6. Conversation, uploaded-file and durable-context migration

The governance reset and this cutover reviewed the accessible Workbench freeze handoff, relevant uploaded/File Library material, repository/PR/release evidence and accessible durable Workbench context/memory.

Surviving material decisions were promoted into canonical repository documentation, including:

- repository-over-conversation authority
- ChatGPT-first responsibility split
- no human clipboard/message-bus workflow
- corrections-first inspectable sprint/sign-off workflow
- non-human waits are not stopping points
- configured target/runner fidelity
- no unapproved external model-credit experiments
- semantic acceptance separate from responsiveness/build health
- explicit deployment terminology
- negative/do-not-reintroduce requirements

Old release patches/installers and the original freeze handoff are not required bootstrap dependencies; current source/CHANGELOG/canonical docs supersede them.

### Historical-conversation coverage limitation

The available ChatGPT interfaces cannot prove exhaustive enumeration/full-text reading of every historical Workbench conversation. That residual archival limitation was explicitly accepted during governance-reset closure.

It is **not a cutover blocker** because governance now makes the repository authoritative and historical conversations non-authoritative evidence. A future chat must not reread old conversations to make an ordinary decision; any deliberately resurfaced historical claim becomes binding only after conscious reconciliation into the current repository record.

## 7. Data/schema/configuration audit

Workbench does not use a relational database schema at this baseline. Core durable state is private local JSON (`State.Version` 3), evolved through decode/normalisation/runtime repair and compatibility tests.

Secret values are intentionally not repository configuration. Security/deployment docs define the boundary without publishing values:

- MCP/relay/runtime credentials remain local/protected
- vault plaintext is not exposed through model-facing tools
- private relay payloads must not contain raw credentials
- public docs/source must not contain deployment-specific private topology or secrets

GitHub release publishing uses repository-scoped GitHub Actions permissions and the built-in `${{ github.token }}` rather than requiring a separately documented project secret value.

## 8. Build/test/release/operation recoverability

Baseline local commands are documented in `README.md` / `CONTRIBUTING.md`, including:

```text
gofmt -w .
go test ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w -H=windowsgui" -o Workbench.exe ./cmd/workbench
go build -o workbench-runner ./cmd/workbench-runner
go build -o workbench-server ./cmd/workbench-server
go build -o workbench-relay ./cmd/workbench-relay
```

CI/release workflows exist for:

- full build/test and production Windows build
- runner validation
- Windows UI responsiveness/screenshot evidence
- release-request preparation
- release asset generation/publication

Release assets are designed to include the Windows app, Windows updater and Linux cluster package with checksums.

The private self-update path is intentionally narrow and documented; it does not accept arbitrary commands/targets.

## 9. Repository cleanliness / recoverability

At cutover audit:

- canonical continuation branch is `main`
- no open Workbench PRs
- no open Workbench issues
- registered `runner://workbench` checkout returned clean `git status --short`
- no important current Workbench source is known to exist only in an uncommitted working tree
- previous local-only/branch/worktree material was preservation-first audited during the governance reset
- old public branch history is recoverable via the historical archive tag without making the archive authoritative

Current remote branch surface is understood as:

- `main`
- `fix/operations-terminal-session-separation`
- `governance/close-reset-20260821`
- `release-request/v0.9.55`

The three non-main branches are known correction/audit/release residue. They are not alternative continuation branches and were not deleted merely for tidiness.

Closed issues #234–#239 were accidentally created during cutover connector/tool verification and immediately marked `not_planned` as non-project audit noise. They are explicitly **not backlog or product requirements**.

## 10. Known corrections required before new feature development

These form the mandatory initial Correction Sprint backlog in the new Project.

1. **P0 — Windows Operations 0.9.55 acceptance closure**
   - establish actual Windows live 0.9.55
   - inspect real Operations UI
   - prove terminal history/session presence does not inflate live job counts
   - confirm genuine pending/running/waiting remote work remains visible
   - execute observation-driven corrections if necessary
   - obtain user sign-off

2. **P1 — Release publication reliability**
   - remove dependence on identical-tree/no-op `main` retriggers

3. **P1 — Full unattended continuation live acceptance**
   - clean proof of `waiting_dependency → automatic resume → useful work → completed`

4. **P1 — Private relay retention governance**
   - define safe retention/compaction/cleanup protecting pending work, audit needs and privacy

5. **P2 — Blender end-to-end live acceptance**
   - fresh typed headless GPU render proof with privacy-safe backend/device evidence

6. **P2 — Unreal startup acceptance/investigation**
   - fresh five-minute result and inherited `zen` investigation

The 0.9.54 false-running source bug itself is **not** an outstanding coding item; its source correction is in 0.9.55. The remaining P0 is installed-target semantic acceptance/sign-off.

## 11. Approved future roadmap

After the initial Correction Sprint is inspected and signed off:

- **Authoritative cross-plane job model** — unify truthful inventory across scheduler-native tasks, CI, direct server operations, typed Windows jobs and AI-worker jobs without inferring execution from transport recency.

This is approved architecture direction, not permission to start before correction sign-off.

## 12. Ideas / unapproved possibilities

These require fresh specification/decision before implementation:

- searchable decisions / project knowledge graph shape (old PR #110 implementation is stale/rejected)
- exact duration of the bounded session-presence lease
- richer drag/reorder or selected-job controls beyond current explicit contracts
- whether relay-state cross-process locking is necessary under the supported topology
- other candidate future integrations listed as candidates in `ROADMAP.md`

## 13. Explicitly superseded / rejected decisions

Important older material that MUST NOT be resurrected as current intent:

- conversation history as project authority
- activity-first Dashboard as the primary Operations model
- all-zero Operations while real remote work exists
- session-active terminal history treated as running work (`Running 100` failure)
- fake elapsed-time progress percentages
- generic inbound/generic-shell Windows authority
- implicit direct ChatGPT → OpenClaw development delegation
- OpenClaw as primary/default coder
- silent local execution when configured target is unavailable
- unapproved external/scarce model-credit testing
- skeleton/prototype/partial backend or UI presented as finished product
- 90-second Unreal smoke
- old Unreal `TNotNull` crash described as the current failure
- Blender GUI preferences assumed sufficient for factory-startup headless GPU use
- no-op release retrigger accepted as the desired release procedure
- old PR #110 implementation treated as current knowledge-graph authority

See `docs/DECISIONS.md` for the normative register and rationale.

## 14. Critical invariants

Future development MUST preserve:

- repository canonical state outranks chat/history/operational checkout
- material corrections/decisions propagate to docs and acceptance tests
- session presence, individual job execution and terminal operation history remain distinct
- progress/queue/worker/capacity data remains truthful and authority-bound
- configured execution targets are honoured or reported unavailable; fallback is explicit/truthful only
- Windows remote capabilities remain outbound typed/allowlisted unless a new explicit security decision changes the boundary
- direct ChatGPT work does not silently route into OpenClaw
- external model-credit probes require explicit authorisation
- release, artifact, deployment/installation and semantic verification remain separate evidence gates
- corrections/sprints produce inspectable product state and user sign-off before moving on
- CI/build/runner/release/deployment waits alone are not stopping conditions
- old conversations do not have to be reread for ordinary project decisions

## 15. Current verification status

- governance reset: **complete**
- canonical docs: **reconciled for cutover**
- source version: **0.9.55**
- P0 source correction: **implemented/tested/merged/released**
- cluster 0.9.55: **deployed/runtime-version verified**
- Windows 0.9.55 installed semantic acceptance: **not yet verified; initial Correction Sprint item**
- registered checkout uncommitted state: **clean**
- open PRs: **zero**
- open issues: **zero**
- continuation branch: **`main`**
- exact continuation SHA: **record in the final cutover report after the last cutover-document commit**

## 16. New Project bootstrap

Use `docs/NEW_PROJECT_HANDOFF.md` plus the exact continuation SHA from the final legacy-Project cutover report.

The new Project must not require this legacy Project's conversation history to understand, test, release, deploy or continue Workbench.
