#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 0 ]] || exit 64
k(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ $(k -n rum-dev-isolated get ingress -o 'jsonpath={.items[*].spec.rules[*].host}') == dev-rum.daisycloversoftware.uk ]] || exit 78
[[ $(k -n rum-dev-isolated get deploy rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}') == ghcr.io/daisycloversoftware/rum-api@sha256:6ad2db06b44acca08d26ac4e062bc77d279513e45a62e862d4bb85e10a21d450 ]] || exit 78
k -n rum-dev-isolated exec -i deploy/rum-api -c php-fpm -- php /dev/stdin <<'PHP'
<?php
require 'vendor/autoload.php';$a=require 'bootstrap/app.php';$a->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
use App\Models\Entity;use App\Models\EntityRelationship;use App\Services\RatDiscoveryPolicy;use App\Services\RatSharedConnectionDiscovery;use Illuminate\Support\Facades\DB;
DB::beginTransaction();DB::statement('SET TRANSACTION READ ONLY');DB::statement("SET LOCAL statement_timeout = '5s'");
try {
$ids=['01m0mw9hkbynssr7kbc0gk534j','01m7x6ez57zpsgbv486y9t0y9h','01m81mbmqvwr3d1nvfp9gkp3xf'];$svc=app(RatSharedConnectionDiscovery::class);$policy=app(RatDiscoveryPolicy::class);$channel=new ReflectionMethod($svc,'isChannel');$keys=new ReflectionMethod($svc,'channelKeys');echo 'EXPECTED_CHANNEL_KEYS '.json_encode($keys->invoke($svc))."\n";
echo 'CLOCK '.json_encode(['app'=>now()->toIso8601String(),'db'=>DB::selectOne('SELECT current_timestamp AS now')->now])."\n";
foreach(Entity::query()->whereIn('id',$ids)->with(['type','classificationAssignments.classification'])->get() as $e){
 $row=$e->only(['id','canonical_name','status','visibility','publication_state','rateability','reviewability','sensitivity','deleted_at']);$row['type']=['key'=>$e->type?->key,'status'=>$e->type?->status];$row['ratAllows']=$policy->allows($e);$row['isChannel']=$channel->invoke($svc,$e);$row['classifications']=$e->classificationAssignments->take(12)->map(fn($x)=>['is_primary'=>$x->is_primary,'valid_from'=>$x->getRawOriginal('valid_from'),'valid_to'=>$x->getRawOriginal('valid_to'),'key'=>$x->classification?->key,'name'=>$x->classification?->canonical_name,'status'=>$x->classification?->status])->all();$row['derived']=array_map(fn($g)=>['via'=>$g['via']->id,'targets'=>array_map(fn($x)=>$x['entity']->id,$g['items'])],$svc->discover($e,[]));echo 'CANONICAL_GATE '.json_encode($row,JSON_THROW_ON_ERROR)."\n";
}
foreach(EntityRelationship::query()->whereIn('id',['01m7z18ppcak2qdskha06a68fb','01m81n8fsgxf1p52bw62e9rzyp'])->with('type')->get() as $e){$row=$e->only(['id','source_entity_id','target_entity_id','visibility','verification_state','provenance','source','valid_from','valid_to']);$row['type']=['key'=>$e->type?->key,'status'=>$e->type?->status];echo 'CANONICAL_EDGE '.json_encode($row,JSON_THROW_ON_ERROR)."\n";}
echo "RAT182_READONLY_GATE_DIAGNOSTIC_COMPLETE\n";
}finally{DB::rollBack();}
PHP
