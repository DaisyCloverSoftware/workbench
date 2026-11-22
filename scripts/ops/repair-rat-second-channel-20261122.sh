#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# -eq 0 ]] || exit 64
k(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
[[ $(k -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}') == 'ghcr.io/daisycloversoftware/rum-api@sha256:7411cc02eb6568e6c6d228b551683294bfac49a6f89c320078af5550adce5d56' ]] || exit 78
k -n rum-dev-isolated exec -i deployment/rum-api -c php-fpm -- php <<'PHP'
<?php
use App\Models\{Entity,EntityRelationship,RatingJourney,RatingJourneyEntry,User};use App\Services\RatCommunityConnectionService;use Illuminate\Support\Facades\DB;
require 'vendor/autoload.php';$app=require 'bootstrap/app.php';$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
function guard($b,$s){if(!$b)throw new RuntimeException($s);}
$result=DB::transaction(function(){
$p='01m0mw9hkbynssr7kbc0gk534j';$c='01m81mbmqvwr3d1nvfp9gkp3xf';$m='01m7x6ez57zpsgbv486y9t0y9h';$a='01m7x6eyapaft7xxmk4cv0vj6x';
$rows=Entity::query()->with('type','classificationAssignments.classification','identifiers')->whereKey([$p,$c])->orderBy('id')->lockForUpdate()->get()->keyBy('id');$person=$rows[$p];$channel=$rows[$c];
guard($person->canonical_name==='Colin Furze'&&$person->type->key==='person'&&$person->publication_state==='published'&&$person->visibility==='public','person changed');
guard($channel->canonical_name==='2 Much ColinFurze'&&$channel->type->key==='digital_resource'&&$channel->publication_state==='published'&&$channel->visibility==='public'&&$channel->classificationAssignments->contains(fn($i)=>$i->valid_to===null&&$i->is_primary&&$i->classification?->canonical_name==='YouTube Channel'),'channel changed');
guard(!$channel->identifiers->contains(fn($i)=>$i->valid_to===null&&$i->namespace==='youtube_channel_id'&&$i->value!=='UCdmcAwbdiXQWs9d4Qd4KzDw'),'conflicting external identifier');
$entry=RatingJourneyEntry::query()->where('entity_id',$c)->whereHas('journey',fn($q)=>$q->where('root_entity_id',$p))->exists();guard(!$entry,'existing cross-entity journey needs inspection');
$j=RatingJourney::query()->where('root_entity_id',$c)->get();echo 'JOURNEY_KINDS='.json_encode($j->map(fn($r)=>['state'=>$r->state,'standalone'=>isset($r->metadata['standaloneCreation'])])->values()).PHP_EOL;
$pair=fn()=>EntityRelationship::query()->where(fn($q)=>$q->where('source_entity_id',$p)->where('target_entity_id',$c))->orWhere(fn($q)=>$q->where('source_entity_id',$c)->where('target_entity_id',$p))->get();
$before=$pair();guard($before->isEmpty(),'relationship already present; no change');
$hashes=fn()=>collect([$p,$m,$c])->mapWithKeys(fn($id)=>[$id=>hash('sha256',DB::table('rating_events')->where('target_entity_id',$id)->orderBy('id')->pluck('id')->toJson())])->all();$h=$hashes();guard(array_values($h)===['4f53cda18c2baa0c0354bb5f9a3ecbe5ed12ab4d8e11ba873c2f11161202b945','c0d71c0cdeb103562b52798f03fe3ae433c42a7663d121137b30cce98805bb4d','ca91bf9718ce128f21a1d468137211cfd14e5ab2a8804fd364faea67510b9428'],'ratings changed since audit');
// Existing isolated RAT QA contributor, never the channel creator or a claimed owner by inference.
$actor=User::query()->findOrFail($a);$s=app(RatCommunityConnectionService::class);$out=$s->connect($actor,$person,$channel,'related_to','outgoing');$r=$out['relationship'];guard($out['created']&&$r->verification_state==='unverified'&&$r->visibility==='public'&&(string)$r->created_by_user_id===$a&&$r->source==='user_submitted','unexpected provenance');
$again=$s->connect($actor,$person,$channel,'related_to','outgoing');guard(!$again['created']&&$again['relationship']->is($r)&&$pair()->count()===1&&$hashes()===$h,'duplicate or rating invariant');return ['relationshipId'=>$r->id,'person'=>$p,'channel'=>$c,'actor'=>$a,'verification'=>$r->verification_state,'source'=>$r->source,'repeatWasNoOp'=>true,'ratingIdHashesUnchanged'=>true];});echo 'COMMITTED='.json_encode($result).PHP_EOL;
PHP
