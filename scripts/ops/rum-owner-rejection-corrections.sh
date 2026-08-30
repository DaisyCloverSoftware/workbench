#!/usr/bin/env bash
set -euo pipefail
umask 077

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <exact-rum-candidate-sha>" >&2
  exit 64
fi
CANDIDATE_SHA="$1"
[[ "$CANDIDATE_SHA" =~ ^[0-9a-f]{40}$ ]] || { echo "full lowercase candidate SHA required" >&2; exit 64; }

REPOSITORY="DaisyCloverSoftware/rum"
BRANCH="sprint-0-rum-owner-rating-flow-20260823"
PR=153
CANONICAL_FOUNDER_BASE="f9d45bebfcc3ce4bbb2d75977dd5614f7403dd5e"

for command in gh git python3 base64 mktemp; do
  command -v "$command" >/dev/null 2>&1 || { echo "missing $command" >&2; exit 2; }
done
TOKEN="${GH_TOKEN:-}"
[[ -n "$TOKEN" ]] || TOKEN="$(gh auth token 2>/dev/null || true)"
[[ -n "$TOKEN" ]] || { echo "GitHub token unavailable" >&2; exit 2; }

head="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/${BRANCH}" --jq '.object.sha')"
[[ "$head" == "$CANDIDATE_SHA" ]] || { echo "PATCH BLOCKED: branch moved" >&2; exit 78; }
pr_state="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/pulls/${PR}" --jq '[.state, (.draft|tostring), .head.sha, (.merged_at // "")] | @tsv')"
IFS=$'\t' read -r state draft pr_head merged_at <<<"$pr_state"
[[ "$state" == open && "$draft" == true && "$pr_head" == "$CANDIDATE_SHA" && -z "$merged_at" ]] || {
  echo "PATCH BLOCKED: PR #${PR} is not open/draft/unmerged at exact head" >&2
  exit 78
}

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
GH_TOKEN="$TOKEN" gh repo clone "$REPOSITORY" "$tmp/rum" -- --no-checkout --filter=blob:none >/dev/null
git -C "$tmp/rum" checkout --detach "$CANDIDATE_SHA" >/dev/null
[[ "$(git -C "$tmp/rum" rev-parse HEAD)" == "$CANDIDATE_SHA" ]] || { echo "PATCH BLOCKED: checkout mismatch" >&2; exit 78; }

# Recover the exact canonical Founder Alpha artwork from the pre-profile owner-approved baseline.
git -C "$tmp/rum" show "${CANONICAL_FOUNDER_BASE}:apps/web/src/components/FounderAlphaBadge.tsx" > "$tmp/rum/apps/web/src/components/FounderAlphaBadge.tsx"
grep -Fq 'Approved small RUM flag' "$tmp/rum/apps/web/src/components/FounderAlphaBadge.tsx" || { echo "PATCH BLOCKED: canonical Founder artwork marker missing" >&2; exit 78; }
grep -Fq 'founder-alpha-approved' "$tmp/rum/apps/web/src/components/FounderAlphaBadge.tsx" || { echo "PATCH BLOCKED: canonical Founder class missing" >&2; exit 78; }

python3 - "$tmp/rum" <<'PY'
from pathlib import Path
import sys

root = Path(sys.argv[1])

def replace_once(rel, old, new, label):
    path = root / rel
    text = path.read_text()
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label} marker mismatch: {count}")
    path.write_text(text.replace(old, new, 1))

# Self-added identities: normalise every persisted ownership surface to self-declared,
# while leaving independently existing identities on the verification workflow.
replace_once(
    'apps/api/app/Http/Controllers/Api/V1/EntityClaimController.php',
    "use App\\Models\\Entity;\nuse App\\Services\\EntityClaimService;",
    "use App\\Models\\AccountEntityLink;\nuse App\\Models\\Entity;\nuse App\\Models\\EntityRelationship;\nuse App\\Services\\EntityClaimService;",
    'entity claim imports',
)
old = '''            // Any claimable identity this account itself introduced may be linked
            // as mine by self-declaration. This establishes ownership/control
            // without pretending it is externally verified. Existing identities
            // created by somebody else stay pending and require verification or
            // claim review.
            if ($claim->status === 'pending'
                && $selfDeclarableIdentity
                && (string) $entity->created_by_user_id === (string) $actor->id) {
                $claim = $service->approve($claim, null, 1);
                $claim->forceFill(['verification_state' => 'self_declared'])->save();
                $entity->forceFill(['verification_state' => 'self_claimed'])->save();
                $selfDeclared = true;
            }
'''
new = '''            // When this account introduces an identity and immediately links it as
            // mine, that self-declaration is the ownership state. Persist the same
            // state on the claim, account link and identity relationship so a reload,
            // navigation or later login cannot fall back to a bogus verification
            // workflow. An independently existing identity still remains pending.
            if ($selfDeclarableIdentity
                && (string) $entity->created_by_user_id === (string) $actor->id
                && in_array($claim->status, ['pending', 'approved'], true)) {
                if ($claim->status === 'pending') {
                    $claim = $service->approve($claim, null, 1);
                }

                $claim->forceFill([
                    'status' => 'approved',
                    'verification_level' => 1,
                    'verification_state' => 'self_declared',
                ])->save();
                $entity->forceFill(['verification_state' => 'self_claimed'])->save();

                AccountEntityLink::query()
                    ->where('user_id', $actor->id)
                    ->where('entity_id', $entity->id)
                    ->where('status', 'active')
                    ->whereNull('valid_to')
                    ->update(['verification_state' => 'self_declared']);

                EntityRelationship::query()
                    ->where('source_entity_id', $entity->id)
                    ->where('created_by_user_id', $actor->id)
                    ->whereNull('valid_to')
                    ->whereHas('type', static fn ($query) => $query->where('key', 'identity_of'))
                    ->update(['verification_state' => 'self_declared']);

                $selfDeclared = true;
            }
'''
replace_once('apps/api/app/Http/Controllers/Api/V1/EntityClaimController.php', old, new, 'self declaration persistence')

# Lock the deeper persisted-state semantics into feature coverage.
replace_once(
    'apps/api/tests/Feature/Entity/RateUrMateMyIdentityProfileTest.php',
    '''        $this->assertDatabaseMissing('entity_claim_verifications', [
            'entity_claim_id' => $claimId,
        ]);
    }

    public function test_self_added_person_is_mine_by_self_declaration_without_verification_pending(): void
''',
    '''        $this->assertDatabaseMissing('entity_claim_verifications', [
            'entity_claim_id' => $claimId,
        ]);
        $this->assertDatabaseHas('account_entity_links', [
            'user_id' => $member->id,
            'entity_id' => $entityId,
            'link_type' => 'controls_identity',
            'status' => 'active',
            'verification_state' => 'self_declared',
        ]);
        $this->assertDatabaseHas('entity_relationships', [
            'source_entity_id' => $entityId,
            'created_by_user_id' => $member->id,
            'verification_state' => 'self_declared',
        ]);
        $this->assertDatabaseHas('entities', [
            'id' => $entityId,
            'verification_state' => 'self_claimed',
        ]);

        $this->postJson('/api/v1/entities/'.$entityId.'/claims', [
            'claimType' => 'controls_identity',
        ])->assertOk()
            ->assertJsonPath('data.status', 'approved')
            ->assertJsonPath('data.verificationState', 'self_declared')
            ->assertJsonPath('meta.selfDeclared', true)
            ->assertJsonPath('meta.verificationPromptRequired', false);
    }

    public function test_self_added_person_is_mine_by_self_declaration_without_verification_pending(): void
''',
    'digital identity persisted self declaration assertions',
)
replace_once(
    'apps/api/tests/Feature/Entity/RateUrMateMyIdentityProfileTest.php',
    '''        $this->assertDatabaseMissing('entity_claim_verifications', [
            'entity_claim_id' => $claimId,
        ]);
    }

    public function test_existing_identity_added_by_somebody_else_remains_a_verification_claim(): void
''',
    '''        $this->assertDatabaseMissing('entity_claim_verifications', [
            'entity_claim_id' => $claimId,
        ]);
        $this->assertDatabaseHas('account_entity_links', [
            'user_id' => $member->id,
            'entity_id' => $entityId,
            'link_type' => 'represents_person',
            'status' => 'active',
            'verification_state' => 'self_declared',
        ]);
        $this->assertDatabaseHas('entities', [
            'id' => $entityId,
            'verification_state' => 'self_claimed',
        ]);
    }

    public function test_existing_identity_added_by_somebody_else_remains_a_verification_claim(): void
''',
    'person persisted self declaration assertions',
)

# Profile payload exposes the stored avatar media URL exactly like MemberResource.
replace_once(
    'apps/api/app/Services/ProfileService.php',
    "                    'avatarTone' => (int) ($profile->avatar_tone ?? 1),\n                    'relationship' => $relationship,",
    "                    'avatarTone' => (int) ($profile->avatar_tone ?? 1),\n                    'avatarUrl' => $profile?->avatar_media_id\n                        ? url('/api/v1/media/'.$profile->avatar_media_id.'/content?variant=thumbnail')\n                        : null,\n                    'relationship' => $relationship,",
    'profile avatar URL payload',
)
replace_once(
    'apps/web/src/lib/profileApi.ts',
    "      avatarTone: number;\n      relationship: 'self' | 'mate' | 'member';",
    "      avatarTone: number;\n      avatarUrl: string | null;\n      relationship: 'self' | 'mate' | 'member';",
    'profile API avatar type',
)

# Profile hero: real image when present, canonical mini Founder badge physically
# anchored to and overlapping the lower-right avatar edge.
replace_once(
    'apps/web/src/screens/UserProfileScreen.tsx',
    '''    <section className="profile-hero" aria-labelledby="profile-name">
      <Avatar initials={initialsFor(data.member.displayName)} tone={data.member.avatarTone} size="xl" label={`${data.member.displayName}'s avatar`}/>
      <div className="profile-hero__copy">
        <div className="profile-hero__name-row"><div><h1 id="profile-name">{data.member.displayName}</h1><p>@{data.member.username}</p></div>{data.member.founder ? <FounderAlphaBadge founder={data.member.founder}/> : null}</div>
''',
    '''    <section className="profile-hero" aria-labelledby="profile-name">
      <div className="profile-hero__avatar">
        <Avatar initials={initialsFor(data.member.displayName)} tone={data.member.avatarTone} size="xl" label={`${data.member.displayName}'s avatar`} src={data.member.avatarUrl}/>
        {data.member.founder ? <span className="profile-hero__founder"><FounderAlphaBadge founder={data.member.founder}/></span> : null}
      </div>
      <div className="profile-hero__copy">
        <div className="profile-hero__name-row"><div><h1 id="profile-name">{data.member.displayName}</h1><p>@{data.member.username}</p></div></div>
''',
    'profile avatar and Founder overlay',
)

# Rating Snapshot: one outer card, one compact three-column principal row,
# filtered result and category controls inside the same card.
old = '''    <section className="profile-rating-overview" aria-labelledby="profile-rating-heading">
      <div className="profile-section-heading"><div><span className="eyebrow">Rating snapshot</span><h2 id="profile-rating-heading">{scopeLabels[data.scope]}</h2></div><ActiveScore summary={data.summary.activeScope}/></div>
      <SegmentedControl label="Profile rating scope" options={scopeOptions} value={data.scope} onChange={(next) => { setCategoryId(null); setCategoryQuery(''); setScope(next); }}/>

      <div className="profile-summary-grid">
        <SummaryCard title="Overall Public Rating" summary={data.summary.overallPublic} detail="Public personal ratings that currently count."/>
        <SummaryCard title="Rate My Rating" summary={data.summary.rateMy} detail="Rate My ratings visible in your current relationship."/>
        {data.summary.matesOnly ? <SummaryCard title="Mates Only Rating" summary={data.summary.matesOnly} detail="Only shown to people entitled to mates-only data."/> : null}
        <SummaryCard title="Filtered Rating" summary={data.summary.filtered} detail={selectedCategory ? `Current category: ${selectedCategory.name}` : 'Choose a category below to calculate this card.'} emptyLabel={selectedCategory ? 'No counted ratings in this category' : 'Select a category'}/>
      </div>

      <div className="profile-category-filter">
        <div><label htmlFor="profile-category-search">Filter this view by category</label><p>Only categories visible in this profile scope are offered.</p></div>
        <div className="profile-category-filter__control">
          <div className="profile-category-search"><Icon name="search" size={18}/><input id="profile-category-search" type="search" value={categoryQuery} placeholder="Search visible categories" autoComplete="off" onChange={(event) => { setCategoryQuery(event.currentTarget.value); if (data.selectedCategoryId) setCategoryId(null); }}/>{data.selectedCategoryId ? <button type="button" onClick={() => { setCategoryId(null); setCategoryQuery(''); }} aria-label="Clear category filter"><Icon name="close" size={17}/></button> : null}</div>
          {visibleCategoryMatches.length ? <div className="profile-category-results" role="listbox" aria-label="Visible category matches">{visibleCategoryMatches.map((category) => <button key={category.id} type="button" role="option" aria-selected={false} onClick={() => { setCategoryQuery(category.name); setCategoryId(category.id); }}>{category.name}</button>)}</div> : null}
          {!data.categories.length ? <span className="profile-category-empty">No visible rating categories in this scope.</span> : null}
        </div>
      </div>
    </section>
'''
new = '''    <section className="profile-rating-overview" aria-labelledby="profile-rating-heading">
      <div className="profile-section-heading"><div><span className="eyebrow">Rating snapshot</span><h2 id="profile-rating-heading">{scopeLabels[data.scope]}</h2></div><ActiveScore summary={data.summary.activeScope}/></div>
      <SegmentedControl label="Profile rating scope" options={scopeOptions} value={data.scope} onChange={(next) => { setCategoryId(null); setCategoryQuery(''); setScope(next); }}/>

      <div className="profile-metrics-row" aria-label="Principal profile ratings">
        <SnapshotMetric label="Overall Public Rating" summary={data.summary.overallPublic}/>
        <SnapshotMetric label="Rate My Rating" summary={data.summary.rateMy}/>
        <SnapshotMetric label="Mates Only Rating" summary={data.summary.matesOnly} unavailable={data.summary.matesOnly === null}/>
      </div>

      <div className="profile-filtered-rating">
        <div className="profile-filtered-rating__value">
          <span>Filtered Rating</span>
          <div><strong>{data.summary.filtered?.score ?? '—'}</strong>{data.summary.filtered?.score !== null && data.summary.filtered !== null ? <small>/100</small> : null}</div>
          <small>{!selectedCategory ? 'Select a category' : data.summary.filtered?.score === null ? 'No counted ratings in this category' : `${data.summary.filtered?.ratingCount ?? 0} counted rating${(data.summary.filtered?.ratingCount ?? 0) === 1 ? '' : 's'}`}</small>
        </div>
        <span className="profile-filtered-rating__category">{selectedCategory?.name ?? 'No category selected'}</span>
      </div>

      <div className="profile-category-filter">
        <div><label htmlFor="profile-category-search">Filter this view by category</label><p>Search the categories available in this view.</p></div>
        <div className="profile-category-filter__control">
          <div className="profile-category-search"><Icon name="search" size={18}/><input id="profile-category-search" type="search" value={categoryQuery} placeholder="Search visible categories" autoComplete="off" onChange={(event) => { setCategoryQuery(event.currentTarget.value); if (data.selectedCategoryId) setCategoryId(null); }}/>{data.selectedCategoryId ? <button type="button" onClick={() => { setCategoryId(null); setCategoryQuery(''); }} aria-label="Clear category filter"><Icon name="close" size={17}/></button> : null}</div>
          {visibleCategoryMatches.length ? <div className="profile-category-results" role="listbox" aria-label="Visible category matches">{visibleCategoryMatches.map((category) => <button key={category.id} type="button" role="option" aria-selected={false} onClick={() => { setCategoryQuery(category.name); setCategoryId(category.id); }}>{category.name}</button>)}</div> : null}
          {!data.categories.length ? <span className="profile-category-empty">No visible rating categories in this scope.</span> : null}
        </div>
      </div>
    </section>
'''
replace_once('apps/web/src/screens/UserProfileScreen.tsx', old, new, 'single-card rating snapshot')

replace_once(
    'apps/web/src/screens/UserProfileScreen.tsx',
    '''function SummaryCard({ title, summary, detail, emptyLabel = 'No counted ratings yet' }: { title: string; summary: ProfileRatingSummary | null; detail: string; emptyLabel?: string }) {
  const score = summary?.score ?? null;
  return <Card className="profile-summary-card"><span>{title}</span>{score === null ? <strong className="profile-summary-card__empty">—</strong> : <div className="profile-summary-card__number"><strong>{score}</strong><small>/100</small></div>}<p>{score === null ? emptyLabel : `${summary?.ratingCount ?? 0} counted rating${(summary?.ratingCount ?? 0) === 1 ? '' : 's'}${summary?.confidenceLabel ? ` · ${summary.confidenceLabel} confidence` : ''}`}</p><small>{detail}</small></Card>;
}
''',
    '''function SnapshotMetric({ label, summary, unavailable = false }: { label: string; summary: ProfileRatingSummary | null; unavailable?: boolean }) {
  const score = summary?.score ?? null;
  const count = summary?.ratingCount ?? 0;
  return <div className="profile-snapshot-metric"><span>{label}</span><div className="profile-snapshot-metric__number"><strong>{score ?? '—'}</strong>{score !== null ? <small>/100</small> : null}</div><small>{unavailable ? 'Not available in this view' : score === null ? 'No counted ratings' : `${count} rating${count === 1 ? '' : 's'}`}</small></div>;
}
''',
    'compact snapshot metric component',
)

replace_once(
    'apps/web/src/screens/UserProfileScreen.tsx',
    '''  if (badge.visual === 'founder') {
    return <div className={`profile-badge-mark profile-badge-mark--founder profile-badge-mark--${badge.founderTier ?? 'bronze'}`} role="img" aria-label={`${badge.founderTier ?? ''} Founder badge number ${badge.founderNumber ?? ''}`}><span>F</span><strong>#{badge.founderNumber}</strong></div>;
  }
''',
    '''  if (badge.visual === 'founder') {
    if (!badge.founderNumber || !badge.founderTier) return null;
    return <div className="profile-badge-mark profile-badge-mark--founder-canonical"><FounderAlphaBadge founder={{ number: badge.founderNumber, tier: badge.founderTier }}/></div>;
  }
''',
    'canonical full Founder badge',
)

# CSS: avatar anchor, responsive selector without overflow, one compact 3-column row,
# compact filtered subsection, and canonical Founder art sizing.
replace_once(
    'apps/web/src/profile.css',
    '''.profile-hero__copy { min-width: 0; display: grid; gap: 12px; }
.profile-hero__name-row { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; }
''',
    '''.profile-hero__avatar { position: relative; width: max-content; flex: 0 0 auto; }
.profile-hero__avatar .avatar--xl { width: 112px; height: 112px; }
.profile-hero__founder { position: absolute; right: -18px; bottom: -16px; width: 56px; height: 56px; display: block; border: 3px solid var(--surface); border-radius: 50%; background: var(--surface); box-shadow: var(--shadow-sm); }
.profile-hero__founder .founder-alpha-approved { width: 100%; height: 100%; margin: 0; }
.profile-hero__copy { min-width: 0; display: grid; gap: 12px; }
.profile-hero__name-row { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; }
''',
    'avatar Founder anchor CSS',
)
replace_once(
    'apps/web/src/profile.css',
    '''.profile-page .segmented { margin-bottom: 18px; }
.profile-rating-overview > .segmented { overflow-x: auto; scrollbar-width: thin; }
.profile-rating-overview > .segmented button { flex: 0 0 auto; min-width: max-content; }

.profile-summary-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}
.profile-summary-card { min-height: 176px; display: flex; flex-direction: column; align-items: flex-start; gap: 7px; padding: 18px; }
.profile-summary-card > span { color: var(--text-muted); font-size: .78rem; font-weight: 850; text-transform: uppercase; letter-spacing: .06em; }
.profile-summary-card__number { display: flex; align-items: baseline; gap: 4px; }
.profile-summary-card__number strong { font-size: 2.5rem; line-height: 1; letter-spacing: -.07em; }
.profile-summary-card__number small { color: var(--text-muted); font-weight: 750; }
.profile-summary-card__empty { font-size: 2.5rem; line-height: 1; color: var(--text-faint); }
.profile-summary-card p { margin: auto 0 0; font-size: .85rem; font-weight: 750; }
.profile-summary-card > small { color: var(--text-muted); line-height: 1.35; }
''',
    '''.profile-page .segmented { margin-bottom: 18px; }
.profile-rating-overview > .segmented {
  grid-auto-flow: row;
  grid-auto-columns: unset;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  overflow: visible;
  overscroll-behavior-inline: auto;
}
.profile-rating-overview > .segmented button { width: 100%; min-width: 0; padding-inline: .5rem; white-space: normal; line-height: 1.15; }

.profile-metrics-row {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 18px;
  background: var(--surface-2);
}
.profile-snapshot-metric { min-width: 0; display: grid; align-content: start; gap: 6px; padding: 14px 12px; }
.profile-snapshot-metric + .profile-snapshot-metric { border-left: 1px solid var(--border); }
.profile-snapshot-metric > span { min-width: 0; color: var(--text-muted); font-size: .72rem; font-weight: 850; line-height: 1.15; text-transform: uppercase; letter-spacing: .045em; overflow-wrap: anywhere; }
.profile-snapshot-metric__number { display: flex; align-items: baseline; gap: 3px; min-width: 0; }
.profile-snapshot-metric__number strong { font-size: clamp(1.65rem, 4vw, 2.2rem); line-height: 1; letter-spacing: -.06em; }
.profile-snapshot-metric__number small, .profile-snapshot-metric > small { color: var(--text-muted); font-size: .72rem; line-height: 1.2; }

.profile-filtered-rating { display: grid; grid-template-columns: minmax(0, 1fr) auto; align-items: center; gap: 12px; margin-top: 14px; padding: 14px 2px; border-top: 1px solid var(--border); border-bottom: 1px solid var(--border); }
.profile-filtered-rating__value { min-width: 0; display: grid; grid-template-columns: auto minmax(0, 1fr); align-items: baseline; gap: 3px 8px; }
.profile-filtered-rating__value > span { grid-column: 1 / -1; color: var(--text-muted); font-size: .72rem; font-weight: 850; text-transform: uppercase; letter-spacing: .055em; }
.profile-filtered-rating__value > div { display: flex; align-items: baseline; gap: 3px; }
.profile-filtered-rating__value strong { font-size: 1.75rem; line-height: 1; letter-spacing: -.05em; }
.profile-filtered-rating__value small { color: var(--text-muted); font-size: .76rem; }
.profile-filtered-rating__category { max-width: 280px; min-width: 0; padding: 6px 9px; border-radius: 999px; background: var(--bg-muted); color: var(--text-muted); font-size: .74rem; font-weight: 750; text-align: right; overflow-wrap: anywhere; }
''',
    'rating snapshot compact CSS',
)
replace_once(
    'apps/web/src/profile.css',
    '''  margin-top: 18px;
  padding-top: 18px;
  border-top: 1px solid var(--border);
''',
    '''  margin-top: 14px;
  padding-top: 0;
  border-top: 0;
''',
    'category filter internal spacing',
)
replace_once(
    'apps/web/src/profile.css',
    '''.profile-badge-mark--founder { align-content: center; gap: 2px; border: 3px solid currentColor; border-radius: 26px; box-shadow: inset 0 0 0 4px rgb(255 255 255 / .45), var(--shadow-sm); font-weight: 900; }
.profile-badge-mark--founder span { font-size: 1.65rem; line-height: 1; }
.profile-badge-mark--founder strong { font-size: .78rem; }
.profile-badge-mark--platinum { background: linear-gradient(135deg, #f9fbfd, #aeb8c1 48%, #edf2f5); color: #242b30; }
.profile-badge-mark--gold { background: linear-gradient(135deg, #fff2ac, #d5a129 48%, #f6dc79); color: #553300; }
.profile-badge-mark--silver { background: linear-gradient(135deg, #f7f8f9, #a8afb5 48%, #e5e7e9); color: #30363a; }
.profile-badge-mark--bronze { background: linear-gradient(135deg, #f4c09f, #b66b41 48%, #dda07c); color: #431f0f; }
''',
    '''.profile-badge-mark--founder-canonical { width: 78px; height: 78px; }
.profile-badge-mark--founder-canonical .founder-alpha-approved { width: 78px; height: 78px; margin: 0; }
''',
    'remove rejected Founder mark CSS',
)
replace_once(
    'apps/web/src/profile.css',
    '''@media (max-width: 900px) {
  .profile-summary-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .profile-persona-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}

@media (max-width: 640px) {
  .profile-page { width: min(100% - 16px, 560px); padding-top: var(--safe-top); }
  .profile-page__top { min-height: 50px; }
  .profile-hero { grid-template-columns: 1fr; justify-items: start; gap: 14px; padding: 18px; border-radius: 22px; }
  .profile-hero__copy { width: 100%; }
  .profile-hero__name-row { align-items: center; }
  .profile-rating-overview, .profile-personas, .profile-history { padding: 16px; border-radius: 22px; }
  .profile-section-heading { align-items: center; }
  .profile-active-score strong { font-size: 1.65rem; }
  .profile-summary-grid, .profile-persona-grid, .profile-badges-grid { grid-template-columns: 1fr; }
  .profile-summary-card { min-height: 150px; }
  .profile-category-filter { grid-template-columns: 1fr; gap: 10px; }
''',
    '''@media (max-width: 900px) {
  .profile-persona-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}

@media (max-width: 640px) {
  .profile-page { width: min(100% - 16px, 560px); padding-top: var(--safe-top); }
  .profile-page__top { min-height: 50px; }
  .profile-hero { grid-template-columns: auto minmax(0, 1fr); justify-items: stretch; gap: 26px; padding: 16px; border-radius: 22px; }
  .profile-hero__avatar .avatar--xl { width: 88px; height: 88px; }
  .profile-hero__founder { right: -15px; bottom: -13px; width: 48px; height: 48px; }
  .profile-hero__copy { width: 100%; }
  .profile-hero__name-row { align-items: center; }
  .profile-rating-overview, .profile-personas, .profile-history { padding: 14px; border-radius: 22px; }
  .profile-section-heading { align-items: center; gap: 10px; margin-bottom: 12px; }
  .profile-active-score strong { font-size: 1.65rem; }
  .profile-rating-overview > .segmented { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .profile-rating-overview > .segmented button { min-height: 42px; padding-inline: .4rem; }
  .profile-snapshot-metric { padding: 11px 7px; gap: 5px; }
  .profile-snapshot-metric > span { font-size: .64rem; letter-spacing: .025em; }
  .profile-snapshot-metric__number strong { font-size: 1.55rem; }
  .profile-snapshot-metric__number small, .profile-snapshot-metric > small { font-size: .66rem; }
  .profile-filtered-rating { grid-template-columns: minmax(0, 1fr) minmax(0, .8fr); gap: 8px; padding-block: 11px; }
  .profile-filtered-rating__category { max-width: none; padding-inline: 7px; font-size: .69rem; }
  .profile-persona-grid, .profile-badges-grid { grid-template-columns: 1fr; }
  .profile-category-filter { grid-template-columns: 1fr; gap: 8px; }
''',
    'mobile profile density and overflow CSS',
)

# Self-declared state reads as established, not neutral/pending.
replace_once(
    'apps/web/src/components/MyIdentitiesCard.tsx',
    "  if (isVerified(item)) return 'positive';\n  if (item.claim.status === 'pending') return 'warning';",
    "  if (isSelfDeclared(item) || isVerified(item)) return 'positive';\n  if (item.claim.status === 'pending') return 'warning';",
    'self declared identity positive tone',
)

# Owner-review source contract covers avatar image, exact overlay anchor, canonical artwork,
# one outer Rating Snapshot, the 3-column row and a real responsive selector.
replace_once(
    'apps/web/src/test/user-profile-owner-review.test.ts',
    "import userProfile from '../screens/UserProfileScreen.tsx?raw';\nimport meScreen from '../screens/MeScreen.tsx?raw';",
    "import userProfile from '../screens/UserProfileScreen.tsx?raw';\nimport founderBadge from '../components/FounderAlphaBadge.tsx?raw';\nimport meScreen from '../screens/MeScreen.tsx?raw';",
    'owner review founder test import',
)
replace_once(
    'apps/web/src/test/user-profile-owner-review.test.ts',
    '''  it('puts the Founder mini chip on the new profile and removes the old standalone placement', () => {
    expect(userProfile).toContain('<FounderAlphaBadge founder={data.member.founder}/>');
    expect(userProfile).toContain('setBadgesOpen(true)');
    expect(userProfile).toContain("badge.visual === 'founder'");
    expect(userProfile).toContain("badge.visual === 'given'");
    expect(meScreen).not.toContain('FounderAlphaBadge');
  });
''',
    '''  it('anchors the canonical Founder artwork to the real profile avatar and uses it again in Badges', () => {
    expect(userProfile).toContain('className="profile-hero__avatar"');
    expect(userProfile).toContain('src={data.member.avatarUrl}');
    expect(userProfile).toContain('className="profile-hero__founder"');
    expect(userProfile).toContain('<FounderAlphaBadge founder={data.member.founder}/>');
    expect(userProfile).toContain('profile-badge-mark--founder-canonical');
    expect(userProfile).not.toContain('<span>F</span><strong>#{badge.founderNumber}</strong>');
    expect(founderBadge).toContain('Approved small RUM flag');
    expect(founderBadge).toContain('founder-alpha-approved');
    expect(founderBadge).toContain('FOUNDER');
    expect(founderBadge).toContain('ALPHA');
    expect(profileCss).toContain('.profile-hero__founder');
    expect(profileCss).toContain('position: absolute');
    expect(profileCss).toContain('right: -15px');
    expect(profileCss).toContain('bottom: -13px');
    expect(meScreen).not.toContain('FounderAlphaBadge');
  });

  it('keeps Rating Snapshot as one compact card with a three-column metric row and non-scrolling mobile selector', () => {
    expect(userProfile).toContain('className="profile-rating-overview"');
    expect(userProfile).toContain('className="profile-metrics-row"');
    expect(userProfile).toContain('className="profile-filtered-rating"');
    expect(userProfile).not.toContain('<SummaryCard');
    expect(profileCss).toContain('.profile-metrics-row');
    expect(profileCss).toContain('grid-template-columns: repeat(3, minmax(0, 1fr));');
    expect(profileCss).toMatch(/\\.profile-rating-overview > \\.segmented \\{[\\s\\S]*?overflow: visible;/);
    expect(profileCss).toMatch(/@media \\(max-width: 640px\\)[\\s\\S]*?\\.profile-rating-overview > \\.segmented \\{ grid-template-columns: repeat\\(2, minmax\\(0, 1fr\\)\\); \\}/);
  });
''',
    'owner review profile visual contract tests',
)

# Fail closed if the rejected visual/layout patterns survived the transform.
profile = (root / 'apps/web/src/screens/UserProfileScreen.tsx').read_text()
css = (root / 'apps/web/src/profile.css').read_text()
founder = (root / 'apps/web/src/components/FounderAlphaBadge.tsx').read_text()
if '<SummaryCard' in profile or 'profile-summary-card' in profile:
    raise SystemExit('rejected nested Rating Snapshot cards remain')
if '<span>F</span><strong>#{badge.founderNumber}</strong>' in profile:
    raise SystemExit('rejected F-number Founder artwork remains')
if 'overflow-x: auto; scrollbar-width: thin;' in css:
    raise SystemExit('rejected Rating Snapshot scrollbar CSS remains')
if 'Approved small RUM flag' not in founder:
    raise SystemExit('canonical Founder artwork missing after transform')
PY

# Basic source sanity before publishing.
grep -Fq 'src={data.member.avatarUrl}' "$tmp/rum/apps/web/src/screens/UserProfileScreen.tsx"
grep -Fq 'profile-metrics-row' "$tmp/rum/apps/web/src/screens/UserProfileScreen.tsx"
grep -Fq 'profile-badge-mark--founder-canonical' "$tmp/rum/apps/web/src/screens/UserProfileScreen.tsx"
grep -Fq "verification_state' => 'self_declared'" "$tmp/rum/apps/api/app/Http/Controllers/Api/V1/EntityClaimController.php"

publish_file() {
  local path="$1" message="$2" expected="$3"
  local current blob body response next
  current="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/${BRANCH}" --jq '.object.sha')"
  [[ "$current" == "$expected" ]] || { echo "PATCH BLOCKED: branch moved before ${path}" >&2; exit 78; }
  blob="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/contents/${path}?ref=${BRANCH}" --jq '.sha')"
  body="$(base64 -w0 "$tmp/rum/$path")"
  response="$(GH_TOKEN="$TOKEN" gh api --method PUT "repos/${REPOSITORY}/contents/${path}" -f message="$message" -f content="$body" -f sha="$blob" -f branch="$BRANCH")"
  next="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["commit"]["sha"])' <<<"$response")"
  [[ "$next" =~ ^[0-9a-f]{40}$ ]] || { echo "PATCH BLOCKED: invalid commit response" >&2; exit 70; }
  printf '%s' "$next"
}

next="$CANDIDATE_SHA"
next="$(publish_file 'apps/web/src/components/FounderAlphaBadge.tsx' 'fix: restore canonical Founder Alpha artwork' "$next")"
next="$(publish_file 'apps/api/app/Http/Controllers/Api/V1/EntityClaimController.php' 'fix: persist self-declared identity ownership' "$next")"
next="$(publish_file 'apps/api/tests/Feature/Entity/RateUrMateMyIdentityProfileTest.php' 'test: lock self-declaration persistence' "$next")"
next="$(publish_file 'apps/api/app/Services/ProfileService.php' 'fix: expose stored profile avatar' "$next")"
next="$(publish_file 'apps/web/src/lib/profileApi.ts' 'fix: type stored profile avatar URL' "$next")"
next="$(publish_file 'apps/web/src/screens/UserProfileScreen.tsx' 'fix: compact owner profile review layout' "$next")"
next="$(publish_file 'apps/web/src/profile.css' 'fix: anchor Founder badge and remove rating overflow' "$next")"
next="$(publish_file 'apps/web/src/components/MyIdentitiesCard.tsx' 'fix: present self declaration as established' "$next")"
next="$(publish_file 'apps/web/src/test/user-profile-owner-review.test.ts' 'test: lock owner profile rejection corrections' "$next")"

final="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/${BRANCH}" --jq '.object.sha')"
[[ "$final" == "$next" ]] || { echo "PATCH BLOCKED: final branch head mismatch" >&2; exit 78; }
printf 'RUM_OWNER_REJECTION_CORRECTIONS_HEAD=%s\n' "$final"
printf 'RUM_PR_153_STATE=OPEN_DRAFT_UNMERGED\n'
printf 'LIVE_MUTATED=NO\n'
printf 'RATE_ANYTHING_MUTATED=NO\n'
