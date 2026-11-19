#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only existing public accepted rating discovery for REAL browser QA; no new DEV ratings or votes.
[[ $# -eq 0 ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == 'ghcr.io/daisycloversoftware/rum-api@sha256:a23e36323b55f3bce46d5b229146e2e7a9d70bdff9e16ea366380940e0629430' ]] || exit 78
php="$(cat <<'RATCJ_ACCEPTED_READONLY_PHP'

require "vendor/autoload.php";
$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
$cj="01m07d2f9fj05f4ypdbwp087qj";
$rows=$db::table("rating_events as e")
 ->join("rating_event_admission_states as a","a.rating_event_id","=","e.id")
 ->join("rating_schemes as s","s.id","=","e.rating_scheme_id")
 ->join("entities as t","t.id","=","e.target_entity_id")
 ->where("e.target_entity_id","!=",$cj)
 ->where("e.status","active")->where("e.audience","public")
 ->where("a.state","accepted")
 ->where("s.key","rum_rate_anything_general_v1")
 ->whereIn("e.source",["rate_anything","rateurmate_public_identity","canonical_rum"])
 ->where("t.visibility","public")->where("t.publication_state","published")
 ->where("t.rateability","open")->orderByDesc("e.submitted_at")
 ->limit(40)->get(["e.id as rating_id","e.value","e.target_entity_id","e.submitted_at","t.canonical_name as entity_name","a.keep_votes","a.sin_bin_votes","a.score"]);
$policy=app(App\Services\RatDiscoveryPolicy::class);
$read=app(App\Services\UniversalEntityReadService::class);
$eligible=[];
foreach($rows as $r){
 $entity=$read->findPublic((string)$r->target_entity_id);
 if(!$entity||!$policy->allows($entity))continue;
 $eligible[]=["ratingId"=>$r->rating_id,"entityId"=>$r->target_entity_id,
 "entityName"=>$r->entity_name,"value"=>$r->value,
 "keepVotes"=>$r->keep_votes,"sinBinVotes"=>$r->sin_bin_votes,
 "net"=>$r->score,"submittedAt"=>$r->submitted_at];
 if(count($eligible)>=8)break;
}
$states=$db::table("rating_events as e")
 ->join("rating_event_admission_states as a","a.rating_event_id","=","e.id")
 ->where("e.target_entity_id","!=",$cj)->where("e.status","active")
 ->where("e.audience","public")->groupBy("a.state")
 ->selectRaw("a.state,count(*) as n")->pluck("n","a.state");
echo "RATCJ_OTHER_PUBLIC_ACCEPTED_CANDIDATES=".json_encode(["acceptedExamples"=>$eligible,
 "otherThingAllActivePublicAdmissionStates"=>$states],JSON_UNESCAPED_SLASHES|JSON_INVALID_UTF8_SUBSTITUTE)."\n";

RATCJ_ACCEPTED_READONLY_PHP
)"
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- timeout 65s php -r "$php"
echo RATCJ_ACCEPTED_EXAMPLE_DISCOVERY_COMPLETE=true
