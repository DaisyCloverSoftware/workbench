#!/usr/bin/env bash
set -Eeuo pipefail
# Owner-selected, fixed-ID canonical DEV-only rating event maintenance through Laravel model lifecycle.
# Usage: preflight / apply. Never delete events, rewrite verdicts, change media, mutate LIVE, or impersonate accounts.
[[ $# -eq 1 && ( "$1" == "inspect" || "$1" == "apply" ) ]] || exit 64
mode="$1"
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
api='ghcr.io/daisycloversoftware/rum-api@sha256:a23e36323b55f3bce46d5b229146e2e7a9d70bdff9e16ea366380940e0629430'
rat='ghcr.io/daisycloversoftware/rum-rate-anything@sha256:ec11bfec432ff10d508850a0a071b3af3787bff656d2decca90eed5951cd0003'
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api" ]] || { echo "API deployment source mismatch"; exit 78; }
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == "$rat" ]] || { echo "RAT deployment source mismatch"; exit 78; }
php="$(cat <<'RAT_CJ_GUARDED_LIFECYCLE_PHP'

require "vendor/autoload.php";
$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
$id="01m07d2f9fj05f4ypdbwp087qj";
$keep="01m0mvrxv8c3s12q0w7gwmzt03";
$owner="01kzgn6a7kswt3jqjb1sp8d4x0";
$operation="ratcj-20261118-owner-original-rating-cleanup-r1";
$mode=$argv[1]??"inspect";
if(!in_array($mode,["inspect","apply"],true))throw new RuntimeException("Unsupported mode");
$expected=[
  "01m0fpz4zebnh4s0py9bs4kd14"=>["rater"=>"01kzgn6a7kswt3jqjb1sp8d4x0","source"=>"rateurmate_public_identity","status"=>"withdrawn","submitted"=>"2026-08-20 13:50:01","value"=>5],
  "01m0mvrxv8c3s12q0w7gwmzt03"=>["rater"=>"01kzgn6a7kswt3jqjb1sp8d4x0","source"=>"canonical_rum","status"=>"withdrawn","submitted"=>"2026-08-22 13:50:12","value"=>5],
  "01m0rbex77q0959rnnf79ndxfn"=>["rater"=>"01m0rbemt8v5n05pm8yahz503c","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-23 22:22:04","value"=>5],
  "01m0rd6x8we2cwzdtzbhvj2m91"=>["rater"=>"01m0rd6ny7fbyed8350w19wwky","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-23 22:52:39","value"=>5],
  "01m0rdg2dpnyjexmpc2cpqnr20"=>["rater"=>"01m0rdftn1d01z2nanv23z4d3d","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-23 22:57:39","value"=>5],
  "01m0rdqcz75km8v9xs8mvk9byh"=>["rater"=>"01m0rdq5yfe02nd78gts4amyzh","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-23 23:01:39","value"=>5],
  "01m0rdrx9rmdfczyfd5167rbqg"=>["rater"=>"01m0rdrpvex3p7fkxkwwn30xb3","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-23 23:02:29","value"=>5],
  "01m0rdwh3w9tt4jbz4bqtngdb5"=>["rater"=>"01m0rdw939q07hs3bvzkkm7tde","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-23 23:04:28","value"=>5],
  "01m0rnt0q8gdz64cbyhqe9fg5z"=>["rater"=>"01m0rnss81e3a451xd21g871xt","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-24 01:22:54","value"=>5],
  "01m0rp1g2h13sws7fgzw6vgya0"=>["rater"=>"01m0rp19v724f3nae8djvb8hjm","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-24 01:26:59","value"=>5],
  "01m0rp6cynyn1xxg8ckgcv4knx"=>["rater"=>"01m0rp63gt7g4aq4cfcr3wa0kn","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-24 01:29:40","value"=>5],
  "01m0s1jr1zbxbpkdjcd143nb1v"=>["rater"=>"01m0s1jebd4kyhfa3d30wz6ncd","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-24 04:48:38","value"=>5],
  "01m0s1qn21ykct3at6a4f2h12n"=>["rater"=>"01m0s1qbdx6et48exd4w1jqytv","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-24 04:51:19","value"=>5],
  "01m0tp48gwbg6z6cgrr6jpdxdt"=>["rater"=>"01m0tp3ws82pg232zd8wjn5g9k","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-24 20:06:58","value"=>5],
  "01m0tpcv2v3eg77kme54dpsy1j"=>["rater"=>"01m0tpcexndpfdr8enc3d763rk","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-24 20:11:39","value"=>5],
  "01m0tq4w1wcmdpr0wdecq0fzs6"=>["rater"=>"01m0tq4fmjps3s3d2h6fb3q6as","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-24 20:24:47","value"=>5],
  "01m0trz11k75xsnr4bh4xfhd7w"=>["rater"=>"01m0tryn16hafzytnvcv9by7j1","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-24 20:56:33","value"=>5],
  "01m0ttpp2h5taqt3j8pw0qz1b3"=>["rater"=>"01kzgn6a7kswt3jqjb1sp8d4x0","source"=>"rate_anything","status"=>"withdrawn","submitted"=>"2026-08-24 21:26:56","value"=>5],
  "01m0v0hzb71nz2xv3aa0yx0rb7"=>["rater"=>"01m0v0hsn86kryvsgtbwcj8wrw","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-24 23:09:13","value"=>5],
  "01m0v0ngt2rj5sxqfkxxt3p0sk"=>["rater"=>"01m0v0nbmrx62gbt0fwq0pqne9","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-24 23:11:10","value"=>5],
  "01m0v0qkxmr694fxwrjbgadaj3"=>["rater"=>"01m0v0q6zfkj8vtzzhv17w2ack","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-24 23:12:18","value"=>5],
  "01m0w8kfw3f2ggk18k59qwdr89"=>["rater"=>"01m0w8k62yhjecx95evn7mhwh5","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-25 10:49:06","value"=>5],
  "01m0w8prc21fg7qj52hs8tsbmd"=>["rater"=>"01m0w8pe7x942gya2btn4qs9wk","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-25 10:50:53","value"=>5],
  "01m0w8rrky2a3vs8s12ztbpdty"=>["rater"=>"01m0w8rawnz5hsrh77892eedfd","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-25 10:51:59","value"=>5],
  "01m0way5m57z5ak3pvrr3qckyq"=>["rater"=>"01m0waxvrzhfnp0htbfm819eps","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-25 11:29:53","value"=>5],
  "01m0wazw5jp2hhqaay8agj7tkh"=>["rater"=>"01m0wazf7weyqzxvjjbx91sr1r","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-25 11:30:49","value"=>5],
  "01m0wb6kx0ab76kbthnw8x82aw"=>["rater"=>"01m0wb66cbae0k4v2harqdqjzj","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-25 11:34:30","value"=>5],
  "01m0wcwnyvs1gwg1c42b6hfg4f"=>["rater"=>"01m0wcwb92a247w3rhx8czkr17","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-25 12:04:02","value"=>5],
  "01m0wcxv2s4813s6v62hccybm4"=>["rater"=>"01m0wcxew5xxw96s0g0dmkycvd","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-25 12:04:40","value"=>5],
  "01m0x1y1wb5mkane66yvrhwepf"=>["rater"=>"01m0x1xm6dnyrpnr5ktqchn8cv","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-25 18:11:47","value"=>5],
  "01m0x2bs8nv0a0x3tagm0cr5bg"=>["rater"=>"01m0x2b7z90zj36nv78xx25j33","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-25 18:19:17","value"=>5],
  "01m0x7mrxb9jjm387e4qqgag33"=>["rater"=>"01m0x7m96sdddgbf2bsw3h4k2w","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-25 19:51:34","value"=>5],
  "01m0x7pg4zf99asrxfpgyerm9x"=>["rater"=>"01m0x7nyrsy0m88h1hwtcjc94d","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-25 19:52:31","value"=>5],
  "01m0xdnhar6j17g7prh26gbtev"=>["rater"=>"01m0xdn4nm5kr1ke693s9389s7","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-25 21:36:51","value"=>5],
  "01m0xdnqvt1v5nc38bd0pbt0qc"=>["rater"=>"01kzgn6a7kswt3jqjb1sp8d4x0","source"=>"rate_anything","status"=>"withdrawn","submitted"=>"2026-08-25 21:36:57","value"=>5],
  "01m0xdpc6hb474tjtfrcnvvfe7"=>["rater"=>"01m0xdnxb6v9rbhd13dz767pg4","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-25 21:37:18","value"=>5],
  "01m0xdzxty4d419q84kjx1kgj0"=>["rater"=>"01kzgn6a7kswt3jqjb1sp8d4x0","source"=>"rate_anything","status"=>"withdrawn","submitted"=>"2026-08-25 21:42:31","value"=>5],
  "01m0xk6256esr02wrhd0101d28"=>["rater"=>"01m0xk5k9dscn1p0wecpzkd5md","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-25 23:13:15","value"=>5],
  "01m0xkk2ze0q8225kr61zvdgsz"=>["rater"=>"01m0xkjfz9p7fkd7td121k27wa","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-25 23:20:22","value"=>5],
  "01m0xsx91vy6mcv26qypbp51ba"=>["rater"=>"01m0xswm8j8nwkvd19ghpbdxdb","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-26 01:10:47","value"=>5],
  "01m0xsy6aet3014r18qv07kzmb"=>["rater"=>"01m0xsxmdzw7j8hshvy5bsp83p","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-26 01:11:17","value"=>5],
  "01m0yqkswgqhchqcb89sy19t6v"=>["rater"=>"01m0yqka9y23f1fwf5ysngcnds","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-26 09:49:54","value"=>5],
  "01m0yqmnbm9zmchscmt9h5v8bn"=>["rater"=>"01m0yqm4csm579v6zsz9emny5d","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-26 09:50:22","value"=>5],
  "01m0yrp3mge8x3fxf09jfjmnes"=>["rater"=>"01m0yrn9h7jbfrkd0m3xyvqqyd","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-26 10:08:38","value"=>5],
  "01m0yrqmwfg8tedz9d02c90a02"=>["rater"=>"01m0yrq4ytb807fzn6n1t7h3g1","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-26 10:09:28","value"=>5],
  "01m0z0xvxbd80cjva4vrdwfhmp"=>["rater"=>"01m0z0x9mqyvd93tk7yc5qvvbm","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-26 12:32:41","value"=>5],
  "01m0z110n52gwys2ck1h767csh"=>["rater"=>"01m0z10g1f2yce1j630yqa8egq","source"=>"rate_anything","status"=>"active","submitted"=>"2026-08-26 12:34:24","value"=>5],
  "01m2gm0rbzf1apwdkp0gqanh14"=>["rater"=>"01m2gm0qbekfk761g8xqyxc6pf","source"=>"rate_anything","status"=>"active","submitted"=>"2026-09-14 18:49:06","value"=>5],
  "01m2gm41wjc0wanpzp477rvx39"=>["rater"=>"01m2gm40hfvh3w4cfxnkj38vjc","source"=>"rate_anything","status"=>"active","submitted"=>"2026-09-14 18:50:54","value"=>5],
  "01m2gwqnvm32pp7aw91hc3ytb7"=>["rater"=>"01m2gwqmr2gmcer0jdx0ypatj6","source"=>"rate_anything","status"=>"active","submitted"=>"2026-09-14 21:21:25","value"=>5],
  "01m2gws4ex22jxbabs27b1b9pv"=>["rater"=>"01m2gws2vy06wwjr7aderv30qe","source"=>"rate_anything","status"=>"active","submitted"=>"2026-09-14 21:22:13","value"=>5],
  "01m2gyq0f0kprpve1j7tcehxzt"=>["rater"=>"01kzgn6a7kswt3jqjb1sp8d4x0","source"=>"rate_anything","status"=>"withdrawn","submitted"=>"2026-09-14 21:56:01","value"=>5],
  "01m2jqzv3d9r7b3yr52dg259sb"=>["rater"=>"01kzgn6a7kswt3jqjb1sp8d4x0","source"=>"rate_anything","status"=>"withdrawn","submitted"=>"2026-09-15 14:36:59","value"=>5],
  "01m2t7yr1bdrbzdj8ycxz6exm8"=>["rater"=>"01kzgn6a7kswt3jqjb1sp8d4x0","source"=>"rate_anything","status"=>"active","submitted"=>"2026-09-18 12:30:41","value"=>5]
];
if(count($expected)!==54||!isset($expected[$keep]))throw new RuntimeException("Frozen rating set invalid");
$success=$db::transaction(function()use($db,$id,$keep,$owner,$operation,$mode,$expected){
  $db::statement("SET TRANSACTION ISOLATION LEVEL SERIALIZABLE");
  $entity=App\Models\Entity::query()->with(["type","classificationAssignments.classification"])->lockForUpdate()->findOrFail($id);
  $primary=$entity->classificationAssignments->first(fn($assignment)=>(bool)$assignment->is_primary&&$assignment->valid_to===null);
  if($entity->canonical_name!=="CJ Investigates"||$entity->type?->key!=="digital_resource"||
     $primary?->classification?->canonical_name!=="YouTube Channel"||
     $entity->publication_state!=="published"||$entity->visibility!=="public"||$entity->rateability!=="open"||
     $entity->deleted_at!==null)throw new RuntimeException("CJ entity invariant changed");
  if($db::table("entity_relationships")->where(fn($q)=>$q->where("source_entity_id",$id)->orWhere("target_entity_id",$id))->whereNull("valid_to")->count()!==0)throw new RuntimeException("CJ relationship graph changed");
  if($db::table("entity_relationships")->where(fn($q)=>$q->where("source_entity_id",$id)->orWhere("target_entity_id",$id))->count()!==22)throw new RuntimeException("CJ relation history changed");
  $images=app(App\Services\ThingImageService::class)->publicImages($entity);
  if($images["profile"]===null||$images["cover"]===null)throw new RuntimeException("Published photos unavailable");
  $imageFingerprint=hash("sha256",json_encode($images,JSON_UNESCAPED_SLASHES));
  $profileRevisionFingerprint=hash("sha256",$db::table("entity_profile_revisions")->where("entity_id",$id)->orderBy("id")->get()->toJson());
  $otherEntityRatingsFingerprint=hash("sha256",$db::table("rating_events")->where("target_entity_id","!=",$id)->orderBy("id")->get()->toJson());
  $records=App\Models\RatingEvent::query()->with(["rater","review","admissionState","admissionVotes","reasonSelections","evidence","revisions"])
   ->where("target_entity_id",$id)->orderBy("id")->lockForUpdate()->get();
  $actual=$records->pluck("id")->all();
  $frozen=array_keys($expected);
  sort($actual);sort($frozen);
  if($actual!==$frozen||$records->count()!==54)throw new RuntimeException("Exact 54-rating inventory changed");
  $markerQuery=$db::table("outbox_events")->where("aggregate_type","entity")->where("aggregate_id",$id)
   ->where("event_type","rating.dev_owner_original_cleanup_completed.v1")->where("payload->operationId",$operation);
  $markers=$markerQuery->count();
  if($markers>1)throw new RuntimeException("Duplicate operation receipts");
  $selected=$records->firstWhere("id",$keep);
  $review=$selected?->review;
  if(!$selected||$selected->rater_account_id!==$owner||$selected->rater?->founder_number!==1||
     $selected->source!=="canonical_rum"||$selected->audience!=="public"||(int)$selected->value!==5||
     $selected->submitted_at?->format("Y-m-d H:i:s")!=="2026-08-22 13:50:12"||
     $selected->created_at?->format("Y-m-d H:i:s")!=="2026-08-22 13:50:12"||
     $selected->rating_context_id!==null||$selected->reasonSelections->count()!==0||
     $review?->status!=="visible"||$review?->author_account_id!==$owner||
     hash("sha256",(string)$review?->body)!=="aa2bcc4c7b1bf6348c46944f1bebd54298d661e4e2db030c08170a99b7d98e2e"||
     $selected->admissionState?->state!=="not_eligible"||$selected->admissionVotes->count()!==0)
   throw new RuntimeException("Protected founder evidence differs from selected original");
  $protectedFingerprint=hash("sha256",json_encode([
    $selected->id,$selected->rater_account_id,$selected->target_entity_id,$selected->rating_scheme_id,
    $selected->rating_context_id,$selected->relationship_entity_id,$selected->value,$selected->audience,
    $selected->source,$selected->legacy_rating_id,$selected->idempotency_key,$selected->one_active_per_rater,
    $selected->metadata,$selected->submitted_at?->toAtomString(),$selected->created_at?->toAtomString(),
    $review->id,$review->author_account_id,$review->body,$review->status,
    $selected->reasonSelections->toJson(),$selected->evidence->toJson(),$selected->revisions->toJson(),
  ],JSON_UNESCAPED_SLASHES));
  $qa=[];$founderCompeting=[];$activeRemove=[];
  foreach($records as $event){
    $f=$expected[(string)$event->id]??null;
    if($f===null||$event->rater_account_id!==$f["rater"]||$event->source!==$f["source"]||
       $event->submitted_at?->format("Y-m-d H:i:s")!==$f["submitted"]||
       (int)$event->value!==$f["value"]||$event->audience!=="public")
      throw new RuntimeException("Frozen evidence mismatch: ".$event->id);
    if($event->id===$keep)continue;
    $isFounder=$event->rater_account_id===$owner;
    if($isFounder){
      if($event->rater?->founder_number!==1)throw new RuntimeException("Founder attribution mismatch");
      $founderCompeting[]=(string)$event->id;
    } else {
      $u=$event->rater;
      if(!$u||!str_ends_with(mb_strtolower(trim((string)$u->email)),"@example.com")||
         !preg_match("/^(rumcheck|rumowner|rumrate|rumcj|rumagain|ratowner)[0-9]/",(string)$u->username)||
         $event->source!=="rate_anything"||($markers===0 && $event->revisions->count()!==0)||
         $db::table("rating_journey_entries")->where("rating_event_id",$event->id)->count()===0)
        throw new RuntimeException("Unrecognised synthetic account or journey: ".$event->id);
      $qa[]=(string)$event->id;
    }
    if($markers===0 && $event->status!==$f["status"])
      throw new RuntimeException("Competing rating status changed since audit: ".$event->id);
    if($event->status==="active")$activeRemove[]=(string)$event->id;
    elseif($event->status!=="withdrawn" && $markers===0)throw new RuntimeException("Unexpected inactive status");
  }
  if(count($qa)!==46||count($founderCompeting)!==7)throw new RuntimeException("QA/founder partition mismatch");
  $preActive=$records->where("status","active")->count();
  $preWithdrawn=$records->where("status","withdrawn")->count();
  $voteRows=$db::table("rating_event_admission_votes")->whereIn("rating_event_id",$actual)->orderBy("id")->get();
  if($voteRows->count()!==17||$voteRows->whereIn("rating_event_id",$qa)->count()!==17||
     $db::table("rating_reason_votes")->whereIn("rating_event_id",$actual)->count()!==0)
    throw new RuntimeException("Community vote dependency baseline changed");
  $founderVoteCount=$db::table("rating_event_admission_votes as v")->join("users as u","u.id","=","v.voter_id")
    ->whereIn("v.rating_event_id",$qa)->where("u.id",$owner)->count();
  if($founderVoteCount!==5)throw new RuntimeException("Founder QA-vote dependency changed");
  $voteFingerprint=hash("sha256",$voteRows->toJson());
  if($markers===1){
    if($selected->status!=="active"||$selected->withdrawn_at!==null||$preActive!==1||
       $records->where("status","removed")->count()!==47||
       $records->where("status","withdrawn")->count()!==6)
       throw new RuntimeException("Existing completed operation receipt conflicts with data");
    echo "CJ_RATING_CLEANUP_ALREADY_APPLIED=".json_encode(["operationId"=>$operation,"keptId"=>$keep,"active"=>1,"retired"=>47,"historicalWithdrawn"=>6],JSON_UNESCAPED_SLASHES)."\n";
    return ["status"=>"already_applied"];
  }
  if($selected->status!=="withdrawn"||$selected->withdrawn_at===null||$preActive!==47||$preWithdrawn!==7||
     count($activeRemove)!==47||$records->where("rater_account_id",$owner)->count()!==8)
    throw new RuntimeException("Current pre-cleanup rating state mismatches frozen audit");
  $plan=["operationId"=>$operation,"keptId"=>$keep,"protectedValue"=>(int)$selected->value,
    "submittedAt"=>$selected->submitted_at?->toAtomString(),"protectedReviewSha256"=>hash("sha256",(string)$review->body),
    "qaIds"=>$qa,"founderCompetingIds"=>$founderCompeting,"activeRetireIds"=>$activeRemove,
    "previouslyWithdrawnCompeting"=>6,"qaVoteCount"=>$voteRows->count(),"founderVotesOnQA"=>$founderVoteCount,
    "expectedResult"=>["totalHistory"=>54,"active"=>1,"inactive"=>53]];
  echo "CJ_RATING_GUARDED_PLAN=".json_encode($plan,JSON_UNESCAPED_SLASHES)."\n";
  if($mode==="inspect")return ["status"=>"preflight_pass"];
  $now=now();$removedIds=[];
  foreach($records as $e){
    if($e->id===$keep||$e->status!=="active")continue;
    $from=(string)$e->status;
    $e->forceFill(["status"=>"removed"])->save();
    App\Models\RatingEventRevision::query()->create([
      "rating_event_id"=>$e->id,"revision_type"=>"dev_data_cleanup_removed",
      "changed_by_user_id"=>null,
      "before_state"=>["status"=>$from,"withdrawnAt"=>$e->withdrawn_at?->toAtomString()],
      "after_state"=>["status"=>"removed","withdrawnAt"=>$e->withdrawn_at?->toAtomString()],
      "reason"=>"Owner-approved isolated DEV QA cleanup: ".$operation,
      "occurred_at"=>$now,
    ]);
    App\Models\OutboxEvent::query()->create([
      "aggregate_type"=>"rating_event","aggregate_id"=>$e->id,
      "event_type"=>"rating.event_moderation_changed.v1",
      "payload"=>["ratingEventId"=>$e->id,"entityId"=>$id,"from"=>$from,"to"=>"removed",
        "maintenanceOperationId"=>$operation,"recalculateReputation"=>true,"manualScoreEdit"=>false],
      "occurred_at"=>$now,"attempts"=>0,
    ]);
    $removedIds[]=(string)$e->id;
  }
  if(count($removedIds)!==47)throw new RuntimeException("Wrong number of retire transitions");
  $oldWithdrawal=$selected->withdrawn_at?->toAtomString();
  $selected->forceFill(["status"=>"active","withdrawn_at"=>null])->save();
  App\Models\RatingEventRevision::query()->create([
    "rating_event_id"=>$keep,"revision_type"=>"dev_data_cleanup_restored",
    "changed_by_user_id"=>null,
    "before_state"=>["status"=>"withdrawn","withdrawnAt"=>$oldWithdrawal],
    "after_state"=>["status"=>"active","withdrawnAt"=>null],
    "reason"=>"Owner-approved exact original ID restore in isolated DEV: ".$operation,
    "occurred_at"=>$now,
  ]);
  App\Models\OutboxEvent::query()->create([
    "aggregate_type"=>"rating_event","aggregate_id"=>$keep,
    "event_type"=>"rating.event_moderation_changed.v1",
    "payload"=>["ratingEventId"=>$keep,"entityId"=>$id,"from"=>"withdrawn","to"=>"active",
      "maintenanceOperationId"=>$operation,"recalculateReputation"=>true,"manualScoreEdit"=>false],
    "occurred_at"=>$now,"attempts"=>0,
  ]);
  $final=App\Models\RatingEvent::query()->where("target_entity_id",$id)->orderBy("id")->get();
  if($final->count()!==54||$final->where("status","active")->count()!==1||
     $final->firstWhere("status","active")?->id!==$keep||
     $final->where("status","removed")->count()!==47||
     $final->where("status","withdrawn")->count()!==6||
     $final->where("audience","public")->where("status","active")->count()!==1)
    throw new RuntimeException("Final 1-active/53-inactive invariant failed");
  $retained=App\Models\RatingEvent::query()->with(["review","reasonSelections","evidence"])->findOrFail($keep);
  if($retained->withdrawn_at!==null||$retained->submitted_at?->format("Y-m-d H:i:s")!=="2026-08-22 13:50:12"||
     hash("sha256",(string)$retained->review?->body)!=="aa2bcc4c7b1bf6348c46944f1bebd54298d661e4e2db030c08170a99b7d98e2e"||
     $retained->reasonSelections->count()!==0)throw new RuntimeException("Protected record changed");
  $postFingerprint=hash("sha256",json_encode([
    $retained->id,$retained->rater_account_id,$retained->target_entity_id,$retained->rating_scheme_id,
    $retained->rating_context_id,$retained->relationship_entity_id,$retained->value,$retained->audience,
    $retained->source,$retained->legacy_rating_id,$retained->idempotency_key,$retained->one_active_per_rater,
    $retained->metadata,$retained->submitted_at?->toAtomString(),$retained->created_at?->toAtomString(),
    $retained->review->id,$retained->review->author_account_id,$retained->review->body,$retained->review->status,
    $retained->reasonSelections->toJson(),$retained->evidence->toJson(),
    $selected->revisions->toJson(),
  ],JSON_UNESCAPED_SLASHES));
  if($postFingerprint!==$protectedFingerprint)throw new RuntimeException("Original protected evidence altered");
  $postVoteRows=$db::table("rating_event_admission_votes")->whereIn("rating_event_id",$actual)->orderBy("id")->get();
  if(hash("sha256",$postVoteRows->toJson())!==$voteFingerprint)throw new RuntimeException("Admission vote evidence altered");
  if($db::table("rating_event_admission_states")->whereIn("rating_event_id",$removedIds)->where("state","removed")->count()!==47||
     $db::table("rating_event_admission_states")->where("rating_event_id",$keep)->value("state")!=="not_eligible")
    throw new RuntimeException("Canonical admission lifecycle postcondition failed");
  if(hash("sha256",$db::table("rating_events")->where("target_entity_id","!=",$id)->orderBy("id")->get()->toJson())!==$otherEntityRatingsFingerprint)
    throw new RuntimeException("Unrelated rating evidence altered");
  if(hash("sha256",$db::table("entity_profile_revisions")->where("entity_id",$id)->orderBy("id")->get()->toJson())!==$profileRevisionFingerprint||
     hash("sha256",json_encode(app(App\Services\ThingImageService::class)->publicImages($entity),JSON_UNESCAPED_SLASHES))!==$imageFingerprint)
    throw new RuntimeException("CJ profile or published photos altered");
  App\Models\OutboxEvent::query()->create([
    "aggregate_type"=>"entity","aggregate_id"=>$id,
    "event_type"=>"rating.dev_owner_original_cleanup_completed.v1",
    "payload"=>["operationId"=>$operation,"keptRatingId"=>$keep,"retiredActiveIds"=>$removedIds,
      "historicalWithdrawnCompetingCount"=>6,"admissionVotesPreserved"=>17,
      "reviewSha256"=>"aa2bcc4c7b1bf6348c46944f1bebd54298d661e4e2db030c08170a99b7d98e2e",
      "active"=>1,"inactive"=>53,"manualScoreEdit"=>false,"liveTouched"=>false],
    "occurred_at"=>$now,"attempts"=>0,
  ]);
  $result=["status"=>"applied","operationId"=>$operation,"keptId"=>$keep,"value"=>5,
   "restorationFrom"=>"withdrawn","retiredActiveCount"=>47,"historicWithdrawnCompetingCount"=>6,
   "finalActive"=>1,"finalInactive"=>53,"historyCount"=>54,"admissionVotesPreserved"=>17,
   "founderVoteHistoryPreserved"=>5,"outboxStatusChangeEvents"=>48,
   "relatedEntityRatingRecordsUnchanged"=>true,"reviewAndMediaIntact"=>true,
   "selectedAdmissionState"=>"not_eligible"];
  echo "CJ_RATING_CLEANUP_TRANSACTION_PASS=".json_encode($result,JSON_UNESCAPED_SLASHES)."\n";
  return $result;
},3);
echo "CJ_RATING_OPERATION_MODE=".$mode."\n";
echo "CJ_RATING_OPERATION_STATUS=".$success["status"]."\n";

RAT_CJ_GUARDED_LIFECYCLE_PHP
)"
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$php" "$mode"
echo RAT_CJ_CANONICAL_DEV_LIFECYCLE_SCRIPT_COMPLETED=true
