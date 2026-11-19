#!/usr/bin/env bash
set -Eeuo pipefail
# Create one disposable *non-CJ* isolated DEV QA Thing and accept rating only through five normal Judge votes.
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
php="$(cat <<'RATCJ_DISPOSABLE_ACCEPTED_QA_PHP'

require "vendor/autoload.php";
$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
$cj="01m07d2f9fj05f4ypdbwp087qj";
$keep="01m0mvrxv8c3s12q0w7gwmzt03";
$slug="rat-pr179-public-accepted-qa-20261119";
$run="ratcj-pr179-accepted-qa-20261119";
$name="RAT Public Accepted Rating QA 20261119";
$authorId="01m0rbemt8v5n05pm8yahz503c";
$voterIds=[
"01m0rd6ny7fbyed8350w19wwky",
"01m0rdftn1d01z2nanv23z4d3d",
"01m0rdq5yfe02nd78gts4amyzh",
"01m0rdrpvex3p7fkxkwwn30xb3",
"01m0rdw939q07hs3bvzkkm7tde"
];
if(count($voterIds)!==5||count(array_unique($voterIds))!==5||in_array($authorId,$voterIds,true))
 throw new RuntimeException("QA actor partition is invalid");
$out=$db::transaction(function()use($db,$cj,$keep,$slug,$run,$name,$authorId,$voterIds){
 $db::selectOne("SELECT pg_advisory_xact_lock(hashtext(?))",[$run]);
 $cjRatings=$db::table("rating_events")->where("target_entity_id",$cj)->get(["id","status"]);
 $cjAdmission=$db::table("rating_event_admission_states")->where("rating_event_id",$keep)->first(["state","keep_votes","sin_bin_votes","score"]);
 if($cjRatings->count()!==1||$cjRatings[0]->id!==$keep||$cjRatings[0]->status!=="active"||
    !$cjAdmission||$cjAdmission->state!=="needs_verdict"||$cjAdmission->keep_votes!==0||
    $cjAdmission->sin_bin_votes!==0||$cjAdmission->score!==0)
  throw new RuntimeException("CJ clean rating baseline drift");
 $existing=App\Models\Entity::query()->where("canonical_slug",$slug)->first();
 if($existing){
  $ratings=$db::table("rating_events")->where("target_entity_id",$existing->id)->get(["id","status"]);
  $state=$ratings->count()===1?$db::table("rating_event_admission_states")->where("rating_event_id",$ratings[0]->id)->first():null;
  if($existing->canonical_name!==$name||$existing->publication_state!=="published"||$ratings->count()!==1||
     $ratings[0]->status!=="active"||$state?->state!=="accepted"||$state?->keep_votes!==5||
     $state?->sin_bin_votes!==0||$state?->score!==5)
   throw new RuntimeException("Existing QA fixture is not the already-accepted exact fixture");
  return ["status"=>"already_staged","thingId"=>$existing->id,"ratingId"=>$ratings[0]->id,"accepted"=>1];
 }
 $actors=App\Models\User::query()->whereIn("id",array_merge([$authorId],$voterIds))->get()->keyBy("id");
 if($actors->count()!==6)throw new RuntimeException("Missing dedicated synthetic QA accounts");
 foreach(array_merge([$authorId],$voterIds) as $actorId){
  $u=$actors[$actorId];$email=mb_strtolower(trim((string)$u->email));
  if($u->status!=="active"||!str_ends_with($email,"@example.com")||
      !str_starts_with((string)$u->username,"rumcheck")||
      !$u->hasVerifiedEmail()||!$u->hasCurrentAdultAssurance()||
      $u->created_at?->isAfter(now()->subHours(24)))
   throw new RuntimeException("Synthetic QA actor eligibility mismatch");
 }
 $judgeEligibility=app(App\Services\JudgeEligibility::class);
 $features=app(App\Services\FeatureAccessService::class);
 foreach($voterIds as $voterId){
  $voter=$actors[$voterId];
  $judgeEligibility->assertCanBrowse($voter);
  if($judgeEligibility->accountTooNew($voter))throw new RuntimeException("QA voter too new");
  $features->assertAllowed($voter,"voting");
 }
 $type=App\Models\EntityType::query()->where("key","product")->firstOrFail();
 $entity=App\Models\Entity::query()->create([
  "entity_type_id"=>$type->id,"canonical_name"=>$name,
  "normalised_name"=>mb_strtolower($name),"canonical_slug"=>$slug,
  "description"=>"Disposable isolated DEV acceptance example. Not a real product.",
  "origin"=>"system_derived","visibility"=>"public","publication_state"=>"published",
  "status"=>"active","rateability"=>"open","reviewability"=>"closed",
  "sensitivity"=>"normal","verification_state"=>"unverified"
 ]);
 $scheme=App\Models\RatingScheme::query()->where("key",App\Services\RateAnythingRatingService::GENERAL_SCHEME_KEY)->where("status","active")->firstOrFail();
 $event=app(App\Services\UniversalRatingEventService::class)->create(
  rater:$actors[$authorId],target:$entity,scheme:$scheme,value:4,
  reviewBody:"Isolated DEV accepted-rating card verification only.",
  audience:"public",source:"rate_anything",
  metadata:["isolatedDevQa"=>true,"acceptedCardAcceptanceRun"=>$run]
 );
 if($event->status!=="active"||$event->admissionState()->firstOrFail()->state!=="needs_verdict")
   throw new RuntimeException("QA rating did not enter canonical Judge pending state");
 $judge=app(App\Services\CanonicalJudgeService::class);
 foreach($voterIds as $voterId){
  $judge->vote($actors[$voterId],$event,1);
 }
 $state=$event->admissionState()->firstOrFail();
 $votes=$db::table("rating_event_admission_votes")->where("rating_event_id",$event->id)->count();
 if($state->state!=="accepted"||(int)$state->keep_votes!==5||
    (int)$state->sin_bin_votes!==0||(int)$state->score!==5||$votes!==5)
  throw new RuntimeException("Real canonical Judge five votes did not accept QA rating");
 $algorithm=App\Models\ReputationAlgorithmVersion::query()->where("key","rum_rate_anything_reputation_v2")->where("status","active")->firstOrFail();
 $snapshot=app(App\Services\Reputation\ReputationSnapshotService::class)->calculateAndStore($entity,$scheme,$algorithm,null);
 if($snapshot->rating_count!==1)throw new RuntimeException("Accepted QA rating not counted by canonical reputation");
 $cards=app(App\Services\RatPublicRatingsService::class)->forThing($entity->id);
 if(($cards["counts"]["accepted"]??null)!==1||($cards["counts"]["awaitingVerdict"]??null)!==0||
    count($cards["items"])!==1||$cards["items"][0]["id"]!==$event->id||
    ($cards["items"][0]["verdict"]["keepVotes"]??null)!==5)
  throw new RuntimeException("New public card API failed accepted QA fixture");
 if($db::table("rating_events")->where("target_entity_id",$cj)->count()!==1||
    $db::table("rating_event_admission_votes")->where("rating_event_id",$keep)->exists())
  throw new RuntimeException("CJ changed during scoped accepted QA");
 return ["status"=>"staged","thingId"=>$entity->id,"slug"=>$entity->canonical_slug,
 "ratingId"=>$event->id,"value"=>$event->value,"admission"=>$state->state,
 "keepVotes"=>5,"sinBinVotes"=>0,"net"=>5,
 "publicAcceptedCards"=>1,"snapshotRatingCount"=>$snapshot->rating_count,
 "temporaryQaOnly"=>true,"cjUntouched"=>true];
},3);
echo "RATCJ_DISPOSABLE_ACCEPTED_QA_STAGED=".json_encode($out,JSON_UNESCAPED_SLASHES)."\n";

RATCJ_DISPOSABLE_ACCEPTED_QA_PHP
)"
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- timeout 125s php -r "$php"
echo RATCJ_DISPOSABLE_ACCEPTED_QA_STAGING_COMPLETE=true
