#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only canonical DEV public Things, audience policies and rating counts.
[[ $# -eq 0 ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == 'ghcr.io/daisycloversoftware/rum-api@sha256:7411cc02eb6568e6c6d228b551683294bfac49a6f89c320078af5550adce5d56' ]] || exit 78
code='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
$rum=app(App\Services\RumEntityAudiencePolicy::class);
$rat=app(App\Services\RatDiscoveryPolicy::class);
$out=[];
foreach(["CJ Investigates","Mark Marshall"] as $name){
 $entities=App\Models\Entity::query()->with("type","classificationAssignments.classification")
 ->where("canonical_name",$name)->where("visibility","public")->whereNotIn("status",["deleted","merged"])->get();
 $out[$name]=$entities->map(fn($e)=>[
  "id"=>(string)$e->id,"name"=>$e->canonical_name,"entityType"=>$e->type?->key,
  "publication"=>$e->publication_state,"ratingEvents"=>$db::table("rating_events")->where("target_entity_id",$e->id)->count(),
  "rumEligible"=>$rum->allows($e),"ratEligible"=>$rat->allows($e),
  "primary"=>$e->classificationAssignments->filter(fn($a)=>$a->valid_to===null&&$a->is_primary)
    ->map(fn($a)=>["key"=>$a->classification?->key,"name"=>$a->classification?->canonical_name])->values()->all()
  ])->values()->all();
}
echo "RAT180_CJ_MARK_PUBLIC_AUDIENCE=".json_encode($out,JSON_UNESCAPED_SLASHES)."\n";
';
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$code"
echo RAT180_CJ_MARK_READONLY_COMPLETE=true
