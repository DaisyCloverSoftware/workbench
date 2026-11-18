#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only predeployment state of the exact public CJ Investigates entity.
[[ $# -eq 0 ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "ghcr.io/daisycloversoftware/rum-api@sha256:098859cf56cac950e32b39464b2fdc7e4ddfa128d4e9ad95e2423c88b2331e7b" ]] || exit 78
php_code='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$id="01m07d2f9fj05f4ypdbwp087qj";$db=Illuminate\Support\Facades\DB::class;
$entity=App\Models\Entity::query()->findOrFail($id);
$policy=app(App\Services\RatCommunityContributionPolicy::class);
$revs=$db::table("entity_profile_revisions")->where("entity_id",$id)->whereIn("field_key",["profile_image","cover_image"])->get();
$approved=$revs->filter(static fn($r)=>$r->outcome==="applied" && $r->moderation_status==="approved")->count();
print "CJ_PUBLIC_STATE=".json_encode(["id"=>(string)$entity->id,"name"=>(string)$entity->canonical_name,"type"=>$entity->type?->key,"status"=>$entity->status,"publication"=>$entity->publication_state,"visibility"=>$entity->visibility,"editPolicy"=>$entity->edit_policy,"editLocked"=>$entity->edit_locked_at!==null,"claimed"=>$policy->isClaimed($entity),"claimCount"=>$db::table("entity_claims")->where("entity_id",$id)->where("status","approved")->whereNull("revoked_at")->count(),"managerCount"=>$db::table("entity_managers")->where("entity_id",$id)->where("status","active")->whereNull("revoked_at")->count(),"profileImageRevisions"=>$revs->where("field_key","profile_image")->count(),"approvedImageRevisions"=>$approved,"publicProfileImage"=>app(App\Services\ThingImageService::class)->publicImages($entity)["profile"]!==null,"version"=>$entity->version],JSON_UNESCAPED_SLASHES)."\n";
';
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$php_code"
echo CJ_READONLY_STATE_COMPLETE=true
