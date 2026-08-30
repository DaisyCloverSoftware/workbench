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
SERVICE="apps/api/app/Services/ProfileService.php"
TEST="apps/api/tests/Feature/Profile/UserProfileTest.php"

for command in gh git python3 mktemp base64; do
  command -v "$command" >/dev/null 2>&1 || { echo "required command unavailable: $command" >&2; exit 2; }
done
TOKEN="${GH_TOKEN:-}"
[[ -n "$TOKEN" ]] || TOKEN="$(gh auth token 2>/dev/null || true)"
[[ -n "$TOKEN" ]] || { echo "GitHub token unavailable" >&2; exit 2; }

assert_candidate_current() {
  local head pr_state state draft pr_head merged_at
  head="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/${BRANCH}" --jq '.object.sha')"
  [[ "$head" == "$CANDIDATE_SHA" ]] || { echo "PATCH BLOCKED: RUM branch moved" >&2; exit 78; }
  pr_state="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/pulls/${PR}" --jq '[.state, (.draft|tostring), .head.sha, (.merged_at // "")] | @tsv')"
  IFS=$'\t' read -r state draft pr_head merged_at <<<"$pr_state"
  [[ "$state" == open && "$draft" == true && "$pr_head" == "$CANDIDATE_SHA" && -z "$merged_at" ]] || {
    echo "PATCH BLOCKED: PR #${PR} is not open/draft/unmerged exact head" >&2
    exit 78
  }
}

assert_candidate_current
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT HUP INT TERM
GH_TOKEN="$TOKEN" gh repo clone "$REPOSITORY" "$work/rum" -- --no-checkout --filter=blob:none >/dev/null
git -C "$work/rum" checkout --detach "$CANDIDATE_SHA" >/dev/null
[[ "$(git -C "$work/rum" rev-parse HEAD)" == "$CANDIDATE_SHA" ]] || { echo "PATCH BLOCKED: checkout mismatch" >&2; exit 78; }

python3 - "$work/rum/$SERVICE" "$work/rum/$TEST" <<'PY'
from pathlib import Path
import sys

service = Path(sys.argv[1])
test = Path(sys.argv[2])
text = service.read_text()

old_ctor = """        private readonly CanonicalJudgeService $canonicalJudge,\n        private readonly JudgeEligibility $judgeEligibility,\n"""
new_ctor = """        private readonly CanonicalJudgeService $canonicalJudge,\n        private readonly JudgeEligibility $judgeEligibility,\n        private readonly UniversalEntityVisibilityService $entityVisibility,\n"""
if text.count(old_ctor) != 1:
    raise SystemExit('ProfileService constructor marker mismatch')
text = text.replace(old_ctor, new_ctor, 1)

old_given = """        $canonical = RatingEvent::query()\n            ->with(['rater.profile', 'targetEntity.accountLinks.user.profile', 'context', 'reasonSelections', 'review', 'admissionState'])\n            ->where('rater_account_id', $member->id)\n            ->where('status', 'active')\n            ->whereIn('audience', $audiences)\n            ->orderByDesc('submitted_at')\n            ->limit(self::HISTORY_LIMIT)\n            ->get()\n            ->map(fn (RatingEvent $event): array => $this->canonicalHistoryItem($event, $viewer))\n            ->all();\n"""
new_given = """        $canonicalQuery = RatingEvent::query()\n            ->with(['rater.profile', 'targetEntity.accountLinks.user.profile', 'context', 'reasonSelections', 'review', 'admissionState'])\n            ->where('rater_account_id', $member->id)\n            ->where('status', 'active')\n            ->whereIn('audience', $audiences);\n        if ((string) $viewer->id !== (string) $member->id) {\n            $canonicalQuery->whereHas('targetEntity', function (Builder $target) use ($viewer): void {\n                $this->entityVisibility->constrain($target, $viewer);\n            });\n        }\n        $canonical = $canonicalQuery\n            ->orderByDesc('submitted_at')\n            ->limit(self::HISTORY_LIMIT)\n            ->get()\n            ->map(fn (RatingEvent $event): array => $this->canonicalHistoryItem($event, $viewer))\n            ->all();\n"""
if text.count(old_given) != 1:
    raise SystemExit('ProfileService historyGiven marker mismatch')
service.write_text(text.replace(old_given, new_given, 1))

text = test.read_text()
marker = """    public function test_help_me_out_is_owner_only_atomic_idempotent_and_privacy_aware(): void\n"""
if text.count(marker) != 1:
    raise SystemExit('UserProfileTest insertion marker mismatch')
case = """    public function test_ratings_given_hides_canonical_targets_not_visible_to_profile_viewer_but_keeps_owner_history(): void\n    {\n        $member = $this->user('Given history owner', 'members');\n        $outsider = $this->user('Given history viewer', 'members');\n        $type = EntityType::query()->whereNotIn('key', ['person', 'persona', 'digital_identity'])->firstOrFail();\n        $target = Entity::query()->create([\n            'entity_type_id' => $type->id,\n            'canonical_name' => 'Hidden profile target',\n            'canonical_slug' => 'hidden-profile-target-'.Str::lower(Str::random(5)),\n            'name_normalised' => 'hidden profile target',\n            'visibility' => 'private',\n            'publication_state' => 'published',\n            'status' => 'active',\n            'rateability' => 'open',\n            'reputation_state' => 'active',\n        ]);\n        $context = RatingContext::query()->where('status', 'active')->orderBy('display_order')->firstOrFail();\n        $event = $this->canonicalRating(\n            $member,\n            $target,\n            $context,\n            RateAnythingRatingService::GENERAL_SCHEME_KEY,\n            'public',\n            5,\n            true,\n        );\n\n        Sanctum::actingAs($outsider);\n        $outside = $this->getJson(\"/api/v1/people/{$member->id}/profile\")->assertOk();\n        self::assertFalse(collect($outside->json('data.ratingsGiven'))->contains(\n            fn (array $item): bool => (string) $item['id'] === (string) $event->id,\n        ));\n        self::assertStringNotContainsString('Hidden profile target', $outside->getContent());\n\n        Sanctum::actingAs($member);\n        $owner = $this->getJson(\"/api/v1/people/{$member->id}/profile\")->assertOk();\n        $item = collect($owner->json('data.ratingsGiven'))->firstWhere('id', (string) $event->id);\n        self::assertIsArray($item);\n        self::assertSame('Hidden profile target', $item['target']['displayName']);\n    }\n\n"""
test.write_text(text.replace(marker, case + marker, 1))
PY

# Build one atomic two-file commit, then fast-forward only if the branch still
# points at the exact reviewed candidate.
base_tree="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/commits/${CANDIDATE_SHA}" --jq '.tree.sha')"
service_blob="$(GH_TOKEN="$TOKEN" gh api --method POST "repos/${REPOSITORY}/git/blobs" \
  -f content="$(base64 -w0 "$work/rum/$SERVICE")" -f encoding=base64 --jq '.sha')"
test_blob="$(GH_TOKEN="$TOKEN" gh api --method POST "repos/${REPOSITORY}/git/blobs" \
  -f content="$(base64 -w0 "$work/rum/$TEST")" -f encoding=base64 --jq '.sha')"
tree_payload="$(python3 - "$base_tree" "$SERVICE" "$service_blob" "$TEST" "$test_blob" <<'PY'
import json,sys
base,service,service_blob,test,test_blob=sys.argv[1:]
print(json.dumps({'base_tree':base,'tree':[
    {'path':service,'mode':'100644','type':'blob','sha':service_blob},
    {'path':test,'mode':'100644','type':'blob','sha':test_blob},
]}))
PY
)"
new_tree="$(GH_TOKEN="$TOKEN" gh api --method POST "repos/${REPOSITORY}/git/trees" --input - --jq '.sha' <<<"$tree_payload")"
commit_payload="$(python3 - "$new_tree" "$CANDIDATE_SHA" <<'PY'
import json,sys
print(json.dumps({'message':'fix: protect profile given-history entity privacy','tree':sys.argv[1],'parents':[sys.argv[2]]}))
PY
)"
new_commit="$(GH_TOKEN="$TOKEN" gh api --method POST "repos/${REPOSITORY}/git/commits" --input - --jq '.sha' <<<"$commit_payload")"
[[ "$new_commit" =~ ^[0-9a-f]{40}$ ]] || { echo "PATCH BLOCKED: invalid commit response" >&2; exit 70; }
assert_candidate_current
GH_TOKEN="$TOKEN" gh api --method PATCH "repos/${REPOSITORY}/git/refs/heads/${BRANCH}" -f sha="$new_commit" -F force=false >/dev/null
final="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/${BRANCH}" --jq '.object.sha')"
[[ "$final" == "$new_commit" ]] || { echo "PATCH BLOCKED: final branch head mismatch" >&2; exit 78; }

printf 'RUM_PROFILE_GIVEN_PRIVACY_HEAD=%s\n' "$final"
printf 'RATINGS_GIVEN_ENTITY_VISIBILITY=CANONICAL_SERVICE\n'
printf 'OWNER_SELF_HISTORY=PRESERVED\n'
printf 'LIVE_RUNTIME_AFFECTED=NO\n'
printf 'RATE_ANYTHING_AFFECTED=NO\n'
