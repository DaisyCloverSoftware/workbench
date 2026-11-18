#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only, image-pinned CJ canonical DEV rating history with privacy-minimised creator fingerprints.
[[ $# -eq 0 ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
api='ghcr.io/daisycloversoftware/rum-api@sha256:a23e36323b55f3bce46d5b229146e2e7a9d70bdff9e16ea366380940e0629430'
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api" ]] || exit 78
php='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
$id="01m07d2f9fj05f4ypdbwp087qj";
$cj=App\Models\Entity::query()->with(["type","classificationAssignments.classification"])->findOrFail($id);
if($cj->canonical_name!=="CJ Investigates")throw new RuntimeException("Wrong Thing.");
$primary=$cj->classificationAssignments->first(fn($a)=>(bool)$a->is_primary && $a->valid_to===null);
$relationships=$db::table("entity_relationships")->where(fn($q)=>$q->where("source_entity_id",$id)->orWhere("target_entity_id",$id))->get(["id","valid_to"]);
$events=App\Models\RatingEvent::query()->with(["rater","scheme","context","reasonSelections","evidence","review","admissionState","admissionVotes","revisions"])
 ->where("target_entity_id",$id)->orderBy("submitted_at")->orderBy("id")->get();
if($events->count()===0||$events->count()>130)throw new RuntimeException("Unexpected rating inventory size.");
$ids=$events->pluck("id")->all();
$entries=App\Models\RatingJourneyEntry::query()->with("journey")->whereIn("rating_event_id",$ids)->get()->groupBy("rating_event_id");
$eventsById=[];
foreach($events as $event){
 $u=$event->rater;
 $m=is_array($event->metadata)?$event->metadata:[];
 $safeMetadata=[];
 foreach(["surface","ratingCategoryKey","structuredReasons","liveRatingFlow","legacyCanonicalUpgrade","ratingJourneyId","legacyStatus","legacyJudgeState"] as $key){
  if(array_key_exists($key,$m))$safeMetadata[$key]=$m[$key];
 }
 $jList=[];
 foreach($entries->get($event->id,collect()) as $je){
  $journey=$je->journey;
  $jList[]=["entryId"=>$je->id,"journeyId"=>$journey?->id,"journeyUserId"=>$journey?->user_id,
  "journeyRoot"=>$journey?->root_entity_id,"journeySurface"=>$journey?->origin_surface,"journeyState"=>$journey?->state,
  "journeyStarted"=>$journey?->started_at?->toAtomString(),"journeyEnded"=>$journey?->completed_at?->toAtomString(),
  "entryState"=>$je->state,"entrySequence"=>$je->sequence,"parentRatingEventId"=>$je->parent_rating_event_id,"relationshipId"=>$je->relationship_id];
 }
 $revisions=$event->revisions->map(fn($r)=>["id"=>$r->id,"type"=>$r->revision_type,"reason"=>$r->reason,"actorId"=>$r->changed_by_user_id,"createdAt"=>$r->occurred_at?->toAtomString()])->values()->all();
 $email=$u?->email!==null?mb_strtolower(trim($u->email)):"";
 $account=["id"=>$u?->id,"username"=>$u?->username,"founderNumber"=>$u?->founder_number,"accountStatus"=>$u?->status,
   "createdAt"=>$u?->created_at?->toAtomString(),"emailVerified"=>$u?->email_verified_at!==null,
   "emailSha256"=>$email!==""?hash("sha256",$email):null,"isExampleDomain"=>$email!=="" && str_ends_with($email,"@example.com")];
 $a=$event->admissionState;
 $eventsById[]=[
  "id"=>$event->id,"rater"=>$account,"value"=>(int)$event->value,"source"=>$event->source,
  "audience"=>$event->audience,"status"=>$event->status,"createdAt"=>$event->created_at?->toAtomString(),
  "submittedAt"=>$event->submitted_at?->toAtomString(),"updatedAt"=>$event->updated_at?->toAtomString(),
  "withdrawnAt"=>$event->withdrawn_at?->toAtomString(),"legacyId"=>$event->legacy_rating_id,
  "oneActivePerRater"=>$event->one_active_per_rater,"scheme"=>$event->scheme?->key,
  "contextKey"=>$event->context?->key,"contextName"=>$event->context?->name,
  "ratingMetadata"=>$safeMetadata,"ratingMetadataKeys"=>array_keys($m),"idempotencyPresent"=>$event->idempotency_key!==null,
  "reasons"=>$event->reasonSelections->map(fn($r)=>["key"=>$r->reason_key,"label"=>$r->reason_label,"status"=>$r->status])->values()->all(),
  "review"=>$event->review?["status"=>$event->review->status,"authorId"=>$event->review->author_account_id,"body"=>mb_substr($event->review->body,0,550)]:null,
  "evidence"=>$event->evidence->map(fn($e)=>["id"=>$e->id,"kind"=>$e->kind,"status"=>$e->status,"label"=>$e->label,"hasUrl"=>$e->url!==null])->values()->all(),
  "admission"=>$a?["state"=>$a->state,"keepVotes"=>$a->keep_votes,"sinBinVotes"=>$a->sin_bin_votes,"score"=>$a->score]:null,
  "admissionVotes"=>$event->admissionVotes->map(fn($v)=>["id"=>$v->id,"voter"=>$v->voter_id,"vote"=>$v->vote,"createdAt"=>$v->created_at?->toAtomString()])->values()->all(),
  "journeyEntries"=>$jList,"revisions"=>$revisions
 ];
}
$aggregateTypes=$db::table("outbox_events")->where("aggregate_type","rating_event")->whereIn("aggregate_id",$ids)->groupBy("event_type")->selectRaw("event_type,count(*) as total")->get();
$accounts=$events->groupBy("rater_account_id")->map(fn($g)=>["count"=>$g->count(),"first"=>$g->min("submitted_at"),"last"=>$g->max("submitted_at")]);
$summaries=["eventCount"=>$events->count(),"statusCounts"=>$events->countBy("status"),"audienceCounts"=>$events->countBy("audience"),"sourceCounts"=>$events->countBy("source"),
 "uniqueRaters"=>$accounts->count(),"perRater"=>$accounts->all(),
 "admissionCounts"=>$events->countBy(fn($e)=>$e->admissionState?->state??"missing"),"withVotes"=>$events->filter(fn($e)=>$e->admissionVotes->isNotEmpty())->count(),
 "totalVotes"=>$events->sum(fn($e)=>$e->admissionVotes->count()),"withJourney"=>$events->filter(fn($e)=>$entries->has($e->id))->count(),"eventOutboxCounts"=>$aggregateTypes];
echo "CJ_RATING_FULL_READONLY_AUDIT=".json_encode([
 "time"=>now()->toAtomString(),"entity"=>["id"=>$id,"name"=>$cj->canonical_name,"type"=>$cj->type?->key,"classification"=>$primary?->classification?->canonical_name,
 "publication"=>$cj->publication_state,"rateability"=>$cj->rateability,"visibility"=>$cj->visibility,"editPolicy"=>$cj->edit_policy],
 "relationsTotal"=>$relationships->count(),"relationsActive"=>$relationships->whereNull("valid_to")->count(),"summary"=>$summaries,"ratings"=>$eventsById
],JSON_UNESCAPED_SLASHES|JSON_INVALID_UTF8_SUBSTITUTE)."\n";
'
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$php"
echo CJ_RATING_FULL_READONLY_AUDIT_COMPLETE=true
