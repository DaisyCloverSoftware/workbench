#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only check for existing synthetic RAT public-persona fixture suitable for guarded browser testing.
[[ $# -eq 0 ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == 'ghcr.io/daisycloversoftware/rum-api@sha256:098859cf56cac950e32b39464b2fdc7e4ddfa128d4e9ad95e2423c88b2331e7b' ]] || exit 78
code='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$id="01m784s54kww2vqw891mnmce71";$e=App\Models\Entity::query()->find($id);
if(!$e){echo "QA_DEMO_PERSONA=missing\n";exit(0);}
$db=Illuminate\Support\Facades\DB::class;$policy=app(App\Services\RatCommunityContributionPolicy::class);
$revs=$db::table("entity_profile_revisions")->where("entity_id",$id)->where("field_key","profile_image")->count();
echo "QA_DEMO_PERSONA=".json_encode(["id"=>$id,"name"=>$e->canonical_name,"type"=>$e->type?->key,"status"=>$e->status,"visibility"=>$e->visibility,"publication"=>$e->publication_state,"editPolicy"=>$e->edit_policy,"claimed"=>$policy->isClaimed($e),"editLocked"=>$e->edit_locked_at!==null,"profileRevisions"=>$revs,"profilePublished"=>app(App\Services\ThingImageService::class)->publicImages($e)["profile"]!==null],JSON_UNESCAPED_SLASHES)."\n";
'
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$code"
echo RAT175_DEMO_PERSONA_READONLY_COMPLETE=true
