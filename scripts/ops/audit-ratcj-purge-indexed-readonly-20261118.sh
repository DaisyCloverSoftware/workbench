#!/usr/bin/env bash
set -Eeuo pipefail
# Bounded read-only exact-id FK audit of 53 CJ DEV rating evidence rows (no historical payload scan).
[[ $# -eq 0 ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == 'ghcr.io/daisycloversoftware/rum-api@sha256:a23e36323b55f3bce46d5b229146e2e7a9d70bdff9e16ea366380940e0629430' ]] || exit 78
php="$(cat <<'RATCJ_FAST_PREFLIGHT_PHP'

require "vendor/autoload.php";$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;$schema=Illuminate\Support\Facades\Schema::class;
$cj="01m07d2f9fj05f4ypdbwp087qj";$keep="01m0mvrxv8c3s12q0w7gwmzt03";
$events=$db::table("rating_events")->where("target_entity_id",$cj)->get(["id","status","rater_account_id","value","source","submitted_at","legacy_rating_id"]);
$purge=$events->where("id","!=",$keep)->pluck("id")->all();
if($events->count()!==54||count($purge)!==53||$events->where("id",$keep)->where("status","active")->count()!==1||
 $events->where("id","!=",$keep)->where("status","removed")->count()!==47||
 $events->where("id","!=",$keep)->where("status","withdrawn")->count()!==6)throw new RuntimeException("Unexpected frozen CJ rating set");
$ids=$events->pluck("id")->all();
$fk=$db::select("SELECT c.relname as t,a.attname as col,co.conname as constraint_name,co.confdeltype as action
 FROM pg_constraint co JOIN pg_class c ON c.oid=co.conrelid
 JOIN pg_attribute a ON a.attrelid=c.oid AND a.attnum=co.conkey[1]
 WHERE co.contype='f' AND co.confrelid=to_regclass('rating_events') ORDER BY c.relname");
$refs=[];
foreach($fk as $row){
 $table=$row->t;$column=$row->col;
 $n=$db::table($table)->whereIn($column,$purge)->count();
 $refs[]=["table"=>$table,"col"=>$column,"onDelete"=>$row->action,"count"=>$n];
}
$known=["rating_journey_entries"=>["rating_event_id","parent_rating_event_id"],
 "profile_help_me_out_pins"=>["rating_id"],"moderation_cases"=>["object_id"],
 "moderation_actions"=>["target_id"],"rating_weight_snapshots"=>["rating_event_id"],"rating_reason_votes"=>["rating_event_id"]];
$extra=[];
foreach($known as $table=>$columns){
 if(!$schema::hasTable($table))continue;
 foreach($columns as $column)if($schema::hasColumn($table,$column))$extra[$table.".".$column]=$db::table($table)->whereIn($column,$purge)->count();
}
$journey=$db::table("rating_journey_entries")->whereIn("rating_event_id",$purge)->get(["id","rating_journey_id","entity_id"]);
$otherEntryCount=$journey->where("entity_id","!=",$cj)->count();
$otherRatingCount=$db::table("rating_events")->where("target_entity_id","!=",$cj)->count();
$outbox=$db::table("outbox_events")->where("aggregate_type","rating_event")->whereIn("aggregate_id",$purge)->get(["id","event_type","aggregate_id"]);
$outboxCounts=$outbox->groupBy("event_type")->map(fn($v)=>$v->count());
$receipts=$db::table("reputation_event_receipts")->whereIn("outbox_event_id",$outbox->pluck("id")->all())->count();
echo "RATCJ_PURGE_FAST_PREFLIGHT=".json_encode([
 "cj"=>$cj,"keep"=>$keep,"before"=>["events"=>$events->count(),"active"=>1,"toDelete"=>count($purge),
 "removed"=>47,"withdrawn"=>6,"otherEntityRatingEvents"=>$otherRatingCount],
 "deleteIds"=>$purge,"foreignKeys"=>$refs,"extra"=>$extra,
 "journey"=>["direct"=>$journey->count(),"otherEntityEntries"=>$otherEntryCount],
 "outbox"=>["ratingEventRecords"=>$outbox->count(),"eventTypes"=>$outboxCounts,"reputationReceipts"=>$receipts],
 "snapshotRows"=>$db::table("entity_reputation_snapshots")->where("entity_id",$cj)->get(["id","is_current","rating_count","algorithm_version_id"]),
 "legacyPointers"=>$events->whereNotNull("legacy_rating_id")->count()
],JSON_UNESCAPED_SLASHES)."\n";

RATCJ_FAST_PREFLIGHT_PHP
)"
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- timeout 65s php -r "$php"
echo RATCJ_FAST_PREFLIGHT_COMPLETE=true
