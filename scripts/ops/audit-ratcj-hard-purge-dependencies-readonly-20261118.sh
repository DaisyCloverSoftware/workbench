#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only inventory of all 53 frozen CJ rating IDs, physical FKs, cross-refs, outbox, votes and reputation.
[[ $# -eq 0 ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == 'ghcr.io/daisycloversoftware/rum-api@sha256:a23e36323b55f3bce46d5b229146e2e7a9d70bdff9e16ea366380940e0629430' ]] || exit 78
php="$(cat <<'RATCJ_PURGE_READONLY_PHP'

require "vendor/autoload.php";
$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
$schema=Illuminate\Support\Facades\Schema::class;
$id="01m07d2f9fj05f4ypdbwp087qj";
$keep="01m0mvrxv8c3s12q0w7gwmzt03";
$purge=["01m0fpz4zebnh4s0py9bs4kd14","01m0rbex77q0959rnnf79ndxfn","01m0rd6x8we2cwzdtzbhvj2m91","01m0rdg2dpnyjexmpc2cpqnr20","01m0rdqcz75km8v9xs8mvk9byh","01m0rdrx9rmdfczyfd5167rbqg","01m0rdwh3w9tt4jbz4bqtngdb5","01m0rnt0q8gdz64cbyhqe9fg5z","01m0rp1g2h13sws7fgzw6vgya0","01m0rp6cynyn1xxg8ckgcv4knx","01m0s1jr1zbxbpkdjcd143nb1v","01m0s1qn21ykct3at6a4f2h12n","01m0tp48gwbg6z6cgrr6jpdxdt","01m0tpcv2v3eg77kme54dpsy1j","01m0tq4w1wcmdpr0wdecq0fzs6","01m0trz11k75xsnr4bh4xfhd7w","01m0ttpp2h5taqt3j8pw0qz1b3","01m0v0hzb71nz2xv3aa0yx0rb7","01m0v0ngt2rj5sxqfkxxt3p0sk","01m0v0qkxmr694fxwrjbgadaj3","01m0w8kfw3f2ggk18k59qwdr89","01m0w8prc21fg7qj52hs8tsbmd","01m0w8rrky2a3vs8s12ztbpdty","01m0way5m57z5ak3pvrr3qckyq","01m0wazw5jp2hhqaay8agj7tkh","01m0wb6kx0ab76kbthnw8x82aw","01m0wcwnyvs1gwg1c42b6hfg4f","01m0wcxv2s4813s6v62hccybm4","01m0x1y1wb5mkane66yvrhwepf","01m0x2bs8nv0a0x3tagm0cr5bg","01m0x7mrxb9jjm387e4qqgag33","01m0x7pg4zf99asrxfpgyerm9x","01m0xdnhar6j17g7prh26gbtev","01m0xdnqvt1v5nc38bd0pbt0qc","01m0xdpc6hb474tjtfrcnvvfe7","01m0xdzxty4d419q84kjx1kgj0","01m0xk6256esr02wrhd0101d28","01m0xkk2ze0q8225kr61zvdgsz","01m0xsx91vy6mcv26qypbp51ba","01m0xsy6aet3014r18qv07kzmb","01m0yqkswgqhchqcb89sy19t6v","01m0yqmnbm9zmchscmt9h5v8bn","01m0yrp3mge8x3fxf09jfjmnes","01m0yrqmwfg8tedz9d02c90a02","01m0z0xvxbd80cjva4vrdwfhmp","01m0z110n52gwys2ck1h767csh","01m2gm0rbzf1apwdkp0gqanh14","01m2gm41wjc0wanpzp477rvx39","01m2gwqnvm32pp7aw91hc3ytb7","01m2gws4ex22jxbabs27b1b9pv","01m2gyq0f0kprpve1j7tcehxzt","01m2jqzv3d9r7b3yr52dg259sb","01m2t7yr1bdrbzdj8ycxz6exm8"];
$all=App\Models\RatingEvent::query()->with(["review","admissionState","admissionVotes","reasonSelections","revisions"])->where("target_entity_id",$id)->orderBy("id")->get();
$actual=$all->pluck("id")->all();sort($actual);$expected=array_merge($purge,[$keep]);sort($expected);
if($actual!==$expected)throw new RuntimeException("CJ rating ID set diverged from exact baseline");
if($all->count()!==54||$all->where("status","active")->count()!==1||$all->firstWhere("status","active")?->id!==$keep)throw new RuntimeException("Not exactly selected one active");
$target=$all->firstWhere("id",$keep);
if($target?->rater_account_id!=="01kzgn6a7kswt3jqjb1sp8d4x0"||$target->source!=="canonical_rum"||
 $target->submitted_at?->format("Y-m-d H:i:s")!=="2026-08-22 13:50:12"||
 (int)$target->value!==5||$target->review?->status!=="visible"||
 hash("sha256",(string)$target->review?->body)!=="aa2bcc4c7b1bf6348c46944f1bebd54298d661e4e2db030c08170a99b7d98e2e"||
 $target->admissionState?->state!=="not_eligible")
 throw new RuntimeException("Retained original mismatch");
if($all->where("status","withdrawn")->count()!==6||$all->where("status","removed")->count()!==47)throw new RuntimeException("53 inactive statuses changed");
$fk=$db::select("
 SELECT n.nspname schema_name,c.relname table_name,a.attname column_name,k.ordinality column_position,
 con.conname constraint_name,con.confdeltype delete_action
 FROM pg_constraint con JOIN pg_class c ON c.oid=con.conrelid
 JOIN pg_namespace n ON n.oid=c.relnamespace
 JOIN LATERAL unnest(con.conkey) WITH ORDINALITY AS k(attnum,ordinality) ON TRUE
 JOIN pg_attribute a ON a.attrelid=c.oid AND a.attnum=k.attnum
 WHERE con.contype='f' AND con.confrelid=to_regclass('public.rating_events')
 ORDER BY n.nspname,c.relname,con.conname,k.ordinality");
$references=[];
foreach($fk as $f){
 $tablename=$f->schema_name.".".$f->table_name;
 $q=$db::table($tablename)->whereIn($f->column_name,$purge);
 $number=$q->count();
 $keepCount=$db::table($tablename)->where($f->column_name,$keep)->count();
 $references[]=["table"=>$tablename,"field"=>$f->column_name,"fk"=>$f->constraint_name,"deleteAction"=>$f->delete_action,"purgeReferences"=>$number,"keepReferences"=>$keepCount];
}
$known=[
 "rating_journey_entries"=>["rating_event_id","parent_rating_event_id"],
 "profile_help_me_out_pins"=>["rating_id"],
 "moderation_cases"=>["object_id"],
 "moderation_actions"=>["target_id"],
 "rating_weight_snapshots"=>["rating_event_id"],
 "rating_reason_votes"=>["rating_event_id"],
 "rating_event_admission_votes"=>["rating_event_id"]
];
$additional=[];
foreach($known as $table=>$columns){
 if(!$schema::hasTable($table))continue;
 foreach($columns as $column)if($schema::hasColumn($table,$column))$additional[]=["table"=>$table,"field"=>$column,
   "purgeReferences"=>$db::table($table)->whereIn($column,$purge)->count(),
   "keepReferences"=>$db::table($table)->where($column,$keep)->count()];
}
$journeyMap=$db::table("rating_journey_entries as je")->join("rating_journeys as j","j.id","=","je.rating_journey_id")
 ->whereIn("je.rating_event_id",$purge)->get(["je.id","je.rating_journey_id","je.entity_id","j.root_entity_id"]);
$journeyCounts=[
 "entriesForPurge"=>$journeyMap->count(),
 "entriesOfDifferentEntity"=>$journeyMap->where("entity_id","!=",$id)->count(),
 "journeysOfDifferentRoot"=>$journeyMap->where("root_entity_id","!=",$id)->count(),
 "parentPurgeReferences"=>$db::table("rating_journey_entries")->whereIn("parent_rating_event_id",$purge)->count(),
 "entriesForKept"=>$db::table("rating_journey_entries")->where("rating_event_id",$keep)->count()
];
$outbox=$db::table("outbox_events")->where("aggregate_type","rating_event")->whereIn("aggregate_id",$purge)->get(["id","event_type","aggregate_id","aggregate_type"]);
$pattern=implode("|",$purge);
$embedded=$db::table("outbox_events")->whereRaw("payload::text ~* ?",[$pattern])->get(["id","aggregate_type","aggregate_id","event_type"]);
$nonRating=$embedded->filter(fn($x)=>!($x->aggregate_type==="rating_event" && in_array($x->aggregate_id,$purge,true)))->values()->all();
$receiptCount=$db::table("reputation_event_receipts")->whereIn("outbox_event_id",$outbox->pluck("id")->all())->count();
$records=$all->whereIn("id",$purge);
$summary=[
 "targetId"=>$id,"keep"=>$keep,"allCount"=>$all->count(),"purgeCount"=>$records->count(),
 "purgeStatusCounts"=>$records->countBy("status"),"keptStatus"=>$target->status,"keptAdmission"=>$target->admissionState?->state,
 "purgeIds"=>$purge,"ratingReviewCount"=>$records->filter(fn($x)=>$x->review!==null)->count(),
 "ratingReasonCount"=>$records->sum(fn($x)=>$x->reasonSelections->count()),
 "admissionVoteCount"=>$records->sum(fn($x)=>$x->admissionVotes->count()),
 "purgeRevisionCount"=>$records->sum(fn($x)=>$x->revisions->count()),
 "relationshipActive"=>$db::table("entity_relationships")->where(fn($q)=>$q->where("source_entity_id",$id)->orWhere("target_entity_id",$id))->whereNull("valid_to")->count(),
 "foreignKeys"=>$references,"knownReferenceCounts"=>$additional,"journeys"=>$journeyCounts,
 "ratingEventOutboxCounts"=>$outbox->groupBy("event_type")->map(fn($x)=>$x->count()),
 "outboxCount"=>$outbox->count(),"embeddedEventCount"=>$embedded->count(),"embeddedOtherEvents"=>$nonRating,
 "reputationEventReceipts"=>$receiptCount,
 "snapshots"=>$db::table("entity_reputation_snapshots")->where("entity_id",$id)->get(["id","is_current","rating_count","score","algorithm_version_id","calculated_at"]),
 "otherEntityRatingEventCount"=>$db::table("rating_events")->where("target_entity_id","!=",$id)->count(),
 "ratingsLegacyPointerCount"=>$records->whereNotNull("legacy_rating_id")->count()
];
echo "RATCJ_PURGE_PREFLIGHT=".json_encode($summary,JSON_UNESCAPED_SLASHES|JSON_INVALID_UTF8_SUBSTITUTE)."\n";

RATCJ_PURGE_READONLY_PHP
)"
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$php"
echo RATCJ_PURGE_PREFLIGHT_COMPLETE=true
