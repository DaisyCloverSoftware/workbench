#!/usr/bin/env bash
set -euo pipefail
# Unauthenticated negative test against existing synthetic isolated DEV Thing.
# No login, no cookies, no CSRF token, no app changes, no owner entity access.
[[ $# -eq 0 ]] || exit 64
root='01m7d2emwsnd0nxw6new8khbje'
pending='01m7f8wrhb6jmjm7fg7xxk0320'
api='ghcr.io/daisycloversoftware/rum-api@sha256:13f71c669c003c6458a378dd5c6f24a53c77ad4955e9c804afd041cf8a3f879d'
rat='ghcr.io/daisycloversoftware/rum-rate-anything@sha256:2cd07b529eca5cfbfcf89a0cd0329c3f8f214079032b211ec80370b2c232e436'
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
image(){ kctl -n "$1" get deployment "$2" -o "jsonpath={.spec.template.spec.containers[?(@.name==\"$3\")].image}"; }
[[ "$(image rum-dev-isolated rum-api php-fpm)" == "$api" ]] || exit 78
[[ "$(image rum-rate-anything-preview rum-rate-anything rate-anything)" == "$rat" ]] || exit 78
snapshot(){
  kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r '
require "vendor/autoload.php";
$app=require "bootstrap/app.php";
$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
$root="01m7d2emwsnd0nxw6new8khbje";
$pending="01m7f8wrhb6jmjm7fg7xxk0320";
$e=$db::table("entities")->where("id",$root)->first();
$p=$db::table("entity_profile_revisions")->where("id",$pending)->where("entity_id",$root)->first();
if (!$e || $e->canonical_name!=="Carmine Nebula Archive rat170-community-a01" || !$p || $p->outcome!=="pending") exit(78);
$rows=$db::table("entity_profile_revisions")->where("entity_id",$root)->orderBy("id")
 ->get(["id","outcome","moderation_status","editor_user_id","proposed_value"]);
echo "syntheticRevisionSnapshotSha256=".hash("sha256",json_encode([
 "description"=>$e->description,
 "version"=>$e->version,
 "rows"=>$rows->all(),
],JSON_THROW_ON_ERROR))."\n";
'
}
before="$(snapshot)"
base='https://dev-rum-ra.daisycloversoftware.uk/api/v1/rate-anything/things'
post(){
  curl -sS --max-time 25 --retry 0 --request POST --header 'Accept: application/json' \
    --header 'Content-Type: application/json' --header 'Origin: https://dev-rum-ra.daisycloversoftware.uk' \
    --data "$2" --output /dev/null --write-out '%{http_code}' "$1"
}
withdraw_status="$(post "$base/$root/edits/$pending/withdraw" '{}')"
edit_status="$(post "$base/$root/edits" '{"field":"description","value":"Anonymous synthetic write must be rejected"}')"
echo "ANONYMOUS_WITHDRAWAL_HTTP=$withdraw_status"
echo "ANONYMOUS_EDIT_HTTP=$edit_status"
for status in "$withdraw_status" "$edit_status"; do
  [[ "$status" == 401 || "$status" == 403 || "$status" == 419 ]] || { echo 'RAT170_UNAUTHENTICATED_REQUEST_NOT_PROVEN_REJECTED' >&2; exit 2; }
done
after="$(snapshot)"
[[ "$before" == "$after" ]] || { echo 'RAT170_UNAUTHORISED_MUTATION_DETECTED' >&2; exit 2; }
echo 'RAT170_ISSUE8_ANONYMOUS_WRITES_REJECTED_SQL_IDENTICAL=true'
