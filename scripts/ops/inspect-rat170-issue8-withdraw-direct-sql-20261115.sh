#!/usr/bin/env bash
set -euo pipefail
# Read-only canonical DEV provenance of synthetic pending -> superseded/withdrawn.
[[ $# -eq 5 ]] || exit 64
for id in "$1" "$2" "$3" "$5"; do [[ "$id" =~ ^[a-z0-9]{26}$ ]] || exit 64; done
[[ "$4" =~ ^[a-z0-9-]{8,42}$ ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
[[ "$(kctl -n rum-dev-isolated get ingress -o jsonpath='{range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)" == 'dev-rum.daisycloversoftware.uk' ]] || exit 78
php='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";
$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
[$uid,$original,$revised,$rid,$direct]=array_slice($argv,1);
$db=Illuminate\Support\Facades\DB::class;
$root="01m7d2emwsnd0nxw6new8khbje";
$owner="01m7bswfh0a3evf271ye4taaf0";
$entity=$db::table("entities")->where("id",$root)->first();
$first=$db::table("entity_profile_revisions")->where("id",$original)->first();
$second=$db::table("entity_profile_revisions")->where("id",$revised)->first();
$directRevision=$db::table("entity_profile_revisions")->where("id",$direct)->first();
$expectedDescription="Issue8 DEV independent community description ".$rid;
$ownerIds=["01m7dxvjchbmnjjnatw0av6a7m","01m7dxwv5cnczfsjset8emjvap"];
$ownerRows=$db::table("entity_profile_revisions")->where("entity_id",$owner)->whereIn("id",$ownerIds)->get();
$oneMeta=$first&&$first->metadata?json_decode((string)$first->metadata,true):[];
$secondMeta=$second&&$second->metadata?json_decode((string)$second->metadata,true):[];
$events=$db::table("outbox_events")->where("aggregate_type","entity")->where("aggregate_id",$root)->get(["event_type","payload"]);
$hasEvent=fn($event,$id)=>$events->contains(function($r)use($event,$id){
 $p=is_string($r->payload)?json_decode($r->payload,true):(array)$r->payload;
 return $r->event_type===$event&&($p["revisionId"]??null)===$id;
});
$result=[
 "syntheticThingPublic"=>(bool)($entity&&$entity->canonical_name==="Carmine Nebula Archive rat170-community-a01"
   &&$entity->visibility==="public"&&$entity->publication_state==="published"),
 "differentFromThingCreator"=>(bool)($entity&&$entity->created_by_user_id!==$uid),
 "originalAuthorMatches"=>(bool)($first&&$first->editor_user_id===$uid&&$first->entity_id===$root),
 "replacementAuthorMatches"=>(bool)($second&&$second->editor_user_id===$uid&&$second->entity_id===$root),
 "bothImageReferences"=>(bool)($first&&$second&&$first->field_key==="image_url"&&$second->field_key==="image_url"),
 "originalSuperseded"=>(bool)($first&&$first->outcome==="superseded"&&$first->moderation_status==="withdrawn"),
 "replacementLinkRecorded"=>(bool)(($oneMeta["supersededByRevisionId"]??null)===$revised),
 "replacementWithdrawn"=>(bool)($second&&$second->outcome==="withdrawn"&&$second->moderation_status==="withdrawn"),
 "withdrawnTimestampPresent"=>(bool)($secondMeta["withdrawnAt"]??null),
 "submittedOriginalEvent"=>$hasEvent("entity.profile_revision_submitted.v1",$original),
 "submittedRevisedEvent"=>$hasEvent("entity.profile_revision_submitted.v1",$revised),
 "supersedeEvent"=>$hasEvent("entity.profile_revision_superseded.v1",$original),
 "withdrawEvent"=>$hasEvent("entity.profile_revision_withdrawn.v1",$revised),
 "directDescriptionPersisted"=>(bool)($entity&&$entity->description===$expectedDescription),
 "directRevisionApplied"=>(bool)($directRevision&&$directRevision->entity_id===$root&&$directRevision->editor_user_id===$uid&&$directRevision->field_key==="description"&&$directRevision->source==="community"&&$directRevision->moderation_status==="approved"&&$directRevision->outcome==="applied"&&$directRevision->proposed_value===$expectedDescription),
 "ownerImageRowsPreserved"=>(bool)($ownerRows->count()===2 && $ownerRows->every(fn($r)=>$r->outcome==="pending"&&$r->moderation_status==="pending")),
 "noClaimCreatedForSyntheticThing"=>(int)$db::table("entity_claims")->where("entity_id",$root)->count()===0,
 "noMutation"=>true,
];
echo "RAT170_ISSUE8_CANONICAL_SQL=".json_encode($result,JSON_THROW_ON_ERROR)."\n";
foreach($result as $key=>$v){if($key!=="noMutation"&&$v!==true)exit(2);}
';
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$php" -- "$@"
echo RAT170_ISSUE8_READONLY_SQL_PASS=true
