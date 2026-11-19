#!/usr/bin/env bash
set -Eeuo pipefail
# Independent post-deploy canonical DEV data and public API audit; read-only and exact image/source.
[[ $# -eq 2 && "$1" =~ ^[0-9a-f]{40}$ ]] || exit 64
src="$1";api="$2"
[[ "$api" =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api" ]] || exit 78
health="$(curl -fsS --retry 3 --max-time 20 https://dev-rum.daisycloversoftware.uk/api/v1/health)"
python3 - "$health" "$src" <<'PY'
import json,sys
h=json.loads(sys.argv[1])
assert h.get("status")=="ok" and sys.argv[2] in str(h.get("version",""))
PY
php="$(cat <<'RATCJ_POSTDEPLOY_DB_PHP'

require "vendor/autoload.php";$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
$cj="01m07d2f9fj05f4ypdbwp087qj";$keep="01m0mvrxv8c3s12q0w7gwmzt03";
$deleted=["01m0fpz4zebnh4s0py9bs4kd14","01m0rbex77q0959rnnf79ndxfn","01m0ttpp2h5taqt3j8pw0qz1b3","01m0x1y1wb5mkane66yvrhwepf","01m0xdnqvt1v5nc38bd0pbt0qc","01m0xdzxty4d419q84kjx1kgj0","01m0xdpc6hb474tjtfrcnvvfe7","01m2gyq0f0kprpve1j7tcehxzt","01m2jqzv3d9r7b3yr52dg259sb","01m0rd6x8we2cwzdtzbhvj2m91","01m0rdg2dpnyjexmpc2cpqnr20","01m0rdqcz75km8v9xs8mvk9byh","01m0rdrx9rmdfczyfd5167rbqg","01m0rdwh3w9tt4jbz4bqtngdb5","01m0rnt0q8gdz64cbyhqe9fg5z","01m0rp1g2h13sws7fgzw6vgya0","01m0rp6cynyn1xxg8ckgcv4knx","01m0s1jr1zbxbpkdjcd143nb1v","01m0s1qn21ykct3at6a4f2h12n","01m0tp48gwbg6z6cgrr6jpdxdt","01m0tpcv2v3eg77kme54dpsy1j","01m0tq4w1wcmdpr0wdecq0fzs6","01m0trz11k75xsnr4bh4xfhd7w","01m0v0hzb71nz2xv3aa0yx0rb7","01m0v0ngt2rj5sxqfkxxt3p0sk","01m0v0qkxmr694fxwrjbgadaj3","01m0w8kfw3f2ggk18k59qwdr89","01m0w8prc21fg7qj52hs8tsbmd","01m0w8rrky2a3vs8s12ztbpdty","01m0way5m57z5ak3pvrr3qckyq","01m0wazw5jp2hhqaay8agj7tkh","01m0wb6kx0ab76kbthnw8x82aw","01m0wcwnyvs1gwg1c42b6hfg4f","01m0wcxv2s4813s6v62hccybm4","01m0x2bs8nv0a0x3tagm0cr5bg","01m0x7mrxb9jjm387e4qqgag33","01m0x7pg4zf99asrxfpgyerm9x","01m0xdnhar6j17g7prh26gbtev","01m0xk6256esr02wrhd0101d28","01m0xkk2ze0q8225kr61zvdgsz","01m0xsx91vy6mcv26qypbp51ba","01m0xsy6aet3014r18qv07kzmb","01m0yqkswgqhchqcb89sy19t6v","01m0yqmnbm9zmchscmt9h5v8bn","01m0yrp3mge8x3fxf09jfjmnes","01m0yrqmwfg8tedz9d02c90a02","01m0z0xvxbd80cjva4vrdwfhmp","01m0z110n52gwys2ck1h767csh","01m2gm0rbzf1apwdkp0gqanh14","01m2gm41wjc0wanpzp477rvx39","01m2gwqnvm32pp7aw91hc3ytb7","01m2gws4ex22jxbabs27b1b9pv","01m2t7yr1bdrbzdj8ycxz6exm8"];
$cjEntity=App\Models\Entity::query()->with(["type","classificationAssignments.classification"])->findOrFail($cj);
$primary=$cjEntity->classificationAssignments->first(fn($x)=>(bool)$x->is_primary&&$x->valid_to===null);
$images=app(App\Services\ThingImageService::class)->publicImages($cjEntity);
$events=App\Models\RatingEvent::query()->with(["rater","review","admissionState","admissionVotes","revisions","reasonSelections"])->where("target_entity_id",$cj)->get();
if($events->count()!==1||$events[0]->id!==$keep||count($deleted)!==53||$db::table("rating_events")->whereIn("id",$deleted)->exists()||
 $db::table("rating_event_reasons")->whereIn("rating_event_id",$deleted)->exists()||
 $db::table("rating_event_admission_votes")->whereIn("rating_event_id",$deleted)->exists()||
 $db::table("rating_event_revisions")->whereIn("rating_event_id",$deleted)->exists()||
 $db::table("rating_reviews")->whereIn("rating_event_id",$deleted)->exists()||
 $db::table("rating_weight_snapshots")->whereIn("rating_event_id",$deleted)->exists()||
 $db::table("rating_journey_entries")->whereIn("rating_event_id",$deleted)->exists()||
 $db::table("outbox_events")->where("aggregate_type","rating_event")->whereIn("aggregate_id",$deleted)->exists())
 throw new RuntimeException("Incorrect permanent purge result in canonical DEV");
$rating=$events[0];$state=$rating->admissionState;
if($rating->rater_account_id!=="01kzgn6a7kswt3jqjb1sp8d4x0"||$rating->rater?->founder_number!==1||
 (int)$rating->value!==5||$rating->status!=="active"||
 $rating->submitted_at?->format("Y-m-d H:i:s")!=="2026-08-22 13:50:12"||
 hash("sha256",(string)$rating->review?->body)!=="aa2bcc4c7b1bf6348c46944f1bebd54298d661e4e2db030c08170a99b7d98e2e"||
 $rating->reasonSelections->isNotEmpty()||$state?->state!=="needs_verdict"||
 (int)$state->keep_votes!==0||(int)$state->sin_bin_votes!==0||(int)$state->score!==0||
 $rating->admissionVotes->count()!==0)
 throw new RuntimeException("Retained original rating or zero community votes altered");
if($cjEntity->canonical_name!=="CJ Investigates"||$cjEntity->type?->key!=="digital_resource"||
 $primary?->classification?->canonical_name!=="YouTube Channel"||
 $cjEntity->visibility!=="public"||$cjEntity->publication_state!=="published"||$cjEntity->rateability!=="open"||
 $images["profile"]===null||$images["cover"]===null||
 $db::table("entity_reputation_snapshots")->where("entity_id",$cj)->exists()||
 $db::table("entity_relationships")->where(fn($q)=>$q->where("source_entity_id",$cj)->orWhere("target_entity_id",$cj))->whereNull("valid_to")->exists())
 throw new RuntimeException("CJ profile/classification or reputation/relationships have changed");
$otherOriginal=$db::table("rating_events")->where("target_entity_id","!=",$cj)
 ->where("source","!=","ratcj-pr179-qa")->count();
if($otherOriginal<80)throw new RuntimeException("Other Things rating event count decreased unexpectedly");
echo "RATCJ_PR179_POSTDEPLOY_CANONICAL_PASS=".json_encode(["physicalRatings"=>1,"protectedId"=>$keep,
 "deleted53Absent"=>true,"reviewPreserved"=>true,"primaryClass"=>"YouTube Channel",
 "ratingValue"=>5,"admissionState"=>"needs_verdict","keep"=>0,"sinBin"=>0,"net"=>0,
 "currentReputationSnapshots"=>0,"activeLinkedThings"=>0,"bothImagesPublished"=>true,
 "otherEntityRatingEventsCount"=>$otherOriginal],JSON_UNESCAPED_SLASHES)."\n";

RATCJ_POSTDEPLOY_DB_PHP
)"
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- timeout 65s php -r "$php"
response="$(curl -fsS --retry 3 --max-time 20 -H 'Cache-Control: no-cache' -H 'Origin: https://dev-rum-ra.daisycloversoftware.uk' 'https://dev-rum.daisycloversoftware.uk/api/v1/public/entities/rat-catalogue/nodes/01m07d2f9fj05f4ypdbwp087qj/ratings')"
python3 - "$response" <<'PY'
import json,sys
x=json.loads(sys.argv[1])
assert x["meta"]["source"]=="rum-dev-canonical"
assert x["meta"]["privacy"]=="private-rum-people-excluded"
d=x["data"];assert d["counts"]=={"publicSubmitted":1,"accepted":0,"awaitingVerdict":1,"inReview":0}
assert len(d["items"])==1
r=d["items"][0]
assert r["id"]=="01m0mvrxv8c3s12q0w7gwmzt03" and r["value"]==5
assert r["review"]=="I have been watching CJ's progress over the years and he is starting to make some great content."
assert r["reasons"]==[] and r["category"]=="General RUM rating"
assert r["verdict"]=={"state":"needs_verdict","keepVotes":0,"sinBinVotes":0,"score":0,"countsTowardAccepted":False}
print("RATCJ_PR179_POSTDEPLOY_PUBLIC_RATINGS_API_PASS=true")
PY
echo RATCJ_PR179_POSTDEPLOY_INDEPENDENT_ACCEPTANCE_COMPLETE=true
