#!/usr/bin/env bash
set -Eeuo pipefail
# Independent read-only physical CJ purge check with PostgreSQL guard hashes and zero-admission-vote invariants.
[[ $# -eq 0 ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == 'ghcr.io/daisycloversoftware/rum-api@sha256:a23e36323b55f3bce46d5b229146e2e7a9d70bdff9e16ea366380940e0629430' ]] || exit 78
php="$(cat <<'RATCJ_INDEPENDENT_PHP'

require "vendor/autoload.php";$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
$cj="01m07d2f9fj05f4ypdbwp087qj";$keep="01m0mvrxv8c3s12q0w7gwmzt03";
$deletedIds=["01m0fpz4zebnh4s0py9bs4kd14","01m0rbex77q0959rnnf79ndxfn","01m0ttpp2h5taqt3j8pw0qz1b3","01m0x1y1wb5mkane66yvrhwepf","01m0xdnqvt1v5nc38bd0pbt0qc","01m0xdzxty4d419q84kjx1kgj0","01m0xdpc6hb474tjtfrcnvvfe7","01m2gyq0f0kprpve1j7tcehxzt","01m2jqzv3d9r7b3yr52dg259sb","01m0rd6x8we2cwzdtzbhvj2m91","01m0rdg2dpnyjexmpc2cpqnr20","01m0rdqcz75km8v9xs8mvk9byh","01m0rdrx9rmdfczyfd5167rbqg","01m0rdwh3w9tt4jbz4bqtngdb5","01m0rnt0q8gdz64cbyhqe9fg5z","01m0rp1g2h13sws7fgzw6vgya0","01m0rp6cynyn1xxg8ckgcv4knx","01m0s1jr1zbxbpkdjcd143nb1v","01m0s1qn21ykct3at6a4f2h12n","01m0tp48gwbg6z6cgrr6jpdxdt","01m0tpcv2v3eg77kme54dpsy1j","01m0tq4w1wcmdpr0wdecq0fzs6","01m0trz11k75xsnr4bh4xfhd7w","01m0v0hzb71nz2xv3aa0yx0rb7","01m0v0ngt2rj5sxqfkxxt3p0sk","01m0v0qkxmr694fxwrjbgadaj3","01m0w8kfw3f2ggk18k59qwdr89","01m0w8prc21fg7qj52hs8tsbmd","01m0w8rrky2a3vs8s12ztbpdty","01m0way5m57z5ak3pvrr3qckyq","01m0wazw5jp2hhqaay8agj7tkh","01m0wb6kx0ab76kbthnw8x82aw","01m0wcwnyvs1gwg1c42b6hfg4f","01m0wcxv2s4813s6v62hccybm4","01m0x2bs8nv0a0x3tagm0cr5bg","01m0x7mrxb9jjm387e4qqgag33","01m0x7pg4zf99asrxfpgyerm9x","01m0xdnhar6j17g7prh26gbtev","01m0xk6256esr02wrhd0101d28","01m0xkk2ze0q8225kr61zvdgsz","01m0xsx91vy6mcv26qypbp51ba","01m0xsy6aet3014r18qv07kzmb","01m0yqkswgqhchqcb89sy19t6v","01m0yqmnbm9zmchscmt9h5v8bn","01m0yrp3mge8x3fxf09jfjmnes","01m0yrqmwfg8tedz9d02c90a02","01m0z0xvxbd80cjva4vrdwfhmp","01m0z110n52gwys2ck1h767csh","01m2gm0rbzf1apwdkp0gqanh14","01m2gm41wjc0wanpzp477rvx39","01m2gwqnvm32pp7aw91hc3ytb7","01m2gws4ex22jxbabs27b1b9pv","01m2t7yr1bdrbzdj8ycxz6exm8"];
$entity=App\Models\Entity::query()->with(["type","classificationAssignments.classification"])->findOrFail($cj);
$primary=$entity->classificationAssignments->first(fn($a)=>(bool)$a->is_primary&&$a->valid_to===null);
$photos=app(App\Services\ThingImageService::class)->publicImages($entity);
$events=App\Models\RatingEvent::query()->with(["rater","review","admissionState","admissionVotes","revisions","reasonSelections","evidence"])->where("target_entity_id",$cj)->get();
if($events->count()!==1||$events[0]->id!==$keep||count($deletedIds)!==53||
 $db::table("rating_events")->whereIn("id",$deletedIds)->exists()||
 $db::table("rating_event_reasons")->whereIn("rating_event_id",$deletedIds)->exists()||
 $db::table("rating_event_revisions")->whereIn("rating_event_id",$deletedIds)->exists()||
 $db::table("rating_event_admission_states")->whereIn("rating_event_id",$deletedIds)->exists()||
 $db::table("rating_event_admission_votes")->whereIn("rating_event_id",$deletedIds)->exists()||
 $db::table("rating_reviews")->whereIn("rating_event_id",$deletedIds)->exists()||
 $db::table("rating_journey_entries")->whereIn("rating_event_id",$deletedIds)->exists()||
 $db::table("rating_weight_snapshots")->whereIn("rating_event_id",$deletedIds)->exists()||
 $db::table("rating_evidence")->whereIn("rating_event_id",$deletedIds)->exists()||
 $db::table("outbox_events")->where("aggregate_type","rating_event")->whereIn("aggregate_id",$deletedIds)->exists())
 throw new RuntimeException("Purged row still present");
$rating=$events[0];$state=$rating->admissionState;
if($rating->rater_account_id!=="01kzgn6a7kswt3jqjb1sp8d4x0"||$rating->rater?->founder_number!==1||
 $rating->source!=="canonical_rum"||$rating->status!=="active"||$rating->withdrawn_at!==null||
 (int)$rating->value!==5||$rating->submitted_at?->format("Y-m-d H:i:s")!=="2026-08-22 13:50:12"||
 $rating->review?->status!=="visible"||
 hash("sha256",(string)$rating->review?->body)!=="aa2bcc4c7b1bf6348c46944f1bebd54298d661e4e2db030c08170a99b7d98e2e"||
 $rating->reasonSelections->isNotEmpty()||$rating->evidence->isNotEmpty()||
 $state?->state!=="needs_verdict"||(int)$state->keep_votes!==0||
 (int)$state->sin_bin_votes!==0||(int)$state->score!==0||$rating->admissionVotes->count()!==0)
 throw new RuntimeException("Selected founder original or zero vote state altered");
if($entity->type?->key!=="digital_resource"||$primary?->classification?->canonical_name!=="YouTube Channel"||
 $entity->visibility!=="public"||$entity->publication_state!=="published"||
 $entity->rateability!=="open"||$photos["profile"]===null||$photos["cover"]===null||
 $db::table("entity_relationships")->where(fn($q)=>$q->where("source_entity_id",$cj)->orWhere("target_entity_id",$cj))->whereNull("valid_to")->exists())
 throw new RuntimeException("Protected CJ profile or linked state altered");
$functions=[
 "rum_rating_event_evidence_guard"=>"53fa59f03e3059cfda3f1d23da7b8053",
 "rum_rating_review_evidence_guard"=>"6caf05605fa6d3f8664d32ea69da501e",
 "rum_rating_event_revision_guard"=>"03d579a7aca171564e54c0b876f06cb2",
 "rum_rating_dimension_value_guard"=>"1af6ea39d64e27ca81e044ceedc10b81"
];
foreach($functions as $func=>$md5){
 $r=$db::selectOne("SELECT md5(pg_get_functiondef(p.oid)) hash FROM pg_proc p WHERE p.oid=to_regprocedure(?)",["public.".$func."()"]);
 if(!$r||$r->hash!==$md5)throw new RuntimeException("DB immutability guard hash differs: ".$func);
}
$trigs=$db::table("pg_trigger")->whereIn("tgname",[
 "rum_rating_event_evidence_guard_trigger","rum_rating_review_evidence_guard_trigger",
 "rum_rating_event_revision_guard_trigger","rum_rating_dimension_value_guard_trigger"
])->get(["tgname","tgenabled"]);
if($trigs->count()!==4||$trigs->where("tgenabled","!=","O")->count()!==0)
 throw new RuntimeException("One or more DB immutability triggers disabled");
$marker=$db::table("outbox_events")->where("aggregate_type","entity")->where("aggregate_id",$cj)->
 where("event_type","rating.dev_permanent_cleanup_completed.v1")->
 where("payload->operationId","ratcj-hard-delete-20261118-owner-authorised-r1")->count();
if($marker!==1||$db::table("rating_events")->where("target_entity_id","!=",$cj)->count()!==80||
 $db::table("entity_reputation_snapshots")->where("entity_id",$cj)->count()!==0)
 throw new RuntimeException("Post-purge receipt, other events, or stale snapshot unexpected");
echo "RATCJ_HARD_PURGE_INDEPENDENT_AUDIT=".json_encode([
 "source"=>"rum-dev-isolated","entityId"=>$cj,"retainedRatingId"=>$keep,"physicalRatings"=>1,
 "deletedExpected"=>count($deletedIds),"allDeletedAbsent"=>true,
 "retainedValue"=>(int)$rating->value,"originalDate"=>$rating->submitted_at?->toAtomString(),
 "reviewPreserved"=>true,"admission"=>$state->state,"keepVotes"=>(int)$state->keep_votes,
 "sinBinVotes"=>(int)$state->sin_bin_votes,"net"=>(int)$state->score,
 "publicEligibleAccepted"=>0,"originalRevisions"=>$rating->revisions->count(),
 "relatedOtherThingsRatingCount"=>80,"activeLinkedThings"=>0,
 "type"=>$primary->classification?->canonical_name,"imagesPreserved"=>true,
 "staleReputationSnapshotCount"=>0,"immutabilityGuardFunctionsHashRestored"=>true,
 "allFourTriggersEnabled"=>true,"completionMarker"=>$marker,
],JSON_UNESCAPED_SLASHES)."\n";

RATCJ_INDEPENDENT_PHP
)"
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- timeout 65s php -r "$php"
echo RATCJ_HARD_PURGE_INDEPENDENT_COMPLETE=true
