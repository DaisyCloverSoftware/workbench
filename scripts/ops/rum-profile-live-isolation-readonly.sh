#!/usr/bin/env bash
set -euo pipefail
umask 077

if [[ $# -ne 6 ]]; then
  echo "usage: $0 <exact-rum-candidate-sha> <rating-rater-email> <mate-email> <linked-thing-name> <self-email> <self-declared-gamertag>" >&2
  exit 64
fi

CANDIDATE_SHA="$1"
RATER_EMAIL="$2"
MATE_EMAIL="$3"
LINKED_NAME="$4"
SELF_EMAIL="$5"
GAMERTAG="$6"
[[ "$CANDIDATE_SHA" =~ ^[0-9a-f]{40}$ ]] || { echo "full lowercase candidate SHA required" >&2; exit 64; }

REPOSITORY="DaisyCloverSoftware/rum"
CANDIDATE_BRANCH="sprint-0-rum-owner-rating-flow-20260823"
CANDIDATE_PR=153
DEV_NAMESPACE="rum-dev-isolated"
LIVE_NAMESPACE="rum-dev"
DEV_HOST="dev-rum.daisycloversoftware.uk"
LIVE_HOST="rateurmate.online"

for command in gh python3 grep; do
  command -v "$command" >/dev/null 2>&1 || { echo "required command unavailable: $command" >&2; exit 2; }
done
TOKEN="${GH_TOKEN:-}"; [[ -n "$TOKEN" ]] || TOKEN="$(gh auth token 2>/dev/null || true)"
[[ -n "$TOKEN" ]] || { echo "GitHub token unavailable" >&2; exit 2; }

branch_sha="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/${CANDIDATE_BRANCH}" --jq '.object.sha')"
[[ "$branch_sha" == "$CANDIDATE_SHA" ]] || { echo "ISOLATION BLOCKED: candidate branch moved" >&2; exit 78; }
pr_state="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/pulls/${CANDIDATE_PR}" --jq '[.state, (.draft|tostring), .head.sha, (.merged_at // "")] | @tsv')"
IFS=$'\t' read -r state draft pr_head merged_at <<<"$pr_state"
[[ "$state" == open && "$draft" == true && "$pr_head" == "$CANDIDATE_SHA" && -z "$merged_at" ]] || {
  echo "ISOLATION BLOCKED: PR #${CANDIDATE_PR} is not open/draft/unmerged at exact head" >&2
  exit 78
}

kctl(){ if command -v k3s >/dev/null 2>&1; then sudo k3s kubectl "$@"; else kubectl "$@"; fi; }
dev_hosts="$(kctl -n "$DEV_NAMESPACE" get ingress rum -o jsonpath='{range .spec.rules[*]}{.host}{"\n"}{end}' 2>/dev/null || true)"
live_hosts="$(kctl -n "$LIVE_NAMESPACE" get ingress rum -o jsonpath='{range .spec.rules[*]}{.host}{"\n"}{end}' 2>/dev/null || true)"
[[ "$dev_hosts" == "$DEV_HOST" ]] || { echo "ISOLATION BLOCKED: DEV ingress mismatch" >&2; exit 78; }
[[ "$live_hosts" == "$LIVE_HOST" ]] || { echo "ISOLATION BLOCKED: LIVE ingress mismatch" >&2; exit 78; }

dev_api_pod="$(kctl -n "$DEV_NAMESPACE" get pods -l 'app.kubernetes.io/instance=rum,app.kubernetes.io/component=api' --field-selector=status.phase=Running -o jsonpath='{.items[0].metadata.name}')"
live_api_pod="$(kctl -n "$LIVE_NAMESPACE" get pods -l 'app.kubernetes.io/instance=rum,app.kubernetes.io/component=api' --field-selector=status.phase=Running -o jsonpath='{.items[0].metadata.name}')"
[[ -n "$dev_api_pod" && -n "$live_api_pod" ]] || { echo "ISOLATION BLOCKED: DEV or LIVE API pod unavailable" >&2; exit 78; }

b64(){ printf '%s' "$1" | python3 -c 'import base64,sys; print(base64.b64encode(sys.stdin.buffer.read()).decode())'; }
rater_b64="$(b64 "$RATER_EMAIL")"; mate_b64="$(b64 "$MATE_EMAIL")"; linked_b64="$(b64 "$LINKED_NAME")"; self_b64="$(b64 "$SELF_EMAIL")"; gamer_b64="$(b64 "$GAMERTAG")"

# SELECT/count/first reads only. Fail closed if a future edit adds any known
# write-capable Eloquent/SQL primitive before this payload is allowed near LIVE.
read -r -d '' PHP_QUERY <<'PHP' || true
require "/var/www/html/vendor/autoload.php";
$app=require "/var/www/html/bootstrap/app.php";
$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$raterEmail=base64_decode($argv[1]);
$mateEmail=base64_decode($argv[2]);
$linkedName=base64_decode($argv[3]);
$selfEmail=base64_decode($argv[4]);
$gamerTag=base64_decode($argv[5]);
$rater=App\Models\User::query()->where('email',$raterEmail)->first();
$self=App\Models\User::query()->where('email',$selfEmail)->first();
$linked=App\Models\Entity::query()->where('canonical_name',$linkedName)->first();
$gamer=App\Models\Entity::query()->where('canonical_name',$gamerTag)->first();
$result=[
  'ratingRater'=>App\Models\User::query()->where('email',$raterEmail)->count(),
  'mate'=>App\Models\User::query()->where('email',$mateEmail)->count(),
  'linked'=>App\Models\Entity::query()->where('canonical_name',$linkedName)->count(),
  'linkedRatings'=>($linked && $rater) ? App\Models\RatingEvent::query()->where('target_entity_id',$linked->id)->where('rater_account_id',$rater->id)->count() : 0,
  'selfUser'=>App\Models\User::query()->where('email',$selfEmail)->count(),
  'gamer'=>App\Models\Entity::query()->where('canonical_name',$gamerTag)->count(),
  'selfClaims'=>($gamer && $self) ? App\Models\EntityClaim::query()->where('entity_id',$gamer->id)->where('claimant_user_id',$self->id)->count() : 0,
];
echo json_encode($result, JSON_UNESCAPED_SLASHES);
PHP

if grep -Eqi -- '::create\(|->save\(|->update\(|->delete\(|::update\(|::delete\(|->forceFill\(|insert\(|upsert\(' <<<"$PHP_QUERY"; then
  echo "ISOLATION BLOCKED: LIVE probe contains a write-capable operation" >&2
  exit 78
fi
probe(){ local ns="$1" pod="$2"; kctl -n "$ns" exec "$pod" -c php-fpm -- php -r "$PHP_QUERY" "$rater_b64" "$mate_b64" "$linked_b64" "$self_b64" "$gamer_b64" 2>/dev/null | tail -n 1; }
dev_json="$(probe "$DEV_NAMESPACE" "$dev_api_pod")"
live_json="$(probe "$LIVE_NAMESPACE" "$live_api_pod")"

read_counts(){ python3 - "$1" <<'PY'
import json,sys
x=json.loads(sys.argv[1]); print(x['ratingRater'],x['mate'],x['linked'],x['linkedRatings'],x['selfUser'],x['gamer'],x['selfClaims'])
PY
}
read -r dev_rater dev_mate dev_linked dev_rating dev_self dev_gamer dev_claim <<<"$(read_counts "$dev_json")"
read -r live_rater live_mate live_linked live_rating live_self live_gamer live_claim <<<"$(read_counts "$live_json")"
[[ "$dev_rater" -ge 1 && "$dev_mate" -ge 1 && "$dev_linked" -ge 1 && "$dev_rating" -ge 1 && "$dev_self" -ge 1 && "$dev_gamer" -ge 1 && "$dev_claim" -ge 1 ]] || {
  echo "ISOLATION FAILED: expected exact verifier fixtures are not all present in isolated DEV" >&2; exit 1;
}
[[ "$live_rater" == 0 && "$live_mate" == 0 && "$live_linked" == 0 && "$live_rating" == 0 && "$live_self" == 0 && "$live_gamer" == 0 && "$live_claim" == 0 ]] || {
  echo "ISOLATION FAILED: one or more exact isolated-DEV verifier fixtures exist in LIVE" >&2; exit 1;
}
printf 'RUM_CANDIDATE_SHA=%s\n' "$CANDIDATE_SHA"
printf 'DEV_FIXTURE_COUNTS=ratingRater:%s mate:%s linked:%s linkedRatings:%s selfUser:%s gamer:%s selfClaims:%s\n' "$dev_rater" "$dev_mate" "$dev_linked" "$dev_rating" "$dev_self" "$dev_gamer" "$dev_claim"
printf 'LIVE_FIXTURE_COUNTS=ratingRater:%s mate:%s linked:%s linkedRatings:%s selfUser:%s gamer:%s selfClaims:%s\n' "$live_rater" "$live_mate" "$live_linked" "$live_rating" "$live_self" "$live_gamer" "$live_claim"
printf 'DEV_FIXTURES_PRESENT=YES\nLIVE_FIXTURES_ABSENT=YES\nLIVE_READ_ONLY=YES\nLIVE_MUTATION=NO\nRATE_ANYTHING_AFFECTED=NO\n'
