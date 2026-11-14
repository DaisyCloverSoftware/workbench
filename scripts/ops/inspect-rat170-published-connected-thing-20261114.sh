#!/usr/bin/env bash
set -euo pipefail
# Exact read-only published-Thing, initial-rating, and returning-connection provenance in isolated DEV.
[[ $# -eq 5 ]] || exit 64
for x in "$@"; do [[ "$x" =~ ^[a-z0-9]{26}$ ]] || exit 64; done
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
[[ "$(kctl -n rum-dev-isolated get ingress -o jsonpath='{.items[0].spec.rules[0].host}')" == 'dev-rum.daisycloversoftware.uk' ]] || exit 78
php='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";
$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
[$root,$thing,$journeyId,$ratingId,$edgeId]=array_slice($argv,1);
$db=Illuminate\Support\Facades\DB::class;
$e=$db::table("entities")->where("id",$thing)->first();
$j=$db::table("rating_journeys")->where("id",$journeyId)->first();
$r=$db::table("rating_events")->where("id",$ratingId)->first();
$edge=$db::table("entity_relationships")->where("id",$edgeId)->whereNull("valid_to")->first();
$creator=$e?$db::table("users")->where("id",$e->created_by_user_id)->first(["username"]):null;
$targetCount=$db::table("rating_events")->where("target_entity_id",$thing)->count();
$pairCount=$db::table("entity_relationships")->whereNull("valid_to")
 ->where(fn($q)=>$q->where(fn($a)=>$a->where("source_entity_id",$root)->where("target_entity_id",$thing))
 ->orWhere(fn($a)=>$a->where("source_entity_id",$thing)->where("target_entity_id",$root)))->count();
$previous=$db::table("entities")->where("id","01m7dbsjhn6a9j373va2yz42yx")->first();
$result=[
 "thingName"=>$e?->canonical_name,
 "thingPublished"=>(bool)($e&&$e->publication_state==="published"&&$e->visibility==="public"),
 "createdBySyntheticMember"=>(bool)($creator&&str_starts_with($creator->username,"raticon")),
 "creatorUsername"=>$creator?->username,
 "journeyOwnedByCreator"=>(bool)($j&&$e&&$j->user_id===$e->created_by_user_id),
 "journeyRootCorrect"=>(bool)($j&&$j->root_entity_id===$thing),
 "journeyState"=>$j?->state,
 "initialRatingPresent"=>(bool)($r&&$r->target_entity_id===$thing),
 "newThingRatingEventCount"=>$targetCount,
 "edgeSourceOriginal"=>(bool)($edge&&$edge->source_entity_id===$root),
 "edgeTargetNew"=>(bool)($edge&&$edge->target_entity_id===$thing),
 "edgeContributorIsCreator"=>(bool)($edge&&$e&&$edge->created_by_user_id===$e->created_by_user_id),
 "edgeProvenance"=>$edge?->provenance,
 "edgeVerification"=>$edge?->verification_state,
 "edgeVisibility"=>$edge?->visibility,
 "activePairEdges"=>$pairCount,
 "originalPendingStillPending"=>(bool)($previous&&$previous->publication_state==="pending"&&$previous->visibility==="private"),
 "originalPendingRatingCount"=>$db::table("rating_events")->where("target_entity_id","01m7dbsjhn6a9j373va2yz42yx")->count(),
 "sqlReadOnly"=>true,
];
echo "RAT170_MISSING_THING_CANONICAL_SQL=".json_encode($result,JSON_THROW_ON_ERROR)."\n";
if(!($result["thingPublished"]&&$result["createdBySyntheticMember"]&&$result["journeyOwnedByCreator"]
 &&$result["journeyRootCorrect"]&&$result["initialRatingPresent"]&&$targetCount===1
 &&$result["edgeSourceOriginal"]&&$result["edgeTargetNew"]&&$result["edgeContributorIsCreator"]
 &&$result["edgeProvenance"]==="user_submitted"&&$result["edgeVerification"]==="unverified"
 &&$result["edgeVisibility"]==="public"&&$pairCount===1&&$result["originalPendingStillPending"]
 &&$result["originalPendingRatingCount"]===0))exit(2);
';
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$php" -- "$@"
echo 'RAT170_MISSING_THING_SQL_PASS=true'
