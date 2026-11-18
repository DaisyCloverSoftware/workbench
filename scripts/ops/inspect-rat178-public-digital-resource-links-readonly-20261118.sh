#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only DEV discovery inventory; no claims, accounts or private relations.
[[ $# -eq 0 ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == 'ghcr.io/daisycloversoftware/rum-api@sha256:c68f9e4902b7ab71abab2cf0ea50244bf80dd93809385add84b764ff1fc34d0c' ]] || exit 78
php='
require "vendor/autoload.php";$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;$policy=app(App\Services\RatDiscoveryPolicy::class);
$items=App\Models\Entity::query()->with(["type","classificationAssignments.classification"])->whereHas("type",fn($q)=>$q->where("key","digital_resource"))
 ->where("visibility","public")->where("publication_state","published")->whereNotIn("status",["deleted","merged"])->get();
$out=[];
foreach($items as $e){
 if(!$policy->allows($e))continue;
 $primary=$e->classificationAssignments->first(fn($a)=>(bool)$a->is_primary&&$a->valid_to===null);
 $rels=$db::table("entity_relationships as r")->whereNull("r.valid_to")->where("r.visibility","public")
 ->where(fn($q)=>$q->where("r.source_entity_id",$e->id)->orWhere("r.target_entity_id",$e->id))
 ->get(["r.id","r.source_entity_id","r.target_entity_id"]);
 $public=[];
 foreach($rels as $r){
  $other=$r->source_entity_id===$e->id?$r->target_entity_id:$r->source_entity_id;
  $o=App\Models\Entity::query()->with("type")->find($other);
  if($o&&$policy->allows($o))$public[]=["otherId"=>(string)$o->id,"otherName"=>$o->canonical_name,"otherType"=>$o->type?->key,"relationId"=>$r->id];
 }
 $out[]=["id"=>(string)$e->id,"name"=>$e->canonical_name,"primaryKey"=>$primary?->classification?->key,"primaryLabel"=>$primary?->classification?->canonical_name,"eligiblePublicRelations"=>$public];
}
echo "RAT178_PUBLIC_DIGITAL_RESOURCE_LINKS=".json_encode($out,JSON_UNESCAPED_SLASHES)."\n";
';
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$php"
echo RAT178_READONLY_DIGITAL_RELATIONS_PASS=true
