#!/usr/bin/env bash
set -Eeuo pipefail
# Independent post-mutation, read-only canonical audit, source-image pinned to isolated DEV.
[[ $# -eq 0 ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == 'ghcr.io/daisycloversoftware/rum-api@sha256:a23e36323b55f3bce46d5b229146e2e7a9d70bdff9e16ea366380940e0629430' ]] || exit 78
php="$(cat <<'RAT_CJ_POST_AUDIT_PHP'

require "vendor/autoload.php";
$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
$id="01m07d2f9fj05f4ypdbwp087qj";$keep="01m0mvrxv8c3s12q0w7gwmzt03";$owner="01kzgn6a7kswt3jqjb1sp8d4x0";
$events=App\Models\RatingEvent::query()->where("target_entity_id",$id)->orderBy("id")->get();
$remaining=$events->where("status","active");
if($events->count()!==54||$remaining->count()!==1||$remaining->first()->id!==$keep||
 $events->where("status","removed")->count()!==47||$events->where("status","withdrawn")->count()!==6)
 throw new RuntimeException("Rating lifecycle not in single-owner-final state");
$chosen=App\Models\RatingEvent::query()->with(["rater","scheme","review","reasonSelections","admissionState","admissionVotes","revisions"])->findOrFail($keep);
if($chosen->rater_account_id!==$owner||$chosen->rater?->founder_number!==1||(int)$chosen->value!==5||
 $chosen->status!=="active"||$chosen->withdrawn_at!==null||
 $chosen->submitted_at?->format("Y-m-d H:i:s")!=="2026-08-22 13:50:12"||
 $chosen->source!=="canonical_rum"||$chosen->audience!=="public"||
 $chosen->reasonSelections->isNotEmpty()||$chosen->rating_context_id!==null||
 hash("sha256",(string)$chosen->review?->body)!=="aa2bcc4c7b1bf6348c46944f1bebd54298d661e4e2db030c08170a99b7d98e2e"||
 $chosen->admissionState?->state!=="not_eligible"||$chosen->admissionVotes->isNotEmpty())
 throw new RuntimeException("Protected genuine review or verdict state not preserved");
$restoreRevs=$chosen->revisions->where("revision_type","dev_data_cleanup_restored");
$removedIds=$events->where("status","removed")->pluck("id")->all();
$removedRevs=$db::table("rating_event_revisions")->whereIn("rating_event_id",$removedIds)->where("revision_type","dev_data_cleanup_removed")->get(["rating_event_id"]);
if($restoreRevs->count()!==1||$removedRevs->count()!==47)throw new RuntimeException("Maintenance audit revisions incomplete");
$votes=$db::table("rating_event_admission_votes")->whereIn("rating_event_id",$events->pluck("id")->all())->get(["id","rating_event_id","voter_id"]);
$founderVotes=$votes->where("voter_id",$owner)->count();
if($votes->count()!==17||$founderVotes!==5||$votes->whereIn("rating_event_id",$removedIds)->count()!==17)
 throw new RuntimeException("Community vote evidence unexpectedly changed");
$states=$db::table("rating_event_admission_states")->whereIn("rating_event_id",$removedIds)->get(["rating_event_id","state"]);
if($states->count()!==47||$states->where("state","removed")->count()!==47)
 throw new RuntimeException("Retired ratings still eligible for Judge");
$marker=$db::table("outbox_events")->where("aggregate_type","entity")->where("aggregate_id",$id)
 ->where("event_type","rating.dev_owner_original_cleanup_completed.v1")
 ->where("payload->operationId","ratcj-20261118-owner-original-rating-cleanup-r1")->count();
$changedOutbox=$db::table("outbox_events")->where("aggregate_type","rating_event")
 ->whereIn("aggregate_id",$events->pluck("id")->all())->where("event_type","rating.event_moderation_changed.v1")
 ->where("payload->maintenanceOperationId","ratcj-20261118-owner-original-rating-cleanup-r1")->count();
if($marker!==1||$changedOutbox!==48)throw new RuntimeException("Lifecycle event-outbox evidence incomplete");
$entity=App\Models\Entity::query()->with(["type","classificationAssignments.classification"])->findOrFail($id);
$primary=$entity->classificationAssignments->first(fn($a)=>(bool)$a->is_primary&&$a->valid_to===null);
$images=app(App\Services\ThingImageService::class)->publicImages($entity);
if($entity->canonical_name!=="CJ Investigates"||$entity->type?->key!=="digital_resource"||
 $primary?->classification?->canonical_name!=="YouTube Channel"||$entity->publication_state!=="published"||
 $entity->rateability!=="open"||$entity->visibility!=="public"||$images["profile"]===null||$images["cover"]===null||
 $db::table("entity_relationships")->where(fn($q)=>$q->where("source_entity_id",$id)->orWhere("target_entity_id",$id))->whereNull("valid_to")->count()!==0)
 throw new RuntimeException("CJ entity protections invalid");
$algo=App\Models\ReputationAlgorithmVersion::query()->where("key","rum_rate_anything_reputation_v2")->where("status","active")->firstOrFail();
$scheme=App\Models\RatingScheme::query()->where("key","rum_rate_anything_general_v1")->where("status","active")->firstOrFail();
$projection=app(App\Services\Reputation\ReputationCalculationService::class)->calculate($entity,$scheme,$algo,null);
$eligible=[];$ineligibleRemoved=0;
foreach($projection->weightAssessments as $a){
 if($a->eligible&&$a->effectiveWeight>0)$eligible[]=$a->ratingEventId;
 if(in_array($a->ratingEventId,$removedIds,true) && (!$a->eligible||$a->effectiveWeight<=0))$ineligibleRemoved++;
}
if(count(array_diff($eligible,[$keep]))!==0||$ineligibleRemoved!==47)
 throw new RuntimeException("Removed ratings have nonzero current reputation weight");
$current=$db::table("entity_reputation_snapshots")->where("entity_id",$id)->where("is_current",true)->orderByDesc("calculated_at")->limit(4)->get(["id","rating_count","score","confidence_label","calculated_at","algorithm_version_id"]);
$given=null;$givenError=null;
try{
 $founder=App\Models\User::query()->findOrFail($owner);
 $profile=app(App\Services\ProfileService::class)->profile($founder,$founder,null,null);
 $matches=collect($profile["data"]["ratingsGiven"]??[])->filter(fn($r)=>(string)($r["id"]??"")===$keep);
 $given=$matches->count()===1?$matches->first():null;
}catch(Throwable $e){$givenError=get_class($e).": ".mb_substr($e->getMessage(),0,180);}
if($given!==null){
 if((int)($given["value"]??0)!==5||hash("sha256",(string)($given["reason"]??""))!=="aa2bcc4c7b1bf6348c46944f1bebd54298d661e4e2db030c08170a99b7d98e2e")
  throw new RuntimeException("Owner Given-history wording differs from protected original");
}
echo "RATCJ_RATING_INDEPENDENT_POSTCHECK=".json_encode([
 "history"=>54,"active"=>1,"inactive"=>53,"removed"=>47,"withdrawn"=>6,
 "original"=>["id"=>$keep,"source"=>$chosen->source,"value"=>(int)$chosen->value,"authorId"=>$chosen->rater_account_id,
 "submitted"=>$chosen->submitted_at?->toAtomString(),"reviewSha256"=>hash("sha256",(string)$chosen->review?->body),
 "admissionState"=>$chosen->admissionState?->state,"revisions"=>$chosen->revisions->count()],
 "audit"=>["restoreRevisions"=>1,"retireRevisions"=>47,"maintenanceOutboxEvents"=>$changedOutbox,"completionMarkers"=>$marker],
 "votes"=>["historicalPreserved"=>$votes->count(),"founderOnQA"=>$founderVotes,"allLinkedToRetired"=>$votes->whereIn("rating_event_id",$removedIds)->count()],
 "reputation"=>["currentSnapshotCount"=>$current->count(),"currentSnapshots"=>$current,
 "previewRatingCount"=>$projection->ratingCount,"eligibleIds"=>$eligible,"allRemovedExcluded"=>true],
 "ownerGivenHistory"=>["visible"=>$given!==null,"id"=>$given["id"]??null,"reasonSha256"=>$given!==null?hash("sha256",(string)$given["reason"]):null,"error"=>$givenError],
 "entity"=>["primary"=>$primary->classification->canonical_name,"images"=>true,"linkedActive"=>0,"published"=>true],
 "otherThingRatingEventsTotal"=>$db::table("rating_events")->where("target_entity_id","!=",$id)->count()
],JSON_UNESCAPED_SLASHES|JSON_INVALID_UTF8_SUBSTITUTE)."\n";

RAT_CJ_POST_AUDIT_PHP
)"
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$php"
echo RATCJ_RATING_INDEPENDENT_POSTCHECK_SCRIPT_COMPLETED=true
