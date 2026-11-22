#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# -eq 0 ]] || exit 64
k(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
for ns in rum-dev rum-dev-isolated rum-rate-anything-preview; do
 case "$ns" in rum-dev) h=rateurmate.online;; rum-dev-isolated) h=dev-rum.daisycloversoftware.uk;; *) h=dev-rum-ra.daisycloversoftware.uk;; esac
 hosts=$(k -n "$ns" get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}'|sort -u); [[ "$hosts" == "$h" ]] || { echo "Unexpected routing $ns"; exit 78; }; echo "ROUTING $ns $h";
 k -n "$ns" get pods -o json | python3 -c 'import json,sys; print([(p["metadata"]["name"],c["name"],c.get("imageID"),c.get("ready")) for p in json.load(sys.stdin)["items"] for c in (p.get("status",{}).get("containerStatuses") or []) if c["name"] in ("php-fpm","rate-anything")])'
done
k -n rum-dev-isolated exec -i deployment/rum-api -c php-fpm -- php <<'PHP'
<?php
require 'vendor/autoload.php'; $app=require 'bootstrap/app.php';$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
use Illuminate\Support\Facades\DB; use App\Models\Entity; use App\Services\RumEntityAudiencePolicy;
$ids=['01m0mw9hkbynssr7kbc0gk534j','01m7x6ez57zpsgbv486y9t0y9h','01m81mbmqvwr3d1nvfp9gkp3xf'];
DB::transaction(function()use($ids){DB::statement('SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY'); foreach(['entities'=>'id','entity_identifiers'=>'entity_id','rating_events'=>'target_entity_id'] as $table=>$field){$rows=DB::table($table)->whereIn($field,$ids)->orderBy('id')->get();echo $table.' rows='.$rows->count().' sha256='.hash('sha256',$rows->toJson()).PHP_EOL;}
$rows=DB::table('entity_relationships')->whereIn('source_entity_id',$ids)->orWhereIn('target_entity_id',$ids)->orderBy('id')->get();echo 'relationships rows='.$rows->count().' sha256='.hash('sha256',$rows->toJson()).PHP_EOL;foreach($ids as $i=>$id){$e=Entity::query()->findOrFail($id);$allowed=app(RumEntityAudiencePolicy::class)->allows($e);if($allowed!==($i===0))throw new RuntimeException('RUM audience drift');echo 'AUDIENCE '.$id.' rum='.json_encode($allowed).PHP_EOL;}});
PHP
