#!/usr/bin/env bash
set -euo pipefail
# Canonical DEV read-only diagnosis of existing owner's rating admission and
# Judge queue *data* eligibility. Never read personal identifiers or write state.
[[ $# -eq 0 ]] || exit 64
kctl() { if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
hosts="$(kctl -n rum-dev-isolated get ingress -o jsonpath='{range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)"
[[ "$hosts" == 'dev-rum.daisycloversoftware.uk' ]] || exit 78
probe='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";
$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
$eid="01m7bswfh0a3evf271ye4taaf0";
$e=$db::table("entities")->where("id",$eid)->whereNull("deleted_at")->first(["canonical_name","publication_state","visibility","rateability","status"]);
$rows=$db::table("rating_events as r")
 ->join("rating_schemes as s","s.id","=","r.rating_scheme_id")
 ->leftJoin("rating_event_admission_states as a","a.rating_event_id","=","r.id")
 ->where("r.target_entity_id",$eid)
 ->get(["r.status","r.audience","r.source","s.key as scheme","a.state as admission","a.score","r.rating_context_id"]);
$list=[];
foreach($rows as $r) {
  $ordinary=in_array($r->source,["rate_anything","rateurmate_public_identity"],true)
   && $r->scheme==="rum_rate_anything_general_v1";
  $eligible=$r->status==="active"&&in_array($r->audience,["public","mates"],true)
   && $r->admission==="needs_verdict"&&$ordinary;
  $list[]=["eventStatus"=>$r->status,"audience"=>$r->audience,"source"=>$r->source,
   "scheme"=>$r->scheme,"admissionState"=>$r->admission,"admissionScore"=>$r->score,
   "ordinaryQueueSourceAndScheme"=>$ordinary,
   "needsQueueDataEligibleWithoutViewerScope"=>$eligible,
   "hasRatingContext"=>$r->rating_context_id!==null];
}
echo "RAT_JUDGE_READ_ONLY=".json_encode([
 "canonicalEntityPresent"=>$e!==null,
 "entityName"=>$e?->canonical_name,
 "entityPublication"=>$e?->publication_state,
 "entityVisibility"=>$e?->visibility,
 "ratingCount"=>$rows->count(),
 "ratingRows"=>$list,
 "viewerEligibilityAndScopeNotAssumed"=>true,
 "noRatingOrAdmissionMutation"=>true
],JSON_THROW_ON_ERROR)."\n";
'
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$probe"
echo RAT_JUDGE_QUERY_READ_ONLY=true
