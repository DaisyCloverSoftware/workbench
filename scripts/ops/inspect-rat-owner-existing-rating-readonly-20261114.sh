#!/usr/bin/env bash
set -euo pipefail
# READ ONLY: no mutations to ratings, admission states, thresholds, snapshots,
# entities, claim controls or publication. Canonical isolated DEV namespace only.
[[ $# -eq 0 ]] || exit 64
kctl() { if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
host="$(kctl -n rum-dev-isolated get ingress -o jsonpath='{range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)"
[[ "$host" == 'dev-rum.daisycloversoftware.uk' ]] || { echo 'canonical DEV host changed' >&2; exit 78; }
probe='
require "vendor/autoload.php";
$app = require "bootstrap/app.php";
$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db = Illuminate\Support\Facades\DB::class;
$entities=$db::table("entities")->where("canonical_name","The Professional Idiots")->whereNull("deleted_at")->get(["id","canonical_name","publication_state","visibility","status"]);
$out=[];
foreach($entities as $e){
 $rows=$db::table("rating_events as r")->leftJoin("rating_event_admission_states as a","r.id","=","a.rating_event_id")
 ->where("r.target_entity_id",$e->id)
 ->get(["r.status","r.audience","a.state as admission_state","a.keep_votes","a.sin_bin_votes","a.score"]);
 $states=[];
 foreach($rows as $r){
  $key=($r->status ?: "unknown")."/".($r->audience ?: "unknown")."/".($r->admission_state ?: "unrecorded");
  $states[$key]=($states[$key] ?? 0)+1;
 }
 $out[]=[
  "canonical_name"=>$e->canonical_name,
  "publication_state"=>$e->publication_state,
  "visibility"=>$e->visibility,
  "status"=>$e->status,
  "rating_events_count"=>$rows->count(),
  "rating_status_audience_admission_counts"=>$states,
 ];
}
echo "RAT_OWNER_RATING_READONLY=".json_encode([
  "namespace"=>"rum-dev-isolated",
  "matching_public_name_records"=>count($out),
  "records"=>$out,
  "rating_events_unchanged"=>true,
  "admission_states_unchanged"=>true,
],JSON_THROW_ON_ERROR)."\n";
'
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$probe"
echo 'RAT_RATING_READ_ONLY_NO_STATUS_MUTATION=true'
