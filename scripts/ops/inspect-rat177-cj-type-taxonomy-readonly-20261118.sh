#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only actual DEV CJ canonical type, classifications, fixed identity/ratings and YouTube Channel catalogue.
[[ $# -eq 0 ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
API='ghcr.io/daisycloversoftware/rum-api@sha256:76731a87574a11547938a398dc15ecf87ced56d63b6622cd50bab5836dfeee71'
RAT='ghcr.io/daisycloversoftware/rum-rate-anything@sha256:448807a70db93b91bb0fd714b854253c93157ad4d89d96255a385b94bf1258bc'
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$API" ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == "$RAT" ]] || exit 78
code='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
$id="01m07d2f9fj05f4ypdbwp087qj";
$e=App\Models\Entity::query()->with("type","classificationAssignments.classification")->findOrFail($id);
$policy=app(App\Services\RatCommunityContributionPolicy::class);
$discovery=app(App\Services\RatDiscoveryPolicy::class);
$assigned=$e->classificationAssignments->map(static fn($r)=>[
 "id"=>$r->id,"key"=>$r->classification?->key,"label"=>$r->classification?->canonical_name,
 "typeKey"=>$r->classification?->entityType?->key,"primary"=>(bool)$r->is_primary,
 "active"=>$r->valid_to===null,"source"=>$r->source,"verification"=>$r->verification_state])->values();
$related=App\Models\Classification::query()->with("entityType")->where(function($q){
 $q->whereRaw("lower(canonical_name) LIKE ?",["%youtube%"])
 ->orWhereRaw("lower(canonical_name) LIKE ?",["%channel%"]);
})->orderBy("canonical_name")->get()->map(static fn($c)=>[
 "id"=>$c->id,"key"=>$c->key,"label"=>$c->canonical_name,"kind"=>$c->entityType?->key,"status"=>$c->status]);
$types=App\Models\EntityType::query()->whereIn("key",["persona","digital_identity","digital_resource","person","organisation"])->get(["id","key","display_name","status"]);
$scheme=$db::table("rating_schemes")->where("key","rum_rate_anything_general_v1")->value("entity_type_constraints");
$ratings=$db::table("rating_events")->where("target_entity_id",$id)->get(["id","rating_scheme_id","source","audience","status"])->groupBy("rating_scheme_id")->map(fn($rows)=>["count"=>$rows->count(),"sources"=>$rows->pluck("source")->unique()->values()->all()]);
$profile=app(App\Services\ThingImageService::class)->publicImages($e);
$out=["id"=>$id,"canonicalName"=>$e->canonical_name,"slug"=>$e->canonical_slug,
 "type"=>$e->type?->key,"typeLabel"=>$e->type?->display_name,
 "status"=>$e->status,"visibility"=>$e->visibility,"publication"=>$e->publication_state,
 "rateability"=>$e->rateability,"reviewability"=>$e->reviewability,"sensitivity"=>$e->sensitivity,
 "editPolicy"=>$e->edit_policy,"editLocked"=>$e->edit_locked_at!==null,
 "claimed"=>$policy->isClaimed($e),"discoverable"=>$discovery->allows($e),
 "ratingOwner"=>$discovery->ratingOwner($e),"ratingEvents"=>$db::table("rating_events")->where("target_entity_id",$id)->count(),
 "ratingsByScheme"=>$ratings,"imageRevisionCount"=>$db::table("entity_profile_revisions")->where("entity_id",$id)->whereIn("field_key",["profile_image","cover_image"])->count(),
 "profilePublished"=>$profile["profile"]!==null,"coverPublished"=>$profile["cover"]!==null,
 "assignments"=>$assigned,"matchingClassifications"=>$related,
 "structuralTypes"=>$types,
 "ratingSchemeAllowsDigitalResource"=>in_array("digital_resource",json_decode((string)$scheme,true)?:[],true),
 "canonicalDuplicates"=>$db::table("entities")->where("canonical_name","CJ Investigates")->count(),
 "activePrimaryCount"=>$db::table("entity_classifications")->where("entity_id",$id)->where("is_primary",true)->whereNull("valid_to")->count()
];
echo "RAT177_CJ_TYPE_TAXONOMY=".json_encode($out,JSON_UNESCAPED_SLASHES)."\n";
';
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$code"
echo RAT177_CJ_TYPE_READONLY_PASS=true
