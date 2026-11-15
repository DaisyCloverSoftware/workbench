#!/usr/bin/env bash
set -euo pipefail
# Fixed, read-only isolated canonical DEV inspection of one owner-named public Thing.
# No URL values, evidence, emails, account IDs, or source bytes are emitted.
[[ $# -eq 0 ]] || { echo 'No arguments permitted' >&2; exit 64; }
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
[[ "$(kctl -n rum-dev-isolated get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)" == "dev-rum.daisycloversoftware.uk" ]] || exit 78
php='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";
$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
$eid="01m7bswfh0a3evf271ye4taaf0";
$e=$db::table("entities")->where("id",$eid)->first();
if(!$e || $e->canonical_name!=="The Professional Idiots") exit(78);
$activeClaims=$db::table("entity_claims")->where("entity_id",$eid)->where("status","approved")->whereNull("revoked_at")->get();
$activeManagers=$db::table("entity_managers")->where("entity_id",$eid)->where("status","active")->whereNull("revoked_at")->get();
$revisions=$db::table("entity_profile_revisions")->where("entity_id",$eid)->whereIn("field_key",["image_url","profile_image","cover_image"])->orderByDesc("occurred_at")->limit(40)->get();
$out=[];
foreach($revisions as $r){
 $meta=$r->metadata===null?[]:(is_string($r->metadata)?json_decode($r->metadata,true):(array)$r->metadata);
 $assetId=is_array($meta)?($meta["mediaAssetId"]??null):null;
 $a=is_string($assetId)?$db::table("media_assets")->where("id",$assetId)->first():null;
 $out[]=[
  "revisionId"=>(string)$r->id,"field"=>(string)$r->field_key,
  "outcome"=>(string)$r->outcome,"moderationStatus"=>(string)$r->moderation_status,
  "source"=>(string)$r->source,"sourceType"=>(is_array($meta)?($meta["imageSource"]??null):null),
  "hasProcessedMedia"=>$a!==null,"mediaId"=>$a?->id,"mediaStatus"=>$a?->status,
  "isWebp"=>$a?->mime_type==="image/webp","mediaAttachedToRevision"=>$a!==null&&$a->attachment_type==="entity_profile_revision"&&(string)$a->attachment_id===(string)$r->id,
  "sourceHash12"=>substr(hash("sha256",(string)($r->proposed_value??"")),0,12),
  "occurredAt"=>$r->occurred_at,
 ];
}
$published=[];
foreach(["profile_image","cover_image"] as $field){
 $r=$db::table("entity_profile_revisions")->where("entity_id",$eid)->where("field_key",$field)
  ->where("outcome","applied")->where("moderation_status","approved")->orderByDesc("occurred_at")->orderByDesc("id")->first();
 $meta=$r===null?[]:(is_string($r->metadata)?json_decode($r->metadata,true):(array)$r->metadata);
 $published[$field]=$r===null?null:["revisionId"=>$r->id,"mediaId"=>$meta["mediaAssetId"]??null];
}
$checks=[];
foreach(["01m7dxvjchbmnjjnatw0av6a7m","01m7dxwv5cnczfsjset8emjvap"] as $id){
 $r=$db::table("entity_profile_revisions")->where("id",$id)->where("entity_id",$eid)->first();
 $checks[]=["id"=>$id,"present"=>$r!==null,"outcome"=>$r?->outcome,"status"=>$r?->moderation_status,"hasMediaMetadata"=>$r?->metadata!==null];
}
echo "RAT170_OWNER_READONLY=".json_encode([
 "entityId"=>$eid,"visibility"=>$e->visibility,"publicationState"=>$e->publication_state,
 "editPolicy"=>$e->edit_policy,"locked"=>$e->edit_locked_at!==null,
 "activeVerifiedClaims"=>$activeClaims->filter(fn($c)=>(int)$c->verification_level>=1)->count(),
 "activeManagersWithEditPermission"=>$activeManagers->filter(fn($m)=>str_contains((string)$m->permissions,"edit_public_info"))->count(),
 "currentImages"=>$published,
 "recentImageRevisions"=>$out,"historicalIds"=>$checks,"noMutation"=>true
],JSON_THROW_ON_ERROR|JSON_UNESCAPED_SLASHES)."\n";
';
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$php"
