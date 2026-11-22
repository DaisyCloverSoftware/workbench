#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# -eq 0 ]] || exit 64
k(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
api=$(k -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')
rat=$(k -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')
printf 'API=%s\nRAT=%s\n' "$api" "$rat"
[[ $api == 'ghcr.io/daisycloversoftware/rum-api@sha256:7411cc02eb6568e6c6d228b551683294bfac49a6f89c320078af5550adce5d56' && $rat == 'ghcr.io/daisycloversoftware/rum-rate-anything@sha256:9a33ea315f52d5e14a5ece0e57895aae8d4eb4d7d7e9955110d49fc9676c64d9' ]] || exit 78
k -n rum-dev-isolated exec -i deployment/rum-api -c php-fpm -- php <<'PHP'
<?php
require 'vendor/autoload.php';
$app=require 'bootstrap/app.php';$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
use App\Models\Entity;use App\Models\EntityRelationship;use Illuminate\Support\Facades\DB;
$p='01m0mw9hkbynssr7kbc0gk534j';$main='01m7x6ez57zpsgbv486y9t0y9h';$external='UCdmcAwbdiXQWs9d4Qd4KzDw';
$records=Entity::query()->with('type','identifiers','classificationAssignments.classification')->where(fn($q)=>$q->whereRaw('LOWER(canonical_name) = ?',['2 much colinfurze'])->orWhereHas('identifiers',fn($i)=>$i->where('namespace','youtube_channel_id')->where('value',$external)))->get();
foreach($records as $e){$ids=$e->identifiers->where('namespace','youtube_channel_id')->whereNull('valid_to')->pluck('value');echo 'CANDIDATE='.json_encode(['id'=>$e->id,'name'=>$e->canonical_name,'kind'=>$e->type?->key,'publication'=>$e->publication_state,'visibility'=>$e->visibility,'youtubeChannelIds'=>$ids->values(),'classification'=>$e->classificationAssignments->filter(fn($a)=>$a->valid_to===null&&$a->is_primary)->map(fn($a)=>$a->classification?->canonical_name)->values()]).PHP_EOL;
$edges=EntityRelationship::query()->where(fn($q)=>$q->where('source_entity_id',$p)->where('target_entity_id',$e->id))->orWhere(fn($q)=>$q->where('source_entity_id',$e->id)->where('target_entity_id',$p))->get(['id','source_entity_id','target_entity_id','verification_state','visibility','valid_to','source']);echo 'ALL_PAIR_EDGES='.$edges->toJson().PHP_EOL;
foreach([$p,$main,$e->id] as $id){$ids=DB::table('rating_events')->where('target_entity_id',$id)->orderBy('id')->pluck('id');echo 'RATINGS='.json_encode(['id'=>$id,'count'=>$ids->count(),'event_id_hash'=>hash('sha256',$ids->toJson())]).PHP_EOL;}}
if($records->count()!==1)exit(78);
PHP
