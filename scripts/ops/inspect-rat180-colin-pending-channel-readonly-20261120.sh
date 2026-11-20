#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only authoritative isolated DEV database checkpoint after one-shot browser create.
[[ $# -eq 0 ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
api='ghcr.io/daisycloversoftware/rum-api@sha256:7411cc02eb6568e6c6d228b551683294bfac49a6f89c320078af5550adce5d56'
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api" ]] || exit 78
code='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
$person="01m0mw9hkbynssr7kbc0gk534j";$channel="01m7x6ez57zpsgbv486y9t0y9h";
$get=function(string $id) use ($db){
 $e=App\Models\Entity::query()->with("type","classificationAssignments.classification","identifiers")->findOrFail($id);
 $assigned=$e->classificationAssignments->filter(fn($a)=>$a->valid_to===null)->map(fn($a)=>[
 "key"=>$a->classification?->key,"name"=>$a->classification?->canonical_name,
 "primary"=>(bool)$a->is_primary,"verificationState"=>$a->verification_state])->values()->all();
 return ["id"=>(string)$e->id,"name"=>(string)$e->canonical_name,
 "type"=>$e->type?->key,"status"=>$e->status,"visibility"=>$e->visibility,
 "publication"=>$e->publication_state,"rateability"=>$e->rateability,
 "ratingEvents"=>$db::table("rating_events")->where("target_entity_id",$id)->count(),
 "classifications"=>$assigned,
 "identifiers"=>$e->identifiers->filter(fn($i)=>$i->valid_to===null)->map(fn($i)=>[
 "namespace"=>$i->namespace,"value"=>$i->value,"verificationState"=>$i->verification_state])->values()->all()];
};
$journeys=$db::table("rating_journeys")->where("root_entity_id",$channel)->get(["id","state","origin_surface","user_id"])
 ->map(fn($j)=>["id"=>$j->id,"state"=>$j->state,"origin"=>$j->origin_surface,"ownerAccountId"=>$j->user_id])->all();
$edges=$db::table("entity_relationships as r")->join("relationship_types as t","t.id","=","r.relationship_type_id")
 ->whereNull("r.valid_to")->where(fn($q)=>$q->where(fn($z)=>$z->where("r.source_entity_id",$person)->where("r.target_entity_id",$channel))
 ->orWhere(fn($z)=>$z->where("r.source_entity_id",$channel)->where("r.target_entity_id",$person)))
 ->get(["r.id","r.source_entity_id","r.target_entity_id","r.verification_state","r.visibility","r.provenance","t.key"]);
$out=["person"=>$get($person),"channel"=>$get($channel),"journeys"=>$journeys,
"relationships"=>$edges,"sameNameRecordCount"=>$db::table("entities")->where("canonical_name","Colin Furze")->count()];
echo "RAT180_COLIN_CANONICAL_READONLY=".json_encode($out,JSON_UNESCAPED_SLASHES)."\\n";
';
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$code"
echo RAT180_CANONICAL_READONLY_COMPLETE=true
