# Workbench repository cleanup manifest — 2026-08-21

Status: **PUBLIC SOURCE / REGISTERED CHECKOUT CLEANUP COMPLETE**

Private relay historical transport is intentionally retained pending a separate retention policy; that does not make the public-source repository cleanup incomplete.

## Baseline and final shape

The reset began with **288 remote branches**, later rising as temporary governance branches were created. Cleanup was preservation-first rather than name/age based.

Final verified cleanup checkpoint after PR #229:

- canonical public branch: `main`;
- active public branch count: **1**;
- open pull requests: **0**;
- registered Workbench source checkout: clean `main`;
- local Workbench worktree count: **1**;
- historical removed public branch graphs: preserved under tag `archive/pre-governance-reset-20260821`;
- separate old local-only SEC-008 history: preserved behind a local audit ref, not republished/canonicalised.

## Remote branch cleanup evidence

### Tier 1 — complete-history reachability

A tested operation refreshed all remote refs and deleted only branches whose tip was already an ancestor of canonical `main`.

Result at execution time:

- **157** fully merged branch refs deleted;
- **135** unique-history refs retained.

No unique-history branch was deleted by this tier.

### Tier 2 — patch-equivalence / PR provenance audit

The remaining refs were audited read-only with `git cherry` plus GitHub `refs/pull/*/head` provenance.

At that audit point:

- 12 refs had zero novel patches and exact PR-head provenance;
- 124 refs contained genuinely novel patch history;
- zero patch-equivalent refs lacked PR-head provenance.

This proved that commit-ID uniqueness did not always mean unique code effect, but also confirmed that many old refs still carried historical patch graphs that should not simply disappear.

### Tier 3 — historical archive consolidation

Instead of leaving historical branches active-looking forever, PR #229 added a tested archive operation.

Before source-ref deletion it:

1. created synthetic archive-anchor commits whose **tree is exactly canonical `main`'s tree** at the checkpoint;
2. attached removed public branch tips as parents in bounded tranches;
3. pushed lightweight tag `archive/pre-governance-reset-20260821`;
4. proved every source branch tip was reachable from that archive;
5. proved the archive checkpoint tree matched canonical `main`;
6. only then deleted the source branch refs.

Live result:

- archived source refs: **137**;
- archive head: `bcb7a1a6056b6f2d4a132bf51dbbf224b57f8832`;
- archive tree: `73b84fd417a567ab2a51baf06cc7f6019dde0ac7`;
- checkpoint `main`: `ce4ecd5f0e47d764d6ad4221619390db1ea70af4`;
- checkpoint `main` tree: `73b84fd417a567ab2a51baf06cc7f6019dde0ac7`;
- archive and main checkpoint trees: **identical**;
- active public branch count after operation: **1 (`main`)**;
- archive operation exit code: 0.

One Git warning reported a duplicate parent ignored while constructing an archive anchor. The operation's later reachability/tree verification still passed and exit code was 0.

The archive tag is historical preservation only. It is not a development branch, migration baseline or requirements source.

## Pull-request cleanup

Old PR #110 (`Add searchable decisions and a project knowledge graph`) was based on Workbench 0.9.10. During the reset it was explicitly closed **unmerged** with a governance comment explaining:

- the old implementation basis is stale;
- the capability idea itself is not rejected;
- any future design must be specified afresh against current architecture/governance.

No open PR remained at the post-#229 cleanup checkpoint.

## Registered operational checkout cleanup

The reset discovered that the registered source checkout was not a trustworthy mirror of canonical source.

### Uncommitted relay-lock experiment

Found:

- one modified relay-state source file;
- three untracked relay-lock/concurrency files.

The experiment was inspected and its rationale/risks preserved in `docs/LOCAL_CHECKOUT_AUDIT_2026-08-21.md`. It was explicitly classified unaccepted/untested, then the exact working-tree changes were removed/restored through a guarded governance cleanup.

### Stale local lineage

The checkout was on an old v0.5-era `main` plus one local-only SEC-008 workflow-pin commit. Current public source already contained the security effect.

The local-only commit was preserved behind a local audit ref, then the registered `main` checkout was realigned to canonical GitHub `main` rather than merged into current product history.

### Stale worktrees

Eight local Workbench worktrees were enumerated:

- one primary checkout;
- six old clean detached worktrees;
- one old named file-mode worktree.

The named worktree's two dirty files were inspected before deletion. Their **full Git blob hashes exactly matched already-published blobs** from public commit `0b601caab1859e42767c5019ba61c01cf3af8c55`.

Only then were those duplicate edits restored and the seven stale secondary worktrees removed normally. No force-removal of unknown dirty work was used.

Final local verification:

- worktree count: 1;
- branch: `main`;
- status: clean;
- local audit ref still preserves the old local-only commit.

## Active-tree cleanup / authority

Canonical governance documents have distinct roles and superseded docs were updated rather than duplicated.

Known product-semantic defects remain in current source/tests where feature-fix work was intentionally frozen. In particular, the 0.9.54 test/implementation that can project terminal remote operations as Running is recorded as a P0 post-reset defect rather than silently changed during cleanup.

The old 0.9.54 release-publication no-op commit remains in history as audit evidence. Public history was not rewritten to hide it.

## Private relay retention

The private relay's append-oriented historical transport was **not mass-purged**.

Current decision:

- bounded live projection prevents old history from defining current Operations state;
- underlying relay history is retained intentionally;
- exact retention/compaction policy remains future design work;
- destructive transport cleanup requires its own pending-request/audit/rollback/privacy safety contract.

## Completion result

For the Workbench public source repository and registered operational source checkout, cleanup is complete to the governance reset's preservation rules:

- remote branch history audited/preserved;
- stale active branch surface reduced to `main` only at cleanup checkpoint;
- old source graphs retained under a non-authoritative archive tag;
- open stale PR resolved;
- local-only history preserved before realignment;
- local working changes classified before removal;
- all local worktrees audited and reduced to one clean checkout;
- final status evidence recorded.

No further repository deletion is required to close the repository-cleanup gate. The remaining overall governance-reset blocker is historical conversation/pruning coverage, not Git/repository hygiene.
