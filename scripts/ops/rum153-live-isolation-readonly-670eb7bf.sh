#!/usr/bin/env bash
set -euo pipefail
umask 077

CANDIDATE_SHA="670eb7bfe646bcd4f1aca80b003323da7b801433"
REPOSITORY="DaisyCloverSoftware/rum"
CANDIDATE_BRANCH="sprint-0-rum-owner-rating-flow-20260823"
CANDIDATE_PR="153"
DEV_NAMESPACE="rum-dev-isolated"
LIVE_NAMESPACE="rum-dev"
RELEASE="rum"
DEV_HOST="dev-rum.daisycloversoftware.uk"
LIVE_HOST="rateurmate.online"
RATER_EMAIL="rumowner17875470673812@example.com"
MATE_EMAIL="rummate17875470673812@example.com"
LINKED_NAME="Quillstone17875470673812Zeta"
GAMERTAG="RUMSelf17875470673812"

for command in gh python3; do
  command -v "$command" >/dev/null 2>&1 || { echo "READONLY CHECK BLOCKED: missing $command" >&2; exit 2; }
done
TOKEN="${GH_TOKEN:-}"
if [[ -z "$TOKEN" ]]; then TOKEN="$(gh auth token 2>/dev/null || true)"; fi
[[ -n "$TOKEN" ]] || { echo "READONLY CHECK BLOCKED: no GitHub token" >&2; exit 2; }

branch_sha="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/${CANDIDATE_BRANCH}" --jq '.object.sha')"
[[ "$branch_sha" == "$CANDIDATE_SHA" ]] || { echo "READONLY CHECK BLOCKED: candidate branch moved" >&2; exit 78; }
pr_state="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/pulls/${CANDIDATE_PR}" --jq '[.state, (.draft|tostring), .head.sha, (.merged_at // "")] | @tsv')"
IFS=$'\t' read -r state draft pr_head merged_at <<<"$pr_state"
[[ "$state" == "open" && "$draft" == "true" && "$pr_head" == "$CANDIDATE_SHA" && -z "$merged_at" ]] || {
  echo "READONLY CHECK BLOCKED: PR #153 is not open/draft/unmerged exact head" >&2
  exit 78
}
unset TOKEN GH_TOKEN GHCR_TOKEN

kctl() {
  if command -v k3s >/dev/null 2>&1; then
    sudo -n k3s kubectl "$@"
  elif sudo -n kubectl version --client >/dev/null 2>&1; then
    sudo -n kubectl "$@"
  else
    kubectl "$@"
  fi
}

dev_hosts="$(kctl -n "$DEV_NAMESPACE" get ingress "$RELEASE" -o jsonpath='{range .spec.rules[*]}{.host}{"\n"}{end}' 2>/dev/null || true)"
live_hosts="$(kctl -n "$LIVE_NAMESPACE" get ingress "$RELEASE" -o jsonpath='{range .spec.rules[*]}{.host}{"\n"}{end}' 2>/dev/null || true)"
[[ "$dev_hosts" == "$DEV_HOST" ]] || { echo "READONLY CHECK BLOCKED: isolated DEV ingress is not DEV-only" >&2; exit 78; }
[[ "$live_hosts" == "$LIVE_HOST" ]] || { echo "READONLY CHECK BLOCKED: LIVE ingress is not LIVE-only" >&2; exit 78; }

dev_api_pod="$(kctl -n "$DEV_NAMESPACE" get pods -l 'app.kubernetes.io/instance=rum,app.kubernetes.io/component=api' --field-selector=status.phase=Running -o jsonpath='{.items[0].metadata.name}')"
live_api_pod="$(kctl -n "$LIVE_NAMESPACE" get pods -l 'app.kubernetes.io/instance=rum,app.kubernetes.io/component=api' --field-selector=status.phase=Running -o jsonpath='{.items[0].metadata.name}')"
[[ -n "$dev_api_pod" && -n "$live_api_pod" ]] || { echo "READONLY CHECK BLOCKED: API pod missing" >&2; exit 78; }

encode() { printf '%s' "$1" | python3 -c 'import base64,sys; print(base64.b64encode(sys.stdin.buffer.read()).decode())'; }
rater_b64="$(encode "$RATER_EMAIL")"
mate_b64="$(encode "$MATE_EMAIL")"
linked_b64="$(encode "$LINKED_NAME")"
gamer_b64="$(encode "$GAMERTAG")"

PHP_READONLY='require "/var/www/html/vendor/autoload.php"; $app=require "/var/www/html/bootstrap/app.php"; $app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap(); $r=base64_decode($argv[1]); $m=base64_decode($argv[2]); $l=base64_decode($argv[3]); $g=base64_decode($argv[4]); $linkedId=App\Models\Entity::where("canonical_name",$l)->value("id"); $gamerId=App\Models\Entity::where("canonical_name",$g)->value("id"); echo App\Models\User::where("email",$r)->count()." ".App\Models\User::where("email",$m)->count()." ".($linkedId?1:0)." ".($linkedId?App\Models\RatingEvent::where("target_entity_id",$linkedId)->count():0)." ".($gamerId?1:0)." ".($gamerId?App\Models\EntityClaim::where("entity_id",$gamerId)->count():0);'

probe() {
  local namespace="$1" pod="$2"
  kctl -n "$namespace" exec "$pod" -c php-fpm -- php -r "$PHP_READONLY" "$rater_b64" "$mate_b64" "$linked_b64" "$gamer_b64" 2>/dev/null | tail -n 1
}

dev_probe="$(probe "$DEV_NAMESPACE" "$dev_api_pod")"
live_probe="$(probe "$LIVE_NAMESPACE" "$live_api_pod")"
read -r dev_rater dev_mate dev_linked dev_ratings dev_gamer dev_claims <<<"$dev_probe"
read -r live_rater live_mate live_linked live_ratings live_gamer live_claims <<<"$live_probe"

[[ "$dev_rater" == "1" && "$dev_mate" == "1" && "$dev_linked" == "1" && "$dev_ratings" =~ ^[1-9][0-9]*$ && "$dev_gamer" == "1" && "$dev_claims" =~ ^[1-9][0-9]*$ ]] || {
  echo "READONLY CHECK FAILED: successful verifier fixtures are not present in isolated DEV as expected: $dev_probe" >&2
  exit 1
}
[[ "$live_rater" == "0" && "$live_mate" == "0" && "$live_linked" == "0" && "$live_ratings" == "0" && "$live_gamer" == "0" && "$live_claims" == "0" ]] || {
  echo "SEV-0: isolated DEV verifier fixtures appeared in LIVE: $live_probe" >&2
  exit 70
}

printf 'RUM_CANDIDATE_SHA=%s\n' "$CANDIDATE_SHA"
printf 'DEV_FIXTURE_COUNTS=rater:%s mate:%s linked:%s ratings:%s gamer:%s claims:%s\n' "$dev_rater" "$dev_mate" "$dev_linked" "$dev_ratings" "$dev_gamer" "$dev_claims"
printf 'LIVE_FIXTURE_COUNTS=rater:%s mate:%s linked:%s ratings:%s gamer:%s claims:%s\n' "$live_rater" "$live_mate" "$live_linked" "$live_ratings" "$live_gamer" "$live_claims"
printf 'DEV_FIXTURES_PRESENT=YES\n'
printf 'LIVE_FIXTURES_ABSENT=YES\n'
printf 'LIVE_READ_ONLY=YES\n'
printf 'LIVE_MUTATION=NO\n'
printf 'RATE_ANYTHING_AFFECTED=NO\n'
