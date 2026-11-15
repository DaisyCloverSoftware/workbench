#!/usr/bin/env bash
set -Eeuo pipefail

# PR170: advance only canonical isolated DEV API and apply one additive media CHECK
# migration. Never mutate LIVE, RAT frontend, worker, web, fixtures or other DBs.
[[ $# -eq 4 ]] || { echo 'usage: source-sha api@sha256 expected-dev-api@sha256 expected-rat-frontend@sha256' >&2; exit 64; }
source_sha="$1"; candidate="$2"; expected_api="$3"; expected_rat="$4"
[[ "$source_sha" =~ ^[0-9a-f]{40}$ ]] || exit 64
[[ "$candidate" =~ ^ghcr[.]io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ ]] || exit 64
[[ "$expected_api" =~ ^ghcr[.]io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ ]] || exit 64
[[ "$expected_rat" =~ ^ghcr[.]io/daisycloversoftware/rum-rate-anything@sha256:[0-9a-f]{64}$ ]] || exit 64
migration='2026_11_15_000001_allow_thing_image_media_purpose'
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
img(){ kctl -n "$1" get deployment "$2" -o "jsonpath={.spec.template.spec.containers[?(@.name==\"$3\")].image}"; }
host(){ kctl -n "$1" get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u; }
[[ "$(host rum-dev-isolated)" == 'dev-rum.daisycloversoftware.uk' ]] || exit 78
[[ "$(host rum-rate-anything-preview)" == 'dev-rum-ra.daisycloversoftware.uk' ]] || exit 78
[[ "$(host rum-dev)" == 'rateurmate.online' ]] || exit 78
for ns in rum-dev-isolated rum-rate-anything-preview rum-dev; do kctl get namespace "$ns" >/dev/null; done
old_api="$(img rum-dev-isolated rum-api php-fpm)"
old_rat="$(img rum-rate-anything-preview rum-rate-anything rate-anything)"
dev_web="$(img rum-dev-isolated rum-web web)"
dev_worker="$(img rum-dev-isolated rum-worker worker)"
preview_api="$(img rum-rate-anything-preview rum-api php-fpm)"
live_api="$(img rum-dev rum-api php-fpm)"
live_web="$(img rum-dev rum-web web)"
live_worker="$(img rum-dev rum-worker worker)"
[[ "$old_api" == "$expected_api" || "$old_api" == "$candidate" ]] || { echo 'Canonical DEV API differs from approved baseline'; exit 78; }
[[ "$old_rat" == "$expected_rat" ]] || { echo 'RAT frontend differs from approved baseline'; exit 78; }
[[ "$live_api" == 'ghcr.io/daisycloversoftware/rum-api:sha-54d31b8' ]] || exit 78
[[ "$live_web" == 'ghcr.io/daisycloversoftware/rum-web:sha-8106675' ]] || exit 78
[[ "$live_worker" == 'ghcr.io/daisycloversoftware/rum-api:sha-54d31b8' ]] || exit 78

# Observe old data without writing rows. Verify table exists and exact constraint
# names before attempting any deployment. Do not print credentials or private data.
php_before='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";
$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
$names=$db::select("SELECT conname FROM pg_constraint WHERE conrelid = '\''media_assets'\''::regclass AND conname IN ('\''media_assets_purpose_allowed'\'','\''media_assets_attachment_type_allowed'\'')");
if (count($names)!==2) exit(78);
echo $db::table("media_assets")->count();
'
before_count="$(kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$php_before" | tr -d '\r\n')"
[[ "$before_count" =~ ^[0-9]+$ ]] || exit 78

changed=false
rollback_api(){
  rc=$?
  trap - ERR
  if [[ "$changed" == true ]]; then
    echo 'Isolated API operation failed; restoring prior isolated API image; any applied additive migration remains intact' >&2
    kctl -n rum-dev-isolated set image deployment/rum-api "php-fpm=$old_api" >/dev/null || true
    kctl -n rum-dev-isolated rollout status deployment/rum-api --timeout=240s || true
  fi
  exit "$rc"
}
trap rollback_api ERR
if [[ "$old_api" != "$candidate" ]]; then
  changed=true
  kctl -n rum-dev-isolated set image deployment/rum-api "php-fpm=$candidate"
fi
kctl -n rum-dev-isolated rollout status deployment/rum-api --timeout=300s
[[ "$(img rum-dev-isolated rum-api php-fpm)" == "$candidate" ]]

# Version check before named migration ensures code and schema can be attributed.
health="$(curl -fsS --retry 5 --retry-delay 2 --max-time 20 https://dev-rum.daisycloversoftware.uk/api/v1/health)"
python3 - "$source_sha" "$health" <<'PY'
import sys,json
sha,obj=sys.argv[1],json.loads(sys.argv[2])
assert obj.get('status')=='ok' and obj.get('service')=='rum-api',obj
assert sha in str(obj.get('version','')),obj
PY

# Only the new named additive migration; never run a generic migration batch,
# migrate:fresh, any seed, or a command in the LIVE namespace.
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php artisan migrate --force --no-interaction --path="database/migrations/$migration.php"
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php artisan migrate:status --path="database/migrations/$migration.php" --no-ansi

php_after='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";
$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
$rows=$db::select("SELECT conname, pg_get_constraintdef(oid) AS definition FROM pg_constraint WHERE conrelid = '\''media_assets'\''::regclass AND conname IN ('\''media_assets_purpose_allowed'\'','\''media_assets_attachment_type_allowed'\'')");
$map=[]; foreach($rows as $row) $map[$row->conname]=(string)$row->definition;
$required=[
"media_assets_purpose_allowed"=>["avatar","rate_my","thing_image"],
"media_assets_attachment_type_allowed"=>["profile_avatar","rate_my","entity_profile_revision"]];
if(count($map)!==2) exit(78);
foreach($required as $name=>$allowed){
  foreach($allowed as $value){if(!str_contains($map[$name],$value))exit(78);}
  if(!str_contains($map[$name],"CHECK"))exit(78);
}
if($db::table("migrations")->where("migration","2026_11_15_000001_allow_thing_image_media_purpose")->count()!==1)exit(78);
echo "RAT170_MEDIA_CHECKS_VERIFIED=".json_encode(["constraints"=>$map,"mediaRowCount"=>$db::table("media_assets")->count()],JSON_THROW_ON_ERROR)."\n";
'
after="$(kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$php_after")"
grep -q 'RAT170_MEDIA_CHECKS_VERIFIED=' <<<"$after"
after_count="$(python3 - "$after" <<'PY'
import json,sys
print(json.loads(sys.argv[1].split('RAT170_MEDIA_CHECKS_VERIFIED=',1)[1])['mediaRowCount'])
PY
)"
[[ "$before_count" == "$after_count" ]] || { echo 'Media record count changed unexpectedly'; false; }
[[ "$(img rum-dev-isolated rum-api php-fpm)" == "$candidate" ]]
[[ "$(img rum-dev-isolated rum-web web)" == "$dev_web" ]]
[[ "$(img rum-dev-isolated rum-worker worker)" == "$dev_worker" ]]
[[ "$(img rum-rate-anything-preview rum-api php-fpm)" == "$preview_api" ]]
[[ "$(img rum-rate-anything-preview rum-rate-anything rate-anything)" == "$old_rat" ]]
[[ "$(img rum-dev rum-api php-fpm)" == "$live_api" ]]
[[ "$(img rum-dev rum-web web)" == "$live_web" ]]
[[ "$(img rum-dev rum-worker worker)" == "$live_worker" ]]
[[ "$(host rum-dev-isolated)" == 'dev-rum.daisycloversoftware.uk' ]]
[[ "$(host rum-dev)" == 'rateurmate.online' ]]
trap - ERR
echo 'RAT170_THING_IMAGES_API_MIGRATION_ISOLATED_OK'
echo "source_sha=$source_sha"
echo "api_image=$candidate"
echo "previous_api=$old_api"
echo "migration=$migration"
echo "$after"
echo 'unchanged_components=rum-web,rum-worker,RAT-preview-api,RAT-frontend,LIVE'
echo 'no_unrelated_migrations=true no_seed=true no_reset=true no_live_mutation=true'
