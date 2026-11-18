#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only CJ Thing identity, type, revision, profile, cover and ratings audit.
[[ $# -eq 4 ]] || exit 64
source="$1";api="$2";rat="$3";expected="$4"
[[ "$source" == '628296e5c57665814799bfedf1ab3ef8f6f163df' ]] || exit 78
[[ "$api" =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ ]] || exit 64
[[ "$rat" =~ ^ghcr\.io/daisycloversoftware/rum-rate-anything@sha256:[0-9a-f]{64}$ ]] || exit 64
[[ "$api" == '__RAT177_API_DIGEST_PENDING__' ]] || exit 78
[[ "$rat" == '__RAT177_RAT_DIGEST_PENDING__' ]] || exit 78
[[ "$expected" == persona || "$expected" == digital_resource ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api" ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == "$rat" ]] || exit 78
php='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$expected=$argv[1];$db=Illuminate\Support\Facades\DB::class;
$id="01m07d2f9fj05f4ypdbwp087qj";$e=App\Models\Entity::query()->with("type")->findOrFail($id);
$p=app(App\Services\RatCommunityContributionPolicy::class);
$assignment=$db::table("entity_classifications as a")->join("classifications as c","c.id","=","a.classification_id")
 ->where("a.entity_id",$id)->whereNull("a.valid_to")->where("a.is_primary",true)
 ->first(["a.id","c.key","c.canonical_name"]);
$imgs=app(App\Services\ThingImageService::class)->publicImages($e);
$typed=$db::table("entity_profile_revisions")->where("entity_id",$id)->where("field_key","thing_type")->get(["id","outcome","moderation_status","editor_user_id","reverts_revision_id"]);
$expectedClass=$expected==="persona"?"youtube_creator":"community_788e6ae3bfcd391d111e3ca31a1332ebf92906d0d0868e7cf7bb";
$oldEnded=$db::table("entity_classifications as a")->join("classifications as c","c.id","=","a.classification_id")
 ->where("a.entity_id",$id)->whereNotNull("a.valid_to")->where("c.key","youtube_creator")->count();
if($e->canonical_name!=="CJ Investigates"||$e->type?->key!==$expected
 ||$e->status!=="unverified"||$e->visibility!=="public"||$e->publication_state!=="published"
 ||$e->rateability!=="open"||$p->isClaimed($e)||$e->edit_locked_at!==null
 ||$assignment?->key!==$expectedClass||$db::table("rating_events")->where("target_entity_id",$id)->count()!==54
 ||$db::table("entities")->where("canonical_name","CJ Investigates")->count()!==1
 ||$imgs["profile"]===null||$imgs["cover"]===null)exit(78);
if($expected==="digital_resource"&&($oldEnded<1||$typed->count()!==1||$typed[0]->outcome!=="applied"
 ||$typed[0]->moderation_status!=="approved"||!$typed[0]->editor_user_id))exit(78);
echo "RAT177_CANONICAL_CJ_STATE=".json_encode(["entityId"=>$id,"structuralType"=>$e->type?->key,
 "primaryClassification"=>$assignment->canonical_name,"ratingEvents"=>54,"profilePublished"=>true,
 "coverPublished"=>true,"typeEditRevisions"=>$typed->count(),"oldYouTubeCreatorEnded"=>$oldEnded>0,
 "unclaimed"=>true,"duplicateEntities"=>false],JSON_UNESCAPED_SLASHES)."\n";
';
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$php" "$expected"
echo RAT177_READONLY_CJ_CANONICAL_STATE_PASS=true