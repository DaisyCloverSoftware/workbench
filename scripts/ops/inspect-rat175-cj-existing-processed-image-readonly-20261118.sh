#!/usr/bin/env bash
set -Eeuo pipefail
# Strictly read-only CJ Investigates canonical DEV revision and sanitized-media inspection.
# No cookies, raw image URLs, account names or media contents leave the runtime.
[[ $# -eq 0 ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
api='ghcr.io/daisycloversoftware/rum-api@sha256:098859cf56cac950e32b39464b2fdc7e4ddfa128d4e9ad95e2423c88b2331e7b'
rat='ghcr.io/daisycloversoftware/rum-rate-anything@sha256:6b9d8d0e5f31a94800c48ead1b66f06fba451c3458f436a54d857a173a152f7d'
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api" ]] || { echo 'CJ_INSPECT_API_IMAGE_GUARD_BLOCKED'; exit 78; }
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == "$rat" ]] || { echo 'CJ_INSPECT_RAT_IMAGE_GUARD_BLOCKED'; exit 78; }
code='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$id="01m07d2f9fj05f4ypdbwp087qj";
$e=App\Models\Entity::query()->findOrFail($id);
$db=Illuminate\Support\Facades\DB::class;
$policy=app(App\Services\RatCommunityContributionPolicy::class);
if($e->canonical_name!=="CJ Investigates"||$e->type?->key!=="persona")exit(78);
$rq=App\Models\EntityProfileRevision::query()->where("entity_id",$id)
 ->whereIn("field_key",["profile_image","cover_image"])->orderByDesc("occurred_at")->orderByDesc("id");
$revs=$rq->get();
$result=[];
foreach($revs as $r){
 $meta=(array)($r->metadata??[]);$assetId=data_get($meta,"mediaAssetId");
 $m=is_string($assetId)?App\Models\MediaAsset::query()->find($assetId):null;
 $u=$r->editor_user_id?App\Models\User::query()->find($r->editor_user_id):null;
 $uri=$r->proposed_value??"";$urlhost="";
 if(is_string($uri)&&str_starts_with($uri,"https://"))$urlhost=parse_url($uri,PHP_URL_HOST)?:null;
 $objExists=$m && $m->display_key ? Illuminate\Support\Facades\Storage::disk($m->storage_disk)->exists($m->display_key) : false;
 $result[]=["revisionId"=>(string)$r->id,"field"=>$r->field_key,"outcome"=>$r->outcome,
 "moderation"=>$r->moderation_status,"imageSource"=>data_get($meta,"imageSource"),
 "sourceHost"=>$urlhost,"supersedesRevisionId"=>$r->supersedes_revision_id??null,
 "proposedValueHash"=>is_string($r->proposed_value)?hash("sha256",$r->proposed_value):null,
 "authorExists"=>$u!==null,"authorActive"=>$u?->status==="active",
 "authorCanDirectEditUnderRunningCode"=>$u?$policy->canDirectEdit($u,$e):null,
 "mediaId"=>$m?->id,"assetOwnerMatchesEditor"=>$m&&$u?((string)$m->owner_id===(string)$u->id):false,
 "mediaStatus"=>$m?->status,"purpose"=>$m?->purpose,"mime"=>$m?->mime_type,
 "mediaSafety"=>$m?->safety_state,"mediaSha256"=>$m?->sha256,"bytes"=>$m?->byte_size,
 "attachmentType"=>$m?->attachment_type,"attachmentMatchesRevision"=>$m?((string)$m->attachment_id===(string)$r->id):false,
 "storedObjectExists"=>$objExists,"deleted"=>$m?->deleted_at!==null,"hasDisplayKey"=>(bool)$m?->display_key];
}
$public=app(App\Services\ThingImageService::class)->publicImages($e);
echo "CJ_EXISTING_IMAGE_DIAGNOSTIC=".json_encode(["entity"=>$id,"status"=>$e->status,"visibility"=>$e->visibility,
 "publication"=>$e->publication_state,"locked"=>$e->edit_locked_at!==null,"isClaimed"=>$policy->isClaimed($e),
 "revisionCount"=>$revs->count(),"revisions"=>$result,"publicProfile"=>$public["profile"]!==null,
 "version"=>$e->version],JSON_UNESCAPED_SLASHES)."\n";
';
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$code"
echo CJ_EXISTING_IMAGE_READONLY_INSPECTION_PASS=true
