#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 3 && $1 =~ ^[a-f0-9]{40}$ && $2 =~ ^(seed|hide|restore|inspect)$ && $3 =~ ^qa181[a-z0-9]{8}$ ]] || exit 64
sha=$1; action=$2; tag=$3
k(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ $(k -n rum-dev-isolated get ingress -o 'jsonpath={.items[*].spec.rules[*].host}') == 'dev-rum.daisycloversoftware.uk' ]] || exit 78
[[ $(curl -fsS --max-time 20 https://dev-rum-ra.daisycloversoftware.uk/VERSION) == "rat-candidate+$sha" ]] || exit 78
curl -fsS --max-time 20 https://dev-rum.daisycloversoftware.uk/api/v1/health | python3 -c 'import sys,json; d=json.load(sys.stdin);assert d["status"]=="ok" and sys.argv[1] in d["version"]' "$sha"
k -n rum-dev-isolated exec -i deployment/rum-api -c php-fpm -- php /dev/stdin "$action" "$tag" <<'PHP'
<?php
require 'vendor/autoload.php'; $app=require 'bootstrap/app.php';$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
use App\Models\{Entity,EntityType,EntityRelationship,RatingJourney,User};use Illuminate\Support\Facades\DB;
[$file,$action,$tag]=$argv; $actor=User::query()->where('username',$tag)->firstOrFail();$rootName="Test Observatory $tag";$existingName="Test Atlas $tag";$mark="Disposable RAT181 acceptance fixture $tag";
$result=DB::transaction(function()use($action,$tag,$actor,$rootName,$existingName,$mark){
$find=fn()=>Entity::query()->with('type')->where('created_by_user_id',$actor->id)->where('description',$mark)->orderBy('id');
if($action==='seed'){if($find()->exists()||Entity::query()->whereIn('canonical_name',[$rootName,$existingName])->exists())throw new RuntimeException('Fixture already exists');foreach([[$rootName,'organisation'],[$existingName,'product']] as [$name,$type]){Entity::query()->create(['entity_type_id'=>EntityType::query()->where('key',$type)->firstOrFail()->id,'canonical_name'=>$name,'normalised_name'=>mb_strtolower($name),'description'=>$mark,'created_by_user_id'=>$actor->id,'status'=>'active','visibility'=>'public','publication_state'=>'published','sensitivity'=>'normal','rateability'=>'open','reviewability'=>'closed','edit_policy'=>'community_governed','verification_state'=>'unverified']);}}
$source=$find()->where('canonical_name',$rootName)->whereHas('type',fn($q)=>$q->where('key','organisation'))->lockForUpdate()->sole();$existing=$find()->where('canonical_name',$existingName)->sole();
if($action==='hide'||$action==='restore'){$source->forceFill(['visibility'=>$action==='hide'?'private':'public'])->save();}
$rows=Entity::query()->where('created_by_user_id',$actor->id)->where('canonical_name','like','%'.$tag)->with('type')->get();$ids=$rows->pluck('id');$edges=EntityRelationship::query()->with('type')->whereNull('valid_to')->where(fn($q)=>$q->whereIn('source_entity_id',$ids)->orWhereIn('target_entity_id',$ids))->get();
return ['source'=>$source->id,'existing'=>$existing->id,'actor'=>$actor->id,'entities'=>$rows->map(fn($e)=>['id'=>$e->id,'name'=>$e->canonical_name,'state'=>$e->publication_state,'visible'=>$e->visibility,'rum'=>app(App\Services\RumEntityAudiencePolicy::class)->allows($e),'ratingIds'=>DB::table('rating_events')->where('target_entity_id',$e->id)->orderBy('id')->pluck('id')]),'edges'=>$edges->map(fn($r)=>['id'=>$r->id,'source'=>$r->source_entity_id,'target'=>$r->target_entity_id,'type'=>$r->type->key,'verification'=>$r->verification_state,'visibility'=>$r->visibility,'provenance'=>$r->provenance,'actor'=>$r->created_by_user_id]),'journeys'=>RatingJourney::query()->where('user_id',$actor->id)->whereIn('root_entity_id',$ids)->get()->map(fn($j)=>['id'=>$j->id,'target'=>$j->root_entity_id,'intent'=>$j->connectionIntent()])];});echo json_encode($result,JSON_THROW_ON_ERROR).PHP_EOL;
PHP
