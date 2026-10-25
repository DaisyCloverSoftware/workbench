#!/usr/bin/env bash
set -euo pipefail
umask 077

if [[ $# -ne 5 ]]; then
  echo "usage: $0 <exact-rum-candidate-sha> <rater-email> <mate-email> <linked-thing-name> <self-declared-gamertag>" >&2
  exit 64
fi

CANDIDATE_SHA="$1"
RATER_EMAIL="$2"
MATE_EMAIL="$3"
LINKED_NAME="$4"
GAMERTAG="$5"
[[ "$CANDIDATE_SHA" =~ ^[0-9a-f]{40}$ ]] || { echo "full lowercase candidate SHA required" >&2; exit 64; }

REPOSITORY="DaisyCloverSoftware/rum"
CANDIDATE_BRANCH="sprint-0-rum-owner-rating-flow-20260823"
CANDIDATE_PR="153"
DEV_NAMESPACE="rum-dev-isolated"
LIVE_NAMESPACE="rum-dev"
DEV_HOST="dev-rum.daisycloversoftware.uk"
LIVE_HOST="rateurmate.online"

for command in gh python3 grep; do
  command -v "$command" >/dev/null 2>&1 || { echo "required command unavailable: $command" >&2; exit 2; }
done

TOKEN="${GH_TOKEN:-}"
[[ -n "$TOKEN" ]] || TOKEN="$(gh auth token 2>/dev/null || true)"
[[ -n "$TOKEN" ]] || { echo "GitHub token unavailable" >&2; exit 2; }

branch_sha="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/${CANDIDATE_BRANCH}" --jq '.object.sha')"
[[ "$branch_sha" == "$CANDIDATE_SHA" ]] || { echo "ISOLATION BLOCKED: candidate branch moved" >&2; exit 78; }
pr_state="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/pulls/${CANDIDATE_PR}" --jq '[.state, (.draft|tostring), .head.sha, (.merged_at // "")] | @tsv')"
IFS=$'\t' read -r state draft pr_head merged_at <<<"$pr_state"
[[ "$state" == "open" && "$draft" == "true" && "$pr_head" == "$CANDIDATE_SHA" && -z "$merged_at" ]] || {
  echo "ISOLATION BLOCKED: PR #${CANDIDATE_PR} is not open/draft/unmerged at the exact candidate." >&2
  exit 78
}

kctl() {
  if command -v k3s >/dev/null 2>&1; then
    sudo k3s kubectl "$@"
  else
    kubectl "$@"
  fi
}

dev_hosts="$(kctl -n "$DEV_NAMESPACE" get ingress rum -o jsonpath='{range .spec.rules[*]}{.host}{"\n"}{end}' 2>/dev/null || true)"
live_hosts="$(kctl -n "$LIVE_NAMESPACE" get ingress rum -o jsonpath='{range .spec.rules[*]}{.host}{"\n"}{end}' 2>/dev/null || true)"
[[ "$dev_hosts" == "$DEV_HOST" ]] || { echo "ISOLATION BLOCKED: isolated DEV ingress is not exactly ${DEV_HOST}." >&2; exit 78; }
[[ "$live_hosts" == "$LIVE_HOST" ]] || { echo "ISOLATION BLOCKED: LIVE ingress is not exactly ${LIVE_HOST}." >&2; exit 78; }

dev_api_pod="$(kctl -n "$DEV_NAMESPACE" get pods -l 'app.kubernetes.io/instance=rum,app.kubernetes.io/component=api' --field-selector=status.phase=Running -o jsonpath='{.items[0].metadata.name}')"
live_api_pod="$(kctl -n "$LIVE_NAMESPACE" get pods -l 'app.kubernetes.io/instance=rum,app.kubernetes.io/component=api' --field-selector=status.phase=Running -o jsonpath='{.items[0].metadata.name}')"
[[ -n "$dev_api_pod" && -n "$live_api_pod" ]] || { echo "ISOLATION BLOCKED: DEV or LIVE API pod is unavailable." >&2; exit 78; }

b64() { printf '%s' "$1" | python3 -c 'import base64,sys; print(base64.b64encode(sys.stdin.buffer.read()).decode())'; }
rater_b64="$(b64 "$RATER_EMAIL")"
mate_b64="$(b64 "$MATE_EMAIL")"
linked_b64="$(b64 "$LINKED_NAME")"
gamer_b64="$(b64 "$GAMERTAG")"

# The PHP payload below contains SELECT/count/value reads only. Keep a fail-
# closed lexical guard so a future edit cannot silently introduce LIVE writes.
read -r -d '' PHP_QUERY <<'PHP' || true
require "/var/www/html/vendor/autoload.php";
$app=require "/var/www/html/bootstrap/app.php";
$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$raterEmail=base64_decode($argv[1]);
$mateEmail=base64_decode($argv[2]);
$linkedName=base64_decode($argv[3]);
$gamerTag=base64_decode($argv[4]);
$rater=App\Models\User::query()->where('email',$raterEmail)->first();
$linked=App\Models\Entity::query()->where('canonical_name',$linkedName)->first();
$gamer=App\Models\Entity::query()->where('canonical_name',$gamerTag)->first();
$result=[
  'rater'=>App\Models\User::query()->where('email',$raterEmail)->count(),
  'mate'=>App\Models\User::query()->where('email',$mateEmail)->count(),
  'linked'=>App\Models\Entity::query()->where('canonical_name',$linkedName)->count(),
  'linkedRatings'=>$linked ? App\Models\RatingEvent::query()->where('target_entity_id',$linked->id)->count() : 0,
  'gamer'=>App\Models\Entity::query()->where('canonical_name',$gamerTag)->count(),
  'claims'=>($gamer && $rater) ? App\Models\EntityClaim::query()->where('entity_id',$gamer->id)->where('claimant_user_id',$rater->id)->count() : 0,
];
echo json_encode($result, JSON_UNESCAPED_SLASHES);
PHP

if grep -Eqi -- '::create\(|->save\(|->update\(|->delete\(|::update\(|::delete\(|->forceFill\(|insert\(|upsert\(' <<<"$PHP_QUERY"; then
  echo "ISOLATION BLOCKED: LIVE probe contains a write-capable operation." >&2
  exit 78
fi

probe_namespace() {
  local namespace="$1" pod="$2"
  kctl -n "$namespace" exec "$pod" -c php-fpm -- php -r "$PHP_QUERY" "$rater_b64" "$mate_b64" "$linked_b64" "$gamer_b64" 2>/dev/null | tail -n 1
}

dev_json="$(probe_namespace "$DEV_NAMESPACE" "$dev_api_pod")"
live_json="$(probe_namespace "$LIVE_NAMESPACE" "$live_api_pod")"

read -r dev_rater dev_mate dev_linked dev_ratings dev_gamer dev_claims <<<"$(python3 - "$dev_json" <<'PY'
import json,sys
x=json.loads(sys.argv[1])
print(x['rater'],x['mate'],x['linked'],x['linkedRatings'],x['gamer'],x['claims'])
PY
)"
read -r live_rater live_mate live_linked live_ratings live_gamer live_claims <<<"$(python3 - "$live_json" <<'PY'
import json,sys
x=json.loads(sys.argv[1])
print(x['rater'],x['mate'],x['linked'],x['linkedRatings'],x['gamer'],x['claims'])
PY
)"

[[ "$dev_rater" -ge 1 && "$dev_mate" -ge 1 && "$dev_linked" -ge 1 && "$dev_ratings" -ge 1 && "$dev_gamer" -ge 1 && "$dev_claims" -ge 1 ]] || {
  echo "ISOLATION FAILED: expected verifier fixtures are not all present in isolated DEV." >&2
  exit 1
}
[[ "$live_rater" == "0" && "$live_mate" == "0" && "$live_linked" == "0" && "$live_ratings" == "0" && "$live_gamer" == "0" && "$live_claims" == "0" ]] || {
  echo "ISOLATION FAILED: one or more exact isolated-DEV verifier fixtures exist in LIVE." >&2
  exit 1
}

printf 'RUM_CANDIDATE_SHA=%s\n' "$CANDIDATE_SHA"
printf 'DEV_FIXTURE_COUNTS=rater:%s mate:%s linked:%s ratings:%s gamer:%s claims:%s\n' "$dev_rater" "$dev_mate" "$dev_linked" "$dev_ratings" "$dev_gamer" "$dev_claims"
printf 'LIVE_FIXTURE_COUNTS=rater:%s mate:%s linked:%s ratings:%s gamer:%s claims:%s\n' "$live_rater" "$live_mate" "$live_linked" "$live_ratings" "$live_gamer" "$live_claims"
printf 'DEV_FIXTURES_PRESENT=YES\n'
printf 'LIVE_FIXTURES_ABSENT=YES\n'
printf 'LIVE_READ_ONLY=YES\n'
printf 'LIVE_MUTATION=NO\n'
printf 'RATE_ANYTHING_AFFECTED=NO\n'
