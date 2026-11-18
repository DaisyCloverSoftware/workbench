#!/usr/bin/env bash
set -Eeuo pipefail
# Exact-image, scope-locked CJ-only canonical reputation service refresh following 1-owner rating cleanup.
[[ $# -eq 0 ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == 'ghcr.io/daisycloversoftware/rum-api@sha256:a23e36323b55f3bce46d5b229146e2e7a9d70bdff9e16ea366380940e0629430' ]] || exit 78
php="$(cat <<'RAT_CJ_REPUTATION_REFRESH_PHP'

require "vendor/autoload.php";
$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;$id="01m07d2f9fj05f4ypdbwp087qj";$keep="01m0mvrxv8c3s12q0w7gwmzt03";
$operation="ratcj-20261118-owner-original-rating-cleanup-r1";
$done=$db::transaction(function()use($db,$id,$keep,$operation){
 $db::statement("SET TRANSACTION ISOLATION LEVEL SERIALIZABLE");
 $entity=App\Models\Entity::query()->lockForUpdate()->findOrFail($id);
 if($entity->canonical_name!=="CJ Investigates"||$entity->visibility!=="public"||$entity->rateability!=="open"||
    $entity->publication_state!=="published")throw new RuntimeException("CJ source changed");
 $ratings=App\Models\RatingEvent::query()->where("target_entity_id",$id)->lockForUpdate()->get();
 $retained=$ratings->firstWhere("status","active");
 if($ratings->count()!==54||$ratings->where("status","active")->count()!==1||
   $retained?->id!==$keep||$retained->source!=="canonical_rum"||
   $ratings->where("status","removed")->count()!==47||$ratings->where("status","withdrawn")->count()!==6||
   $db::table("rating_event_admission_states")->where("rating_event_id",$keep)->value("state")!=="not_eligible")
  throw new RuntimeException("Canonical cleanup state has changed");
 $scheme=App\Models\RatingScheme::query()->where("key","rum_rate_anything_general_v1")->where("status","active")->firstOrFail();
 $algorithm=App\Models\ReputationAlgorithmVersion::query()->where("key","rum_rate_anything_reputation_v2")->where("status","active")->firstOrFail();
 if(!is_array($algorithm->configuration)||($algorithm->configuration["public"]??false)!==true)throw new RuntimeException("Public v2 algorithm unavailable");
 $projection=app(App\Services\Reputation\ReputationCalculationService::class)->calculate($entity,$scheme,$algorithm,null);
 $included=collect($projection->weightAssessments)->filter(fn($a)=>$a->eligible&&$a->effectiveWeight>0)->pluck("ratingEventId")->values()->all();
 if($included!==[$keep]||$projection->ratingCount!==1)throw new RuntimeException("Not exactly the protected original in canonical calculator");
 $pre=$db::table("entity_reputation_snapshots")->where("entity_id","!=",$id)->orderBy("id")->get()->toJson();
 $otherRatings=$db::table("rating_events")->where("target_entity_id","!=",$id)->orderBy("id")->get()->toJson();
 $eventBefore=json_encode($db::table("rating_events")->where("id",$keep)->first(),JSON_UNESCAPED_SLASHES);
 $voteBefore=$db::table("rating_event_admission_votes")->whereIn("rating_event_id",$ratings->pluck("id")->all())->orderBy("id")->get()->toJson();
 $existing=App\Models\EntityReputationSnapshot::query()->where("entity_id",$id)
   ->where("rating_scheme_id",$scheme->id)->where("algorithm_version_id",$algorithm->id)
   ->whereNull("rating_context_id")->where("is_current",true)->first();
 if($existing && $existing->rating_count===1 &&
     $existing->calculated_at?->greaterThanOrEqualTo(\Carbon\CarbonImmutable::parse("2026-11-18T22:28:00+00:00"))){
   $weights=$db::table("rating_weight_snapshots")->where("reputation_snapshot_id",$existing->id)->where("eligible",true)->pluck("rating_event_id")->all();
   if($weights===[$keep]){
     echo "CJ_REPUTATION_ALREADY_CURRENT=".json_encode(["snapshotId"=>$existing->id,"ratingCount"=>1,"includedIds"=>$weights],JSON_UNESCAPED_SLASHES)."\n";
     return ["status"=>"already_current","snapshotId"=>$existing->id,"ratingCount"=>1];
   }
 }
 $snapshot=app(App\Services\Reputation\ReputationSnapshotService::class)->calculateAndStore($entity,$scheme,$algorithm,null);
 $eligibleWeights=$db::table("rating_weight_snapshots")->where("reputation_snapshot_id",$snapshot->id)->where("eligible",true)->get(["rating_event_id","effective_weight"]);
 if($snapshot->rating_count!==1||$eligibleWeights->count()!==1||$eligibleWeights[0]->rating_event_id!==$keep||
    $db::table("rating_weight_snapshots")->where("reputation_snapshot_id",$snapshot->id)->count()!==54)
  throw new RuntimeException("Canonical projection did not isolate the protected original");
 if($db::table("entity_reputation_snapshots")->where("entity_id",$id)->where("is_current",true)->count()!==1||
    $db::table("entity_reputation_snapshots")->where("entity_id","!=",$id)->orderBy("id")->get()->toJson()!==$pre)
  throw new RuntimeException("Unrelated snapshot or current-snapshot postcondition failed");
 if($db::table("rating_events")->where("target_entity_id","!=",$id)->orderBy("id")->get()->toJson()!==$otherRatings||
    json_encode($db::table("rating_events")->where("id",$keep)->first(),JSON_UNESCAPED_SLASHES)!==$eventBefore||
    $db::table("rating_event_admission_votes")->whereIn("rating_event_id",$ratings->pluck("id")->all())->orderBy("id")->get()->toJson()!==$voteBefore)
  throw new RuntimeException("Rating evidence changed during projection");
 App\Models\OutboxEvent::query()->create([
  "aggregate_type"=>"entity_reputation_snapshot","aggregate_id"=>$snapshot->id,
  "event_type"=>"reputation.updated.v1",
  "payload"=>["entityId"=>$id,"ratingSchemeKey"=>$scheme->key,"ratingContextKey"=>null,"algorithmKey"=>$algorithm->key,
    "score"=>(int)round((float)$snapshot->score),"confidenceLabel"=>$snapshot->confidence_label,
    "ratingCount"=>(int)$snapshot->rating_count,"calculatedAt"=>$snapshot->calculated_at?->toAtomString(),
    "maintenanceOperationId"=>$operation,"manualScoreEdit"=>false],
  "occurred_at"=>now(),"attempts"=>0
 ]);
 return ["status"=>"projected","snapshotId"=>$snapshot->id,"ratingCount"=>(int)$snapshot->rating_count,
  "includedIds"=>[$keep],"otherEventsExcluded"=>53,"noManualScoreEdit"=>true,
  "selectedVerdictState"=>"not_eligible","publicScoreComputedByCanonicalAlgorithm"=>(int)round((float)$snapshot->score)];
},3);
echo "CJ_REPUTATION_SCOPED_REFRESH=".json_encode($done,JSON_UNESCAPED_SLASHES)."\n";

RAT_CJ_REPUTATION_REFRESH_PHP
)"
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$php"
echo CJ_REPUTATION_SCOPED_REFRESH_SCRIPT_COMPLETED=true
