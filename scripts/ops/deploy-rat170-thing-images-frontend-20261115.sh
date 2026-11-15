#!/usr/bin/env bash
set -Eeuo pipefail

# PR170 Thing-image frontend only. Run after the isolated canonical API has
# been deployed with the exact same source and named media-check migration.
[[ $# -eq 4 ]] || { echo 'usage: source-sha rat@sha256 canonical-api@sha256 expected-previous-rat@sha256' >&2; exit 64; }
source_sha="$1"; candidate="$2"; api="$3"; previous_expected="$4"
[[ "$source_sha" =~ ^[0-9a-f]{40}$ ]] || exit 64
[[ "$candidate" =~ ^ghcr[.]io/daisycloversoftware/rum-rate-anything@sha256:[0-9a-f]{64}$ ]] || exit 64
[[ "$previous_expected" =~ ^ghcr[.]io/daisycloversoftware/rum-rate-anything@sha256:[0-9a-f]{64}$ ]] || exit 64
[[ "$api" =~ ^ghcr[.]io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
img(){ kctl -n "$1" get deployment "$2" -o "jsonpath={.spec.template.spec.containers[?(@.name==\"$3\")].image}"; }
host(){ kctl -n "$1" get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u; }
[[ "$(host rum-dev-isolated)" == 'dev-rum.daisycloversoftware.uk' ]] || exit 78
[[ "$(host rum-rate-anything-preview)" == 'dev-rum-ra.daisycloversoftware.uk' ]] || exit 78
[[ "$(host rum-dev)" == 'rateurmate.online' ]] || exit 78
[[ "$(img rum-dev-isolated rum-api php-fpm)" == "$api" ]] || { echo 'Required isolated API is not deployed'; exit 78; }
old="$(img rum-rate-anything-preview rum-rate-anything rate-anything)"
[[ "$old" == "$previous_expected" || "$old" == "$candidate" ]] || exit 78

# Verify the named additive migration actually ran in canonical isolated DEV.
php='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";
$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
if($db::table("migrations")->where("migration","2026_11_15_000001_allow_thing_image_media_purpose")->count()!==1)exit(78);
$rows=$db::select("SELECT conname, pg_get_constraintdef(oid) AS definition FROM pg_constraint WHERE conrelid = '\''media_assets'\''::regclass AND conname IN ('\''media_assets_purpose_allowed'\'','\''media_assets_attachment_type_allowed'\'')");
if(count($rows)!==2)exit(78);
foreach($rows as $row){
  $want=$row->conname==="media_assets_purpose_allowed"?"thing_image":"entity_profile_revision";
  if(!str_contains($row->definition,$want))exit(78);
}
echo "MIGRATION_READY\n";
'
status="$(kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$php")"
[[ "$status" == *MIGRATION_READY* ]] || exit 78
health="$(curl -fsS --retry 4 --retry-delay 2 --max-time 20 https://dev-rum.daisycloversoftware.uk/api/v1/health)"
python3 - "$health" "$source_sha" <<'PY'
import sys,json
data=json.loads(sys.argv[1])
assert data.get('status')=='ok' and data.get('service')=='rum-api'
assert sys.argv[2] in str(data.get('version',''))
PY

# Ensure only RAT frontend spec changes. Immutable before/after of all other
# deployment specs, configmaps, ingress and persistent data resources.
protected_snapshot(){
  for ns in rum-rate-anything-preview rum-dev-isolated rum-dev; do
    kctl -n "$ns" get deployment,statefulset,cronjob,configmap,persistentvolumeclaim,ingress -o json | python3 -c '
import json,sys
items=json.load(sys.stdin)["items"]
rows=[]
for item in items:
  m=item["metadata"]
  if m["namespace"]=="rum-rate-anything-preview" and item["kind"]=="Deployment" and m["name"]=="rum-rate-anything": continue
  rows.append({"namespace":m["namespace"],"kind":item["kind"],"name":m["name"],"spec":item.get("spec"),"data":item.get("data")})
print(json.dumps(sorted(rows,key=lambda x:(x["namespace"],x["kind"],x["name"])),sort_keys=True,separators=(",",":")))' 
  done | sha256sum | cut -d' ' -f1
}
before="$(protected_snapshot)"
api_before="$(img rum-dev-isolated rum-api php-fpm)"
live_api="$(img rum-dev rum-api php-fpm)"
live_web="$(img rum-dev rum-web web)"
live_worker="$(img rum-dev rum-worker worker)"
[[ "$live_api" == 'ghcr.io/daisycloversoftware/rum-api:sha-54d31b8' ]] || exit 78
[[ "$live_web" == 'ghcr.io/daisycloversoftware/rum-web:sha-8106675' ]] || exit 78
[[ "$live_worker" == 'ghcr.io/daisycloversoftware/rum-api:sha-54d31b8' ]] || exit 78
changed=false
rollback(){
  rc=$?; trap - ERR
  if [[ "$changed" == true ]]; then
    echo 'RAT Thing-image frontend verification failed; restoring prior isolated frontend' >&2
    kctl -n rum-rate-anything-preview set image deployment/rum-rate-anything "rate-anything=$old" >/dev/null || true
    kctl -n rum-rate-anything-preview rollout status deployment/rum-rate-anything --timeout=240s || true
  fi
  exit "$rc"
}
trap rollback ERR
if [[ "$old" != "$candidate" ]]; then
  changed=true
  kctl -n rum-rate-anything-preview set image deployment/rum-rate-anything "rate-anything=$candidate"
fi
kctl -n rum-rate-anything-preview rollout status deployment/rum-rate-anything --timeout=300s
[[ "$(img rum-rate-anything-preview rum-rate-anything rate-anything)" == "$candidate" ]]
served="$(curl -fsS --retry 6 --retry-delay 2 --max-time 20 -H 'Cache-Control: no-cache' https://dev-rum-ra.daisycloversoftware.uk/VERSION | tr -d '\r\n')"
[[ "$served" == "rat-candidate+$source_sha" ]] || { echo "RAT served source mismatch"; false; }
[[ "$(img rum-dev-isolated rum-api php-fpm)" == "$api_before" ]]
[[ "$(img rum-dev rum-api php-fpm)" == "$live_api" ]]
[[ "$(img rum-dev rum-web web)" == "$live_web" ]]
[[ "$(img rum-dev rum-worker worker)" == "$live_worker" ]]
[[ "$(host rum-dev)" == 'rateurmate.online' ]]
after="$(protected_snapshot)"
[[ "$before" == "$after" ]] || { echo 'A protected deployment/config spec changed during RAT promotion'; false; }
trap - ERR
echo 'RAT170_THING_IMAGES_FRONTEND_ISOLATED_OK'
echo "source_sha=$source_sha"
echo "rat_image=$candidate"
echo "canonical_api=$api"
echo "previous_rat=$old"
echo "served_version=$served"
echo "protected_before=$before"
echo "protected_after=$after"
echo 'no_worker_web_migrations_seed_reset_or_live_mutation=true'
