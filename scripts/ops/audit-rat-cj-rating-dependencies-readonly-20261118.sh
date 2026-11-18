#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only dependency and author-vs-synthetic evidence inventory for CJ ratings.
[[ $# -eq 0 ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
api='ghcr.io/daisycloversoftware/rum-api@sha256:a23e36323b55f3bce46d5b229146e2e7a9d70bdff9e16ea366380940e0629430'
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api" ]] || exit 78
php='
require "vendor/autoload.php";$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;$schema=Illuminate\Support\Facades\Schema::class;
$id="01m07d2f9fj05f4ypdbwp087qj";$owner="01kzgn6a7kswt3jqjb1sp8d4x0";
$events=App\Models\RatingEvent::query()->where("target_entity_id",$id)->orderBy("submitted_at")->get(["id","rater_account_id","status","source","submitted_at"]);
$ids=$events->pluck("id")->all();
if($events->count()!==54)throw new RuntimeException("Changed event count; no business operation authorised");
$owners=$events->where("rater_account_id",$owner);
$qa=$events->where("rater_account_id","!=",$owner);
$qaIds=$qa->pluck("id")->all();
$users=App\Models\User::query()->whereIn("id",$qa->pluck("rater_account_id")->all())->get(["id","username","email","created_at","founder_number"])->keyBy("id");
foreach($qa as $e){$u=$users->get($e->rater_account_id);
 if(!$u||!str_ends_with(mb_strtolower(trim((string)$u->email)),"@example.com")||!preg_match("/^(rumcheck|rumowner|rumrate|rumcj|rumagain|ratowner)[0-9]/",$u->username)||$e->status!=="active")throw new RuntimeException("Unexpected non-test account");
}
$votes=App\Models\RatingEventAdmissionVote::query()->whereIn("rating_event_id",$ids)->get(["id","rating_event_id","voter_id","vote","created_at"]);
$voteUsers=App\Models\User::query()->whereIn("id",$votes->pluck("voter_id")->all())->get(["id","username","email","founder_number"])->keyBy("id");
$voteRows=$votes->map(function($v)use($owner,$voteUsers,$qaIds){
 $u=$voteUsers->get($v->voter_id);
 return ["id"=>$v->id,"rating"=>$v->rating_event_id,"ratingQA"=>in_array($v->rating_event_id,$qaIds,true),"voterId"=>$v->voter_id,
 "voterUsername"=>$u?->username,"voterIsExample"=>$u!==null&&str_ends_with(mb_strtolower((string)$u->email),"@example.com"),
 "voterFounder"=>$u?->founder_number,"vote"=>$v->vote,"created"=>$v->created_at?->toAtomString()];
})->values()->all();
$reasonVotes=$schema::hasTable("rating_reason_votes")?$db::table("rating_reason_votes")->whereIn("rating_event_id",$ids)->get(["id","rating_event_id","voter_id","vote","created_at"]):collect();
$reasonRows=$reasonVotes->map(function($v)use($owner,$qaIds){return ["id"=>$v->id,"rating"=>$v->rating_event_id,"ratingQA"=>in_array($v->rating_event_id,$qaIds,true),"voterId"=>$v->voter_id,"vote"=>$v->vote,"created"=>$v->created_at];})->values()->all();
$states=$db::table("rating_event_admission_states")->whereIn("rating_event_id",$ids)->get(["rating_event_id","state","keep_votes","sin_bin_votes","score"]);
$byState=$states->groupBy("state")->map(fn($v)=>$v->count());
$snapshots=$schema::hasTable("entity_reputation_snapshots")?$db::table("entity_reputation_snapshots")->where("entity_id",$id)->orderByDesc("calculated_at")->limit(25)->get(["id","rating_count","is_current","calculated_at","algorithm_version_id","rating_scheme_id"]):collect();
$moderation=$schema::hasTable("moderation_cases")?$db::table("moderation_cases")->where("object_type","rating_event")->whereIn("object_id",$ids)->get(["id","object_id","status","queue"]):collect();
$related=$db::table("rating_journey_entries")->whereIn("rating_event_id",$ids)->orWhereIn("parent_rating_event_id",$ids)->get(["id","rating_event_id","parent_rating_event_id","relationship_id"]);
echo "CJ_RATING_DEPENDENCIES=".json_encode(["time"=>now()->toAtomString(),"ratingCount"=>$events->count(),"ownerHistory"=>$owners->count(),
 "ownerActive"=>$owners->where("status","active")->count(),"qaConfirmed"=>$qa->count(),"qaActive"=>$qa->where("status","active")->count(),
 "qaAllGeneratedAndExample"=>true,"admissionStateCounts"=>$byState,"admissionVotes"=>$voteRows,
 "reasonVotes"=>$reasonRows,"currentSnapshots"=>$snapshots,"moderationCases"=>$moderation,
 "journeyEntryCounts"=>["all"=>$related->count(),"qa"=>$related->whereIn("rating_event_id",$qaIds)->count()],
 "otherThingRatingEventsTotal"=>$db::table("rating_events")->where("target_entity_id","!=",$id)->count()],JSON_UNESCAPED_SLASHES|JSON_INVALID_UTF8_SUBSTITUTE)."\n";
'
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$php"
echo CJ_RATING_DEPENDENCIES_AUDIT_COMPLETE=true
