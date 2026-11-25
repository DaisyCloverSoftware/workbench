#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 0 ]] || exit 64
sha=2359178af19fdd9b5b45d103c892cdc36c2ad086
api=ghcr.io/daisycloversoftware/rum-api@sha256:5e8486292d3f1b0ab0b834cd98028892ccf73a600548234d737ea7402e60d336
rat=ghcr.io/daisycloversoftware/rum-rate-anything@sha256:ba60b264d34730fb32a8f20a0ebbd26030dbff5917ad2de1407471f904cb091f
ops=$(cd "$(dirname "$0")" && pwd);t=$(mktemp -d);trap 'rm -rf "$t"' EXIT
k(){ if command -v k3s >/dev/null;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ $(k -n rum-dev-isolated get deploy rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}') == "$api" ]]
[[ $(k -n rum-rate-anything-preview get deploy rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}') == "$rat" ]]
[[ $(curl -fsS --retry 2 --max-time 25 https://dev-rum-ra.daisycloversoftware.uk/VERSION) == "rat-candidate+$sha" ]]
curl -fsS --retry 2 --max-time 25 https://dev-rum.daisycloversoftware.uk/api/v1/health | python3 -c 'import json,sys;v=json.load(sys.stdin);assert v["status"]=="ok" and v["version"]=="rat-candidate+"+sys.argv[1];print("FINAL_API_HEALTH",v["version"])' "$sha"
# Verify the exact implementation being read in the canonical DEV container, without altering it.
git -C "$t" init -q;git -C "$t" remote add origin https://github.com/DaisyCloverSoftware/rum.git;git -C "$t" fetch -q --depth=1 origin "$sha"
for f in app/Services/RatSharedConnectionDiscovery.php app/Services/RatExplorerService.php;do
 expected=$(git -C "$t" show "$sha:apps/api/$f" | sha256sum | cut -d' ' -f1)
 actual=$(k -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- sha256sum "$f" | cut -d' ' -f1)
 [[ "$actual" == "$expected" ]];echo "RUNNING_SOURCE_MATCH $f $actual"
done
k -n rum-dev-isolated exec -i deployment/rum-api -c php-fpm -- php <<'PHP'
<?php
require 'vendor/autoload.php';$app=require 'bootstrap/app.php';$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
use Illuminate\Support\Facades\DB;use App\Services\RatSharedConnectionDiscovery;
DB::transaction(function(){
 DB::statement('SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY');
 $person='01m0mw9hkbynssr7kbc0gk534j';$main='01m7x6ez57zpsgbv486y9t0y9h';$second='01m81mbmqvwr3d1nvfp9gkp3xf';
 $p=DB::table('entities')->where('id',$person)->first();if(!$p||$p->status!=='unverified'||$p->visibility!=='public'||$p->publication_state!=='published')throw new RuntimeException('Persona state drift');
 $expected=['01m7x7kn332fqjxyj9b78zrmh0'=>[$main,$person],'01m81n8fsgxf1p52bw62e9rzyp'=>[$person,$second]];
 $rows=DB::table('entity_relationships')->whereIn('source_entity_id',[$person,$main,$second])->orWhereIn('target_entity_id',[$person,$main,$second])->orderBy('id')->get();if($rows->count()!==2)throw new RuntimeException('Unexpected relationship count');
 foreach($rows as $r){if(!isset($expected[$r->id])||[$r->source_entity_id,$r->target_entity_id]!==$expected[$r->id]||$r->visibility!=='public'||$r->verification_state!=='unverified'||$r->provenance!=='user_submitted'||$r->valid_to!==null)throw new RuntimeException('Relationship drift');echo 'CANONICAL_UNCHANGED_PATH_EDGE '.json_encode(['id'=>$r->id,'source'=>$r->source_entity_id,'target'=>$r->target_entity_id,'verification'=>$r->verification_state,'provenance'=>$r->provenance]).PHP_EOL;}
 echo 'PUBLIC_UNVERIFIED_PERSON_PRESERVED '.$person.PHP_EOL;
 echo 'RUNNING_DISCOVERY_LIMITS '.json_encode((new ReflectionClass(RatSharedConnectionDiscovery::class))->getConstants()).PHP_EOL;
});
PHP
bash "$ops/check-rat181-protected-baseline.sh" >"$t/audit"
for expected in \
 'entities rows=3 sha256=c1090b07abd3511fb5b4c40862025ca1b69e12c0742739a303d52b7f02563e7e' \
 'entity_identifiers rows=1 sha256=db111e02ca58748931dd5252ff6fc4ab03332bcf37678297ff241c3ca821f791' \
 'rating_events rows=2 sha256=51fd8c48c16b3ec3ad33a5d7a70031c9f0122a51bad8bbb8cd4b8d45c25336e4' \
 'relationships rows=2 sha256=688a57ffb9558a55e5098d3a04006abdb733446d15bd6828077263538b088c7c';do grep -Fxq "$expected" "$t/audit";done
cat "$t/audit"
echo "RAT182_FINAL_CANONICAL_AND_RUNNING_SOURCE_AUDIT_PASS sha=$sha canonical_writes=0"
