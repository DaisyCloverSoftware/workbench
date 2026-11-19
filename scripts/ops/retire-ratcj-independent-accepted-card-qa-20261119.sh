#!/usr/bin/env bash
set -Eeuo pipefail
# Tear down isolated *non-CJ* acceptance fixture through canonical Judge vote removals and owner withdrawal; hide from public.
[[ $# -eq 2 && "$1" =~ ^[0-9a-f]{40}$ ]] || exit 64
src="$1";api="$2"
[[ "$api" =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api" ]] || exit 78
health="$(curl -fsS --retry 4 --max-time 20 https://dev-rum.daisycloversoftware.uk/api/v1/health)"
python3 - "$health" "$src" <<'PY'
import json,sys
h=json.loads(sys.argv[1]);assert h.get('status')=='ok' and sys.argv[2] in str(h.get('version',''))
PY
php="$(cat <<'RATCJ_QA_RETIRE_PHP'

require "vendor/autoload.php";$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
$cj="01m07d2f9fj05f4ypdbwp087qj";
$keep="01m0mvrxv8c3s12q0w7gwmzt03";
$slug="rat-pr179-public-accepted-qa-20261119";
$run="ratcj-pr179-accepted-qa-20261119";
$authorId="01m0rbemt8v5n05pm8yahz503c";
$voterIds=["01m0rd6ny7fbyed8350w19wwky","01m0rdftn1d01z2nanv23z4d3d",
 "01m0rdq5yfe02nd78gts4amyzh","01m0rdrpvex3p7fkxkwwn30xb3",
 "01m0rdw939q07hs3bvzkkm7tde"];
$result=$db::transaction(function()use($db,$cj,$keep,$slug,$run,$authorId,$voterIds){
 $db::selectOne("SELECT pg_advisory_xact_lock(hashtext(?))",[$run]);
 $entity=App\Models\Entity::query()->where("canonical_slug",$slug)->lockForUpdate()->firstOrFail();
 if($entity->canonical_name!=="RAT Public Accepted Rating QA 20261119"||
    $entity->origin!=="system_derived")
   throw new RuntimeException("Not exact separately-scoped QA fixture");
 $events=App\Models\RatingEvent::query()->where("target_entity_id",$entity->id)->get();
 if($events->count()!==1||$events[0]->rater_account_id!==$authorId||
    $events[0]->source!=="rate_anything"||
    ($events[0]->metadata["acceptedCardAcceptanceRun"]??null)!==$run)
   throw new RuntimeException("Unexpected QA rating set or provenance");
 $event=$events[0];
 if($event->status==="withdrawn"&&$entity->publication_state==="hidden"){
  if($db::table("rating_event_admission_votes")->where("rating_event_id",$event->id)->exists()||
     $db::table("entity_reputation_snapshots")->where("entity_id",$entity->id)->exists())
   throw new RuntimeException("Previously retired QA fixture still has votes or reputation");
  return ["status"=>"already_retired","thingId"=>$entity->id,"ratingId"=>$event->id];
 }
 $state=$event->admissionState()->firstOrFail();
 if($entity->publication_state!=="published"||$entity->visibility!=="public"||
    $event->status!=="active"||$state->state!=="accepted"||(int)$state->keep_votes!==5||
    (int)$state->sin_bin_votes!==0||(int)$state->score!==5||
    $db::table("rating_event_admission_votes")->where("rating_event_id",$event->id)->count()!==5)
   throw new RuntimeException("Accepted QA baseline changed");
 $voters=App\Models\User::query()->whereIn("id",$voterIds)->get()->keyBy("id");
 if($voters->count()!==5)throw new RuntimeException("QA voter identity set changed");
 $judge=app(App\Services\CanonicalJudgeService::class);
 foreach($voterIds as $id)$judge->removeVote($voters[$id],$event);
 $state=$event->admissionState()->firstOrFail();
 if($state->state!=="needs_verdict"||(int)$state->score!==0||
    $db::table("rating_event_admission_votes")->where("rating_event_id",$event->id)->count()!==0)
   throw new RuntimeException("Canonical QA votes not reset");
 $author=App\Models\User::query()->findOrFail($authorId);
 app(App\Services\UniversalRatingEventService::class)->withdraw($event,$author,"Completed disposable isolated DEV accepted-card verification");
 $entity->forceFill(["visibility"=>"members","publication_state"=>"hidden","rateability"=>"closed"])->save();
 $snapshots=App\Models\EntityReputationSnapshot::query()->where("entity_id",$entity->id)->get();
 foreach($snapshots as $snapshot)$snapshot->delete();
 if($db::table("rating_events")->where("id",$event->id)->value("status")!=="withdrawn"||
    $db::table("rating_event_admission_states")->where("rating_event_id",$event->id)->value("state")!=="removed"||
    $db::table("rating_event_admission_votes")->where("rating_event_id",$event->id)->exists()||
    $db::table("entity_reputation_snapshots")->where("entity_id",$entity->id)->exists()||
    $entity->publication_state!=="hidden")
   throw new RuntimeException("QA fixture retirement postcondition failed");
 $cjEvents=$db::table("rating_events")->where("target_entity_id",$cj)->get(["id","status"]);
 if($cjEvents->count()!==1||$cjEvents[0]->id!==$keep||
    $db::table("rating_event_admission_votes")->where("rating_event_id",$keep)->exists())
   throw new RuntimeException("CJ baseline changed while retiring separate QA");
 return ["status"=>"retired","thingId"=>$entity->id,"ratingId"=>$event->id,
  "votesRemovedThroughJudge"=>5,"newPublicationState"=>"hidden","newRatingStatus"=>"withdrawn",
  "qaSnapshotsRemoved"=>$snapshots->count(),"cjUntouched"=>true];
},3);
echo "RATCJ_DISPOSABLE_ACCEPTED_QA_RETIRED=".json_encode($result,JSON_UNESCAPED_SLASHES)."\n";

RATCJ_QA_RETIRE_PHP
)"
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- timeout 125s php -r "$php"
echo RATCJ_DISPOSABLE_QA_RETIREMENT_COMPLETE=true
