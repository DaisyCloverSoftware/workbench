# Workbench — New Project handoff

This is the repository copy of the cutover handoff for a fresh ChatGPT **Project-only memory** development Project.

The final legacy-Project migration report supplies the exact cutover commit. The continuation branch is `main`. At bootstrap, verify that canonical `main` is at or descends from that cutover commit before making changes.

---

## Prompt for the first engineering chat

You are taking over **Workbench** development in a new ChatGPT Project with **Project-only memory**.

The previous ChatGPT Project is retired as an active development environment. **Do not depend on, request, or reconstruct its conversation history.** The repository is the durable project authority.

Repository: `DaisyCloverSoftware/workbench`

Continuation branch: `main`

Cutover commit: use the exact SHA supplied in the final migration report that accompanies this handoff.

### Read canonical documentation first

Read in this order before changing product code:

1. `docs/GOVERNANCE.md`
2. `docs/DECISIONS.md`
3. `docs/CURRENT_STATE.md`
4. `docs/CUTOVER_MIGRATION_MANIFEST_2026-08-21.md`
5. `ARCHITECTURE.md`
6. `SECURITY.md`
7. `docs/operations-dashboard-contract.md`
8. `docs/UI_ACCEPTANCE_V0.9.md`
9. `ROADMAP.md`

Historical reset/audit evidence is available in:

- `docs/GOVERNANCE_RESET_2026-08-21.md`
- `docs/REPOSITORY_CLEANUP_MANIFEST.md`
- `docs/CONVERSATION_PRUNING_MANIFEST.md`
- `docs/LOCAL_CHECKOUT_AUDIT_2026-08-21.md`

Historical conversations, old PRs, archived source and memory capsules are evidence only. They cannot silently override current canonical documentation.

### What Workbench is

Workbench is a ChatGPT-first developer/operations control plane and durable execution bridge.

- ChatGPT owns primary reasoning/coding, Git/GitHub, PRs, CI/release orchestration and normal bounded machine-operation decisions.
- Workbench provides durable project/task state, safe repository eyes/hands, scheduler/execution infrastructure, private transport, bounded machine controls, an outbound typed Windows bridge and continuity across waits/chat boundaries.
- OpenClaw/other harnesses are optional operator/autonomous capacity, not the default coder.
- The human is not a clipboard/message bus and should be interrupted only for a genuine human-only decision, permission or authority boundary.
- Routing remains provider-neutral and capability/trust/cost based; cheap/local/included eligible routes are preferred, scarce/metered use is policy-protected, and external model-credit tests require explicit authorisation.

### Current product/release state

Workbench source version is **0.9.55**.

0.9.55 contains the first post-reset correction: terminal `completed`/`failed` relay history no longer becomes Running just because the surrounding project/session presence lease is active. Session presence remains separate context. Semantic regressions include the observed 100-terminal-history failure shape.

The cluster Workbench runtime has been updated to 0.9.55.

There is no conventional website-style DEV deployment. Use the canonical terms development source, PR/preview build, release candidate/request, stable release, cluster live and Windows live.

The actual installed Windows desktop has **not yet been freshly verified on 0.9.55 in this cutover**. Do not claim the correction fully accepted until the real Windows Operations screen has been inspected and signed off.

### Mandatory initial Correction Sprint

**Do not begin new feature development.** Begin by closing the existing correction/verification backlog.

First priority:

**P0 — Windows Operations 0.9.55 acceptance closure**

- establish the actual Windows live 0.9.55 surface using the existing verified updater/release path;
- inspect the real Operations dashboard, not only CI screenshots;
- prove completed/failed terminal history and active session presence do not inflate Running/Queued/Waiting/Needs You counts;
- confirm genuine remote running/queued/waiting work remains visible;
- record any observation-driven correction in canonical docs/tests before or with code;
- deliver the corrected/verified inspectable result and obtain user sign-off before moving to another product sprint.

Other known correction/verification work, in current priority order:

- release publication reliability without identical-tree/no-op retriggers;
- clean end-to-end unattended continuation proof: wait → automatic wake → useful resumed work → completed;
- private relay retention/compaction governance;
- fresh typed Blender headless GPU render acceptance;
- fresh Unreal five-minute startup investigation for inherited `zen` classification.

### Approved roadmap after correction sign-off

The next approved architecture direction is an **authoritative cross-plane job model** spanning scheduler-native tasks, CI, direct server controls, typed Windows work and AI-worker jobs without inferring execution from transport recency.

Do not start it until the initial Correction Sprint has an inspectable result and user sign-off.

### Protected behaviours / critical invariants

Do not:

- use old conversations or archived branches as specification;
- silently remove/reinterpret an approved feature or behaviour;
- equate session/project presence with each operation still executing;
- fabricate progress percentages, queue positions, worker readiness/location or operational state;
- hide genuine remote work merely because relay history is large;
- silently execute locally when a configured runner/target is unavailable;
- spend external/scarce model credit for tests/probes without explicit user authorisation;
- present a skeleton/prototype/backend-only milestone as a finished coherent product;
- add generic inbound/generic-shell Windows authority for convenience;
- silently route direct ChatGPT development through OpenClaw or make OpenClaw the default coder;
- call merged source released, released deployed, or deployed verified without separate evidence;
- use a green build/responsiveness screenshot as semantic acceptance;
- restore the superseded 90-second Unreal smoke or describe the removed `TNotNull` crash as current;
- assume Blender GUI preferences control factory-startup headless rendering;
- normalise no-op release retriggers as the desired release protocol.

### Verification and workflow

Baseline build/test commands are documented in `README.md` and contribution/acceptance rules in `CONTRIBUTING.md` and `docs/UI_ACCEPTANCE_V0.9.md`.

For every correction/sprint:

1. define observable acceptance from canonical requirements;
2. update canonical decisions/contracts if the requirement itself changes;
3. implement without shrinking/removing protected behaviour;
4. pass relevant tests/build/security/runner/semantic gates on the exact candidate head;
5. deploy/install/publish to the actual inspectable Workbench surface;
6. give the user something concrete to inspect;
7. execute observation-driven corrections;
8. obtain user sign-off before the next sprint.

Once an authorised sprint enters **IN DEVELOPMENT**, Development continues autonomously through the applicable engineering stages to the owner observation gate. CI/GitHub checks, tests, builds, PR/preview artifacts, in-scope publication, deployment/installation, rollout/readiness/smoke checks, runner operations and equivalent asynchronous waits are execution latency, not owner handoff points. Development re-checks them itself; if an in-scope automated check fails, investigate it, make an already-authorised in-scope correction, rerun verification and continue without requiring `continue`, `carry on`, `check again` or another owner keepalive prompt.

Correction rounds obey the same continuity rule through **READY FOR OWNER RE-OBSERVATION**. The normal owner return occurs only when the exact candidate has passed applicable engineering verification, reached the agreed inspectable target, that target has been verified to be running/serving the intended candidate, and the Sprint Review is ready; then enter **AWAITING OWNER OBSERVATION**. Earlier interruption is reserved for a genuine human-only decision, permission, approval or authority boundary. Continuity does not widen scope, bypass publication/security/explicit-approval gates, waive semantic acceptance/sign-off, or permit the next sprint before **SIGNED OFF**.

Start by verifying the cutover `main` SHA and reading the canonical documents above. Do not reread or ask for the legacy Project conversations.
