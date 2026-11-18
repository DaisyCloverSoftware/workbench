#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only CJ current public node and founder-authorised history parity after physical purge.
[[ $# -eq 0 ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == 'ghcr.io/daisycloversoftware/rum-api@sha256:a23e36323b55f3bce46d5b229146e2e7a9d70bdff9e16ea366380940e0629430' ]] || exit 78
php='
require "vendor/autoload.php";$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$id="01m07d2f9fj05f4ypdbwp087qj";$keep="01m0mvrxv8c3s12q0w7gwmzt03";$founderId="01kzgn6a7kswt3jqjb1sp8d4x0";
$user=App\Models\User::query()->findOrFail($founderId);
$p=app(App\Services\ProfileService::class)->profile($user,$user,null,null);
$rows=collect($p["data"]["ratingsGiven"]??[])->filter(fn($v)=>(string)($v["target"]["id"]??"")===$id)->values();
if($rows->count()!==1||($rows[0]["id"]??null)!==$keep||(int)($rows[0]["value"]??0)!==5||
 hash("sha256",(string)($rows[0]["reason"]??""))!=="aa2bcc4c7b1bf6348c46944f1bebd54298d661e4e2db030c08170a99b7d98e2e")
 throw new RuntimeException("Owner history still includes unwanted CJ ratings or changed review");
echo "RATCJ_FOUNDER_HISTORY_ONE_ORIGINAL_PASS=".json_encode(["id"=>$keep,"displayCount"=>1,"reviewIntact"=>true],JSON_UNESCAPED_SLASHES)."\n";
'
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$php"
response="$(curl -fsS --retry 3 --max-time 25 -H 'Cache-Control: no-cache' 'https://dev-rum.daisycloversoftware.uk/api/v1/public/entities/rat-catalogue/nodes/01m07d2f9fj05f4ypdbwp087qj')"
python3 - "$response" <<'PY'
import json,sys
n=json.loads(sys.argv[1])["data"]
assert n["active"]["id"]=="01m07d2f9fj05f4ypdbwp087qj"
assert n["active"]["reputation"] is None, "stale public 3.3 persists"
assert n["topLinked"]==[] and n["categories"]==[]
print("RATCJ_CURRENT_RAT_PUBLIC_REPUTATION_NOT_RATED_PASS=true")
PY
echo RATCJ_ONE_RATING_PUBLIC_AND_OWNER_HISTORY_AUDIT_PASS=true
