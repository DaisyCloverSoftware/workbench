#!/usr/bin/env bash
set -euo pipefail
# Canonical isolated-DEV read-only identity, media integrity, and protected
# owner snapshot verification. Never prints source URLs, credentials or notes.
[[ $# -eq 0 ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
host="$(kctl -n rum-dev-isolated get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)"
[[ "$host" == 'dev-rum.daisycloversoftware.uk' ]] || exit 78
php='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";
$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
$fixtures=[
 ["kind"=>"profile-url","name"=>"RAT170 Image QA profile-url rat170img4way-20261115-r13","field"=>"profile_image","mode"=>"linked"],
 ["kind"=>"profile-upload","name"=>"Purple Lantern Product Fixture rat170img4way-20261115-r15","field"=>"profile_image","mode"=>"uploaded"],
 ["kind"=>"cover-url","name"=>"Green Tempest Artwork Catalogue rat170img4way-20261115-r15","field"=>"cover_image","mode"=>"linked"],
 ["kind"=>"cover-upload","name"=>"Amber Prairie Display Collection rat170img4way-20261115-r15","field"=>"cover_image","mode"=>"uploaded"]
];
$out=[];
foreach($fixtures as $f){
 $entity=$db::table("entities")->where("canonical_name",$f["name"])->first();
 if(!$entity || $entity->publication_state!=="published" || $entity->visibility!=="public")exit(2);
 $rows=$db::table("entity_profile_revisions")->where("entity_id",$entity->id)
  ->where("field_key",$f["field"])->where("outcome","pending")->orderByDesc("occurred_at")->get();
 if($rows->count()!==1)exit(2);
 $rev=$rows->first();
 $meta=is_string($rev->metadata)?json_decode($rev->metadata,true):(array)$rev->metadata;
 if(!is_array($meta)||($meta["imageSource"]??null)!==$f["mode"])exit(2);
 $assetId=$meta["mediaAssetId"]??null;
 if(!is_string($assetId))exit(2);
 $asset=$db::table("media_assets")->where("id",$assetId)->first();
 if(!$asset || $asset->purpose!=="thing_image" || $asset->status!=="ready"
  || $asset->mime_type!=="image/webp" || $asset->safety_state!=="not_flagged"
  || $asset->attachment_type!=="entity_profile_revision" || (string)$asset->attachment_id!==(string)$rev->id
  || (string)$asset->owner_id!==(string)$rev->editor_user_id
  || !$asset->display_key || !$asset->thumbnail_key)exit(2);
 if($rev->moderation_status!=="pending")exit(2);
 $out[]=["kind"=>$f["kind"],"entityId"=>$entity->id,"revisionId"=>$rev->id,"mediaAssetId"=>$assetId,
  "field"=>$f["field"],"sourceType"=>$f["mode"],"moderationStatus"=>$rev->moderation_status,
  "outcome"=>$rev->outcome,"mediaPurpose"=>$asset->purpose,"attachmentVerified"=>true,"processedWebp"=>true,
  "privatePending"=>true];
}
$legacy=$db::table("media_assets")->where("purpose","<>","thing_image")->orderBy("id")->get();
$legacyFingerprint=hash("sha256",json_encode($legacy->map(fn($row)=>(array)$row)->all(),JSON_THROW_ON_ERROR));
if($legacy->count()!==2 || $legacyFingerprint!=="438b32173d5bf651c68811e040f83ac154534dc69e136dc3cec40b9ccf521db5")exit(2);
$owner=$db::table("entities")->where("id","01m7bswfh0a3evf271ye4taaf0")->first();
if(!$owner || $owner->canonical_name!=="The Professional Idiots")exit(2);
$ids=["01m7dxvjchbmnjjnatw0av6a7m","01m7dxwv5cnczfsjset8emjvap"];
$protected=[];
foreach($ids as $id){
 $rev=$db::table("entity_profile_revisions")->where("id",$id)->where("entity_id",$owner->id)->first();
 if(!$rev || $rev->outcome!=="pending" || $rev->moderation_status!=="pending"
  || substr(hash("sha256",(string)$rev->proposed_value),0,12)!=="65bf10c7b074"
  || $rev->metadata!==null)exit(2);
 $protected[]=["revisionId"=>$id,"outcome"=>$rev->outcome,"moderationStatus"=>$rev->moderation_status,
  "proposedValueHash12"=>"65bf10c7b074","metadataUnchanged"=>true];
}
$names=$db::select("SELECT conname,pg_get_constraintdef(oid) AS definition FROM pg_constraint WHERE conrelid = '\''media_assets'\''::regclass AND conname IN ('\''media_assets_purpose_allowed'\'','\''media_assets_attachment_type_allowed'\'')");
if(count($names)!==2 || !$db::table("migrations")->where("migration","2026_11_15_000001_allow_thing_image_media_purpose")->exists())exit(2);
$map=[];
foreach($names as $row)$map[$row->conname]=$row->definition;
if(!str_contains($map["media_assets_purpose_allowed"],"thing_image")||!str_contains($map["media_assets_attachment_type_allowed"],"entity_profile_revision"))exit(2);
echo "RAT170_FOUR_IMAGE_CANONICAL_READONLY=".json_encode([
 "fourPendingImages"=>$out,"existingLegacyMediaCount"=>$legacy->count(),"existingLegacyMediaFingerprintUnchanged"=>true,
 "allMediaCount"=>$db::table("media_assets")->count(),
 "protectedOwnerRevisions"=>$protected,"constraintsVerified"=>true,"migrationRecorded"=>true,"noMutation"=>true
],JSON_THROW_ON_ERROR)."\n";
';
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$php"
echo RAT170_FOUR_IMAGE_CANONICAL_DB_PASS=true
