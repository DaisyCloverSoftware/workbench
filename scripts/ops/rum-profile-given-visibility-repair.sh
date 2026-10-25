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
for command in gh git python3 base64 mktemp; do command -v "$command" >/dev/null 2>&1 || { echo "missing $command" >&2; exit 2; }; done
TOKEN="${GH_TOKEN:-}"; [[ -n "$TOKEN" ]] || TOKEN="$(gh auth token 2>/dev/null || true)"
[[ -n "$TOKEN" ]] || { echo "GitHub token unavailable" >&2; exit 2; }

head="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/${BRANCH}" --jq '.object.sha')"
[[ "$head" == "$CANDIDATE_SHA" ]] || { echo "PATCH BLOCKED: branch moved" >&2; exit 78; }
pr_state="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/pulls/${PR}" --jq '[.state, (.draft|tostring), .head.sha, (.merged_at // "")] | @tsv')"
IFS=$'\t' read -r state draft pr_head merged_at <<<"$pr_state"
[[ "$state" == open && "$draft" == true && "$pr_head" == "$CANDIDATE_SHA" && -z "$merged_at" ]] || { echo "PATCH BLOCKED: PR not open/draft/unmerged exact head" >&2; exit 78; }

tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT HUP INT TERM
GH_TOKEN="$TOKEN" gh repo clone "$REPOSITORY" "$tmp/rum" -- --no-checkout --filter=blob:none >/dev/null
git -C "$tmp/rum" checkout --detach "$CANDIDATE_SHA" >/dev/null

python3 - "$tmp/rum/$SERVICE" "$tmp/rum/$TEST" <<'PY'
from pathlib import Path
import sys
service=Path(sys.argv[1]); test=Path(sys.argv[2])
text=service.read_text()

def replace_once(source, old, new, label):
    count=source.count(old)
    if count != 1:
        raise SystemExit(f'{label} marker count={count}')
    return source.replace(old,new,1)

text=replace_once(text,
"        private readonly CanonicalJudgeService $canonicalJudge,\n        private readonly JudgeEligibility $judgeEligibility,\n    ) {}",
"        private readonly CanonicalJudgeService $canonicalJudge,\n        private readonly JudgeEligibility $judgeEligibility,\n        private readonly UniversalEntityVisibilityService $entityVisibility,\n    ) {}",
'ProfileService constructor')
text=replace_once(text,
"            ->get()\n            ->map(fn (RatingEvent $event): array => $this->canonicalHistoryItem($event, $viewer))\n            ->all();\n\n        $legacyAudiences = array_values(array_intersect($audiences, ['public', 'mates', 'private']));",
"            ->get()\n            ->filter(fn (RatingEvent $event): bool => $this->givenTargetVisibleToViewer($event, $viewer, $member))\n            ->values()\n            ->map(fn (RatingEvent $event): array => $this->canonicalHistoryItem($event, $viewer))\n            ->all();\n\n        $legacyAudiences = array_values(array_intersect($audiences, ['public', 'mates', 'private']));",
'given-history filter')
marker="    /**\n     * @param  list<string>  $personaIds\n     * @return list<array<string,mixed>>\n     */\n    private function historyReceived"
method="""    private function givenTargetVisibleToViewer(RatingEvent $event, User $viewer, User $member): bool
    {
        if ((string) $viewer->id === (string) $member->id) {
            return true;
        }

        $target = $event->targetEntity;
        if (! $target instanceof Entity) {
            return false;
        }

        $query = Entity::query()
            ->whereKey($target->id)
            ->whereNotIn('status', ['deleted', 'merged']);
        $this->entityVisibility->constrain($query, $viewer);

        return $query->exists();
    }

    /**
     * @param  list<string>  $personaIds
     * @return list<array<string,mixed>>
     */
    private function historyReceived"""
text=replace_once(text, marker, method, 'given visibility method')
service.write_text(text)

t=test.read_text()
test_marker="    public function test_badges_derive_from_historical_data_and_founder_source_of_truth(): void\n"
new_test="""    public function test_ratings_given_hide_non_visible_canonical_entity_targets_from_other_profile_viewers(): void
    {
        $member = $this->user('History owner', 'members');
        $viewer = $this->user('History viewer', 'members');
        $targetOwner = $this->user('Hidden target owner', 'members');
        $hiddenTarget = $this->identityFor($this->personFor($targetOwner), 'Hidden Given Identity', 'persona', 'private');
        $context = RatingContext::query()->where('status', 'active')->firstOrFail();
        $event = $this->canonicalRating(
            $member,
            $hiddenTarget,
            $context,
            RateAnythingRatingService::GENERAL_SCHEME_KEY,
            'public',
            4,
            true,
        );

        Sanctum::actingAs($viewer);
        $otherView = $this->getJson("/api/v1/people/{$member->id}/profile")->assertOk();
        self::assertFalse(collect($otherView->json('data.ratingsGiven'))->contains(
            fn (array $item): bool => (string) $item['id'] === (string) $event->id,
        ));
        self::assertStringNotContainsString('Hidden Given Identity', $otherView->getContent());

        Sanctum::actingAs($member);
        $selfView = $this->getJson("/api/v1/people/{$member->id}/profile")->assertOk();
        $item = collect($selfView->json('data.ratingsGiven'))->firstWhere('id', (string) $event->id);
        self::assertNotNull($item);
        self::assertSame('Hidden Given Identity', $item['target']['displayName']);
    }

"""
t=replace_once(t, test_marker, new_test+test_marker, 'given-history privacy test')
test.write_text(t)
PY

current="$CANDIDATE_SHA"
for path in "$SERVICE" "$TEST"; do
  blob="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/contents/${path}?ref=${BRANCH}" --jq '.sha')"
  body="$(base64 -w0 "$tmp/rum/$path")"
  message='fix: fail closed on profile given-history entity visibility'
  [[ "$path" == "$TEST" ]] && message='test: lock profile given-history entity privacy'
  response="$(GH_TOKEN="$TOKEN" gh api --method PUT "repos/${REPOSITORY}/contents/${path}" -f message="$message" -f content="$body" -f sha="$blob" -f branch="$BRANCH")"
  current="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["commit"]["sha"])' <<<"$response")"
  [[ "$current" =~ ^[0-9a-f]{40}$ ]] || { echo "PATCH BLOCKED: invalid commit response" >&2; exit 70; }
done
final="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/${BRANCH}" --jq '.object.sha')"
[[ "$final" == "$current" ]] || { echo "PATCH BLOCKED: final branch head mismatch" >&2; exit 78; }
printf 'RUM_PROFILE_GIVEN_VISIBILITY_HEAD=%s\n' "$final"
printf 'THIRD_PARTY_HIDDEN_ENTITY_HISTORY=FAIL_CLOSED\n'
printf 'SELF_HISTORY_PRESERVED=YES\n'
