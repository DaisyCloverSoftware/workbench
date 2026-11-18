#!/usr/bin/env bash
set -Eeuo pipefail
# Owner-authorised, bounded recovery of EXACT pre-existing processed CJ image.
# Use existing application's author-owned publishExistingImage service, never
# direct table writes, new image import, auth/claim grants, or staff override.
[[ $# -eq 1 && ( "$1" == "check" || "$1" == "apply" ) ]] || { echo 'Usage: check|apply' >&2; exit 64; }
mode="$1"
sha='9c8af9fe07e9a0c4b0b1896f4be1903719b781e3'
api='ghcr.io/daisycloversoftware/rum-api@sha256:76731a87574a11547938a398dc15ecf87ced56d63b6622cd50bab5836dfeee71'
rat='ghcr.io/daisycloversoftware/rum-rate-anything@sha256:3938ed73de97546fccc527f9025cd5b7b21fb56d69bd4eb9be3a7ebabffb3919'
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api" ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == "$rat" ]] || exit 78
[[ "$(curl -fsS --max-time 20 https://dev-rum-ra.daisycloversoftware.uk/VERSION | tr -d '\r\n')" == "rat-candidate+$sha" ]] || exit 78
health="$(curl -fsS --max-time 20 https://dev-rum.daisycloversoftware.uk/api/v1/health)"
python3 - "$health" "$sha" <<'PY'
import sys,json
v=json.loads(sys.argv[1])
assert v.get("status")=="ok" and v.get("service")=="rum-api"
assert sys.argv[2] in str(v.get("version",""))
PY
code='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$mode=$argv[1];
$id="01m07d2f9fj05f4ypdbwp087qj";
$revisionId="01m7q2fdfz4kb104tbgbjr6veh";
$mediaId="01m7q2fbv84465vcd0jsng1vbp";
$expectedUrlSha="e099fd23338bf840707da079d96a3c33015c61c326643f971ae8ac366c4bec43";
$expectedWebpSha="55124575ce8c175d9def68e6918a269b876f5b88e4ceff7498b902011684e1a9";
$e=App\Models\Entity::query()->findOrFail($id);
$revision=App\Models\EntityProfileRevision::query()->where("entity_id",$id)->findOrFail($revisionId);
$media=App\Models\MediaAsset::query()->findOrFail($mediaId);
$author=App\Models\User::query()->findOrFail($revision->editor_user_id);
$p=app(App\Services\RatCommunityContributionPolicy::class);
$d=app(App\Services\RatDiscoveryPolicy::class);
$db=Illuminate\Support\Facades\DB::class;
$mediaMeta=(array)($revision->metadata??[]);
if($e->canonical_name!=="CJ Investigates"||$e->type?->key!=="persona"||$e->status!=="unverified"
 ||$e->visibility!=="public"||$e->publication_state!=="published"||$e->edit_policy!=="claimant_controlled"
 ||$e->edit_locked_at!==null||!$d->allows($e)||$p->isClaimed($e)||!$p->canDirectEdit($author,$e)
 ||$author->status!=="active")exit(78);
if($db::table("entity_claims")->where("entity_id",$id)->where("status","approved")->whereNull("revoked_at")->exists()
 ||$db::table("entity_managers")->where("entity_id",$id)->where("status","active")->whereNull("revoked_at")->exists())exit(78);
if($db::table("entity_profile_revisions")->where("entity_id",$id)->whereIn("field_key",["profile_image","cover_image"])->count()!==1)exit(78);
if($revision->field_key!=="profile_image"||$revision->outcome!=="pending"||$revision->moderation_status!=="pending"
 ||(string)data_get($mediaMeta,"mediaAssetId")!==$mediaId||data_get($mediaMeta,"imageSource")!=="linked"
 ||$revision->previous_value!==null||!is_string($revision->proposed_value)
 ||hash("sha256",$revision->proposed_value)!==$expectedUrlSha
 ||parse_url($revision->proposed_value,PHP_URL_HOST)!=="yt3.googleusercontent.com")exit(78);
if((string)$media->owner_id!==(string)$author->id||$media->purpose!=="thing_image"
 ||$media->status!=="ready"||$media->safety_state!=="not_flagged"||$media->deleted_at!==null
 ||$media->mime_type!=="image/webp"||$media->sha256!==$expectedWebpSha
 ||$media->attachment_type!=="entity_profile_revision"||(string)$media->attachment_id!==$revisionId
 ||!$media->display_key)exit(78);
$disk=Illuminate\Support\Facades\Storage::disk($media->storage_disk);
if(!$disk->exists($media->display_key))exit(78);
$bytes=$disk->get($media->display_key);
if(!is_string($bytes)||hash("sha256",$bytes)!==$expectedWebpSha)exit(78);
$info=getimagesizefromstring($bytes);
if(!is_array($info)||($info["mime"]??"")!=="image/webp"||($info[0]??0)<1||($info[1]??0)<1)exit(78);
if(app(App\Services\ThingImageService::class)->publicImages($e)["profile"]!==null)exit(78);
$ratings=$db::table("rating_events")->where("target_entity_id",$id)->count();
echo "CJ_PUBLISH_PREFLIGHT=".json_encode(["eligible"=>true,"claimed"=>false,"revision"=>$revisionId,
"media"=>$mediaId,"assetSha256"=>$expectedWebpSha,"sanitisedMime"=>$media->mime_type,
"dimensions"=>[(int)$info[0],(int)$info[1]],"imageRevisions"=>1,"ratingCount"=>$ratings,
"canDirectEdit"=>$p->canDirectEdit($author,$e)],JSON_UNESCAPED_SLASHES)."\n";
if($mode==="check"){echo "CJ_EXISTING_IMAGE_SERVICE_CHECK_PASS=true\n";exit(0);}
try{
 $published=app(App\Services\ThingEditService::class)->publishExistingImage($author,$e,$revision);
}catch(Throwable $error){
 echo "CJ_EXISTING_IMAGE_PUBLISH_FAILED=".get_class($error)."\n";
 exit(78);
}
$after=$e->fresh();
$public=app(App\Services\ThingImageService::class)->publicImages($after)["profile"];
if($published->id!==$revisionId||$published->outcome!=="applied"||$published->moderation_status!=="approved"
 ||$public===null||!str_contains((string)($public["url"]??""),$mediaId)
 ||$db::table("entity_profile_revisions")->where("entity_id",$id)->whereIn("field_key",["profile_image","cover_image"])->count()!==1
 ||$db::table("rating_events")->where("target_entity_id",$id)->count()!==$ratings
 ||$p->isClaimed($after)||$after->status!=="unverified")exit(78);
echo "CJ_EXISTING_IMAGE_PUBLISHED=".json_encode(["entity"=>$id,"revision"=>$revisionId,
"media"=>$mediaId,"revisionCount"=>1,"publicUrlMediaMatches"=>true,"ratingCountUnchanged"=>true,
"outcome"=>$published->outcome,"moderation"=>$published->moderation_status],JSON_UNESCAPED_SLASHES)."\n";
';
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$code" "$mode"
echo "RAT175_CJ_EXISTING_IMAGE_SERVICE_${mode^^}_DONE=true"
