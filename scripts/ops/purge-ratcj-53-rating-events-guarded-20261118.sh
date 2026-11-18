#!/usr/bin/env bash
set -Eeuo pipefail
# Physical delete of exact owner-authorised 53 CJ DEV ratings. Foreign keys enforced, guarded serialisable transaction, idempotent.
[[ $# -eq 0 ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == 'ghcr.io/daisycloversoftware/rum-api@sha256:a23e36323b55f3bce46d5b229146e2e7a9d70bdff9e16ea366380940e0629430' ]] || exit 78
php="$(cat <<'RATCJ_HARD_DELETE_PHP'

require "vendor/autoload.php";
$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
$cj="01m07d2f9fj05f4ypdbwp087qj";
$keep="01m0mvrxv8c3s12q0w7gwmzt03";
$operationId="ratcj-hard-delete-20261118-owner-authorised-r1";
$deleteIds=["01m0fpz4zebnh4s0py9bs4kd14","01m0rbex77q0959rnnf79ndxfn","01m0ttpp2h5taqt3j8pw0qz1b3","01m0x1y1wb5mkane66yvrhwepf","01m0xdnqvt1v5nc38bd0pbt0qc","01m0xdzxty4d419q84kjx1kgj0","01m0xdpc6hb474tjtfrcnvvfe7","01m2gyq0f0kprpve1j7tcehxzt","01m2jqzv3d9r7b3yr52dg259sb","01m0rd6x8we2cwzdtzbhvj2m91","01m0rdg2dpnyjexmpc2cpqnr20","01m0rdqcz75km8v9xs8mvk9byh","01m0rdrx9rmdfczyfd5167rbqg","01m0rdwh3w9tt4jbz4bqtngdb5","01m0rnt0q8gdz64cbyhqe9fg5z","01m0rp1g2h13sws7fgzw6vgya0","01m0rp6cynyn1xxg8ckgcv4knx","01m0s1jr1zbxbpkdjcd143nb1v","01m0s1qn21ykct3at6a4f2h12n","01m0tp48gwbg6z6cgrr6jpdxdt","01m0tpcv2v3eg77kme54dpsy1j","01m0tq4w1wcmdpr0wdecq0fzs6","01m0trz11k75xsnr4bh4xfhd7w","01m0v0hzb71nz2xv3aa0yx0rb7","01m0v0ngt2rj5sxqfkxxt3p0sk","01m0v0qkxmr694fxwrjbgadaj3","01m0w8kfw3f2ggk18k59qwdr89","01m0w8prc21fg7qj52hs8tsbmd","01m0w8rrky2a3vs8s12ztbpdty","01m0way5m57z5ak3pvrr3qckyq","01m0wazw5jp2hhqaay8agj7tkh","01m0wb6kx0ab76kbthnw8x82aw","01m0wcwnyvs1gwg1c42b6hfg4f","01m0wcxv2s4813s6v62hccybm4","01m0x2bs8nv0a0x3tagm0cr5bg","01m0x7mrxb9jjm387e4qqgag33","01m0x7pg4zf99asrxfpgyerm9x","01m0xdnhar6j17g7prh26gbtev","01m0xk6256esr02wrhd0101d28","01m0xkk2ze0q8225kr61zvdgsz","01m0xsx91vy6mcv26qypbp51ba","01m0xsy6aet3014r18qv07kzmb","01m0yqkswgqhchqcb89sy19t6v","01m0yqmnbm9zmchscmt9h5v8bn","01m0yrp3mge8x3fxf09jfjmnes","01m0yrqmwfg8tedz9d02c90a02","01m0z0xvxbd80cjva4vrdwfhmp","01m0z110n52gwys2ck1h767csh","01m2gm0rbzf1apwdkp0gqanh14","01m2gm41wjc0wanpzp477rvx39","01m2gwqnvm32pp7aw91hc3ytb7","01m2gws4ex22jxbabs27b1b9pv","01m2t7yr1bdrbzdj8ycxz6exm8"];
$expectedReview="aa2bcc4c7b1bf6348c46944f1bebd54298d661e4e2db030c08170a99b7d98e2e";
if(count($deleteIds)!==53||count(array_unique($deleteIds))!==53||in_array($keep,$deleteIds,true))throw new RuntimeException("Frozen authorised IDs invalid");
$out=$db::transaction(function()use($db,$cj,$keep,$operationId,$deleteIds,$expectedReview){
 $db::statement("SET TRANSACTION ISOLATION LEVEL SERIALIZABLE");
 $db::select("SELECT pg_advisory_xact_lock(hashtext(?))",[$operationId]);
 $entity=App\Models\Entity::query()->with(["type","classificationAssignments.classification"])->whereKey($cj)->lockForUpdate()->firstOrFail();
 $primary=$entity->classificationAssignments->first(fn($x)=>(bool)$x->is_primary&&$x->valid_to===null);
 if($entity->canonical_name!=="CJ Investigates"||$entity->type?->key!=="digital_resource"||
    $entity->visibility!=="public"||$entity->publication_state!=="published"||$entity->rateability!=="open"||
    $primary?->classification?->canonical_name!=="YouTube Channel"||$entity->deleted_at!==null)
   throw new RuntimeException("CJ canonical identity/classification changed");
 if($db::table("entity_relationships")->where(fn($q)=>$q->where("source_entity_id",$cj)->orWhere("target_entity_id",$cj))->whereNull("valid_to")->count()!==0)
   throw new RuntimeException("CJ linked graph unexpectedly active");
 $original=App\Models\RatingEvent::query()->with(["review","admissionState","admissionVotes","reasonSelections"])->whereKey($keep)->lockForUpdate()->firstOrFail();
 if($original->target_entity_id!==$cj||$original->rater_account_id!=="01kzgn6a7kswt3jqjb1sp8d4x0"||
    $original->source!=="canonical_rum"||$original->status!=="active"||$original->withdrawn_at!==null||
    $original->submitted_at?->format("Y-m-d H:i:s")!=="2026-08-22 13:50:12"||
    $original->created_at?->format("Y-m-d H:i:s")!=="2026-08-22 13:50:12"||
    $original->rating_context_id!==null||$original->reasonSelections->isNotEmpty()||
    (int)$original->value!==5||$original->audience!=="public"||
    hash("sha256",(string)$original->review?->body)!==$expectedReview||
    $original->review?->status!=="visible"||$original->review?->author_account_id!==$original->rater_account_id||
    !in_array($original->admissionState?->state,["not_eligible","needs_verdict"],true)||
    $original->admissionVotes->count()!==0||
    (int)($original->admissionState?->keep_votes??-1)!==0||
    (int)($original->admissionState?->sin_bin_votes??-1)!==0||
    (int)($original->admissionState?->score??-1)!==0)
   throw new RuntimeException("Protected original rating or current admission altered");
 $selectedBefore=$db::table("rating_events")->where("id",$keep)->first();
 $reviewBefore=$db::table("rating_reviews")->where("rating_event_id",$keep)->first();
 $originalRevisionBefore=$db::table("rating_event_revisions")->where("rating_event_id",$keep)->orderBy("id")->get()->toJson();
 $selectedEventsBefore=$db::table("outbox_events")->where("aggregate_type","rating_event")->where("aggregate_id",$keep)->orderBy("id")->get()->toJson();
 $otherRatingsBefore=$db::table("rating_events")->where("target_entity_id","!=",$cj)->orderBy("id")->get()->toJson();
 $otherSnapshotsBefore=$db::table("entity_reputation_snapshots")->where("entity_id","!=",$cj)->orderBy("id")->get()->toJson();
 $editHistoryBefore=$db::table("entity_profile_revisions")->where("entity_id",$cj)->orderBy("id")->get()->toJson();
 $imageBefore=hash("sha256",json_encode(app(App\Services\ThingImageService::class)->publicImages($entity),JSON_UNESCAPED_SLASHES));
 $marker=$db::table("outbox_events")->where("aggregate_type","entity")->where("aggregate_id",$cj)
  ->where("event_type","rating.dev_permanent_cleanup_completed.v1")->where("payload->operationId",$operationId)->count();
 if($marker>1)throw new RuntimeException("Duplicate purge marker");
 if($marker===0 && $original->admissionState?->state!=="not_eligible")throw new RuntimeException("Protected original admission baseline changed");
 $current=$db::table("rating_events")->where("target_entity_id",$cj)->lockForUpdate()->orderBy("id")->get(["id","status","source","rater_account_id","legacy_rating_id"]);
 if($marker===1){
   if($current->count()!==1||$current[0]->id!==$keep||
      $db::table("rating_event_admission_states")->where("rating_event_id",$keep)->value("state")!=="needs_verdict"||
      $db::table("entity_reputation_snapshots")->where("entity_id",$cj)->count()!==0)
     throw new RuntimeException("Completed receipt does not match final state");
   return ["status"=>"already_applied","keptId"=>$keep,"deleted"=>53,"active"=>1];
 }
 $actual=$current->pluck("id")->all();sort($actual);$expected=array_merge([$keep],$deleteIds);sort($expected);
 if($actual!==$expected||$current->count()!==54||
    $current->where("id","!=",$keep)->where("status","removed")->count()!==47||
    $current->where("id","!=",$keep)->where("status","withdrawn")->count()!==6||
    $db::table("rating_events")->where("target_entity_id","!=",$cj)->count()!==80)
   throw new RuntimeException("Exact 54-rating inventory diverged");
 $deletedRows=$current->whereIn("id",$deleteIds);
 if($deletedRows->whereNotNull("legacy_rating_id")->count()!==0)throw new RuntimeException("Unexpected legacy rating pointer");
 $fk=$db::select("SELECT c.relname AS t,a.attname AS col FROM pg_constraint co JOIN pg_class c ON c.oid=co.conrelid
  JOIN pg_attribute a ON a.attrelid=c.oid AND a.attnum=co.conkey[1]
  WHERE co.contype='f' AND co.confrelid=to_regclass('rating_events') ORDER BY c.relname");
 $allowed=[
  "rating_challenges.rating_event_id","rating_dimension_values.rating_event_id",
  "rating_event_admission_states.rating_event_id","rating_event_admission_votes.rating_event_id",
  "rating_event_reasons.rating_event_id","rating_event_revisions.rating_event_id",
  "rating_evidence.rating_event_id","rating_journey_entries.parent_rating_event_id",
  "rating_journey_entries.rating_event_id","rating_reviews.rating_event_id",
  "rating_weight_snapshots.rating_event_id"
 ];
 $seen=array_map(fn($f)=>$f->t.".".$f->col,$fk);sort($seen);sort($allowed);
 if($seen!==$allowed)throw new RuntimeException("Unknown rating foreign key; stop before delete");
 $counts=[];
 foreach($fk as $f){$key=$f->t.".".$f->col;$counts[$key]=$db::table($f->t)->whereIn($f->col,$deleteIds)->count();}
 $expectedCounts=[
  "rating_challenges.rating_event_id"=>0,"rating_dimension_values.rating_event_id"=>0,
  "rating_event_admission_states.rating_event_id"=>53,"rating_event_admission_votes.rating_event_id"=>17,
  "rating_event_reasons.rating_event_id"=>52,"rating_event_revisions.rating_event_id"=>53,
  "rating_evidence.rating_event_id"=>0,"rating_journey_entries.parent_rating_event_id"=>0,
  "rating_journey_entries.rating_event_id"=>51,"rating_reviews.rating_event_id"=>41,
  "rating_weight_snapshots.rating_event_id"=>53,
 ];
 foreach($expectedCounts as $key=>$number)if(($counts[$key]??-1)!==$number)throw new RuntimeException("FK dependency set diverged: ".$key);
 if($db::table("profile_help_me_out_pins")->where("rating_kind","rating_event")->whereIn("rating_id",$deleteIds)->exists()||
    $db::table("moderation_cases")->where("object_type","rating_event")->whereIn("object_id",$deleteIds)->exists()||
    $db::table("moderation_actions")->where("target_type","rating_event")->whereIn("target_id",$deleteIds)->exists()||
    $db::table("rating_reason_votes")->whereIn("rating_event_id",$deleteIds)->exists())
   throw new RuntimeException("Unexpected cross-reference or moderation dependency");
 $journeyEntries=$db::table("rating_journey_entries")->whereIn("rating_event_id",$deleteIds)->get(["id","entity_id","rating_journey_id"]);
 if($journeyEntries->count()!==51||$journeyEntries->where("entity_id","!=",$cj)->count()!==0)
   throw new RuntimeException("Linked journey entry outside target scope");
 $foreignWeight=$db::table("rating_weight_snapshots AS w")->join("entity_reputation_snapshots AS s","s.id","=","w.reputation_snapshot_id")
  ->whereIn("w.rating_event_id",$deleteIds)->where("s.entity_id","!=",$cj)->count();
 if($foreignWeight!==0)throw new RuntimeException("Unrelated reputation snapshot references targeted rating");
 $snapshotIds=$db::table("entity_reputation_snapshots")->where("entity_id",$cj)->pluck("id")->all();
 if(count($snapshotIds)!==1||$snapshotIds[0]!=="01m7rcm1ggxycw482nvq29jjpq")throw new RuntimeException("Snapshot baseline changed");
 $outbox=$db::table("outbox_events")->where("aggregate_type","rating_event")->whereIn("aggregate_id",$deleteIds)->get(["id","event_type"]);
 if($outbox->count()!==134)throw new RuntimeException("Rating-event outbox baseline changed");
 $outboxTypes=$outbox->groupBy("event_type")->map(fn($g)=>$g->count());
 foreach(["rating.event_created.v1"=>53,"rating.event_withdrawn.v1"=>6,"rating.event_moderation_changed.v1"=>47,"rating_event.admission_voted.v1"=>28] as $type=>$n)
   if(($outboxTypes[$type]??0)!==$n)throw new RuntimeException("Unrecognised outbox event type/count");
 $reputationOutbox=$db::table("outbox_events")->where("aggregate_type","entity_reputation_snapshot")->whereIn("aggregate_id",$snapshotIds)->pluck("id")->all();
 $outboxIds=$outbox->pluck("id")->all();
 $combined=array_merge($outboxIds,$reputationOutbox);
 $receiptIds=$db::table("reputation_event_receipts")->whereIn("outbox_event_id",$combined)->pluck("id")->all();
 if(count($receiptIds)>20)throw new RuntimeException("Unexpected reputation receipt dependencies");
 $selectedVotes=$db::table("rating_event_admission_votes")->where("rating_event_id",$keep)->count();
 if($selectedVotes!==0)throw new RuntimeException("Protected original has new community votes");
 $plan=["operationId"=>$operationId,"kept"=>$keep,"deletedIds"=>$deleteIds,
  "ratingDependentRows"=>$counts,"removedOutbox"=>count($outboxIds),
  "reputationOutbox"=>count($reputationOutbox),"reputationReceipts"=>count($receiptIds),
  "deletedJourneyEntries"=>$journeyEntries->count(),"deletedSnapshots"=>count($snapshotIds),
  "protectedAdmissionFrom"=>"not_eligible","protectedAdmissionTo"=>"needs_verdict"];
 echo "RATCJ_HARD_DELETE_GUARDED_PLAN=".json_encode($plan,JSON_UNESCAPED_SLASHES)."\n";

 // Temporary, transaction-private, exact-ID exception to the immutable
 // rating evidence triggers. PostgreSQL MVCC means other sessions continue
 // seeing the unchanged functions until commit. Original definitions are
 // restored and hash-verified before this transaction may commit.
 $guardHashes=[
  "rum_rating_event_evidence_guard"=>"53fa59f03e3059cfda3f1d23da7b8053",
  "rum_rating_review_evidence_guard"=>"6caf05605fa6d3f8664d32ea69da501e",
  "rum_rating_event_revision_guard"=>"03d579a7aca171564e54c0b876f06cb2",
 ];
 $guardDefinitions=[];
 foreach($guardHashes as $name=>$sha){
  $row=$db::selectOne("SELECT pg_get_functiondef(p.oid) def,pg_get_userbyid(p.proowner) owner_name
   FROM pg_proc p WHERE p.oid=to_regprocedure(?)",["public.".$name."()"]);
  if(!$row||$row->owner_name!=="rum_app"||md5($row->def)!==$sha)
   throw new RuntimeException("Unexpected immutable trigger definition or ownership: ".$name);
  $guardDefinitions[$name]=$row->def;
 }
 $db::statement("CREATE TEMPORARY TABLE ratcj_purge_scope (rating_event_id varchar(26) PRIMARY KEY) ON COMMIT DROP");
 $db::table("ratcj_purge_scope")->insert(array_map(fn($row)=>["rating_event_id"=>$row],$deleteIds));
 if($db::table("ratcj_purge_scope")->count()!==53||
    $db::table("ratcj_purge_scope")->where("rating_event_id",$keep)->exists())
   throw new RuntimeException("Temporary deletion scope invalid");
 $db::selectOne("SELECT set_config(?, ?, true)",["rum.ratcj_rating_purge_operation",$operationId]);
 foreach($guardDefinitions as $name=>$definition){
  $column=$name==="rum_rating_event_evidence_guard"?"id":"rating_event_id";
  $allow="IF current_setting(" . "'rum.ratcj_rating_purge_operation', true)" .
    " = " . "'".$operationId."'" .
    " AND EXISTS (SELECT 1 FROM pg_temp.ratcj_purge_scope WHERE rating_event_id = OLD.".$column.")";
  if($column==="id")$allow.=" AND OLD.target_entity_id = '".$cj."'";
  $allow.=" THEN RETURN OLD; END IF;";
  if($name==="rum_rating_event_revision_guard"){
   $needle="RAISE EXCEPTION 'rating_event_revisions are append-only';";
   $replacement="IF TG_OP = 'DELETE' THEN ".$allow." END IF;\n".$needle;
  }else{
   $needle="IF TG_OP = 'DELETE' THEN";
   $replacement=$needle."\n".$allow;
  }
  if(substr_count($definition,$needle)!==1)throw new RuntimeException("Unexpected function body for ".$name);
  $patched=str_replace($needle,$replacement,$definition);
  $db::unprepared($patched);
 }
 echo "RATCJ_TRANSACTION_PRIVATE_IMMUTABILITY_SCOPE_ACTIVE=true\n";

 if(count($receiptIds)>0){
  $deletedReceipts=$db::table("reputation_event_receipts")->whereIn("id",$receiptIds)->delete();
  if($deletedReceipts!==count($receiptIds))throw new RuntimeException("Reputation receipt cleanup mismatch");
 }
 $deletedOutbox=$db::table("outbox_events")->whereIn("id",$combined)->delete();
 if($deletedOutbox!==count($combined))throw new RuntimeException("Rating-specific outbox purge mismatch");
 $deletedJourney=$db::table("rating_journey_entries")->whereIn("id",$journeyEntries->pluck("id")->all())->where("entity_id",$cj)->delete();
 if($deletedJourney!==51)throw new RuntimeException("Rating journey entry purge mismatch");
 $removedSnapshots=$db::table("entity_reputation_snapshots")->where("entity_id",$cj)->whereIn("id",$snapshotIds)->delete();
 if($removedSnapshots!==1||
    $db::table("rating_weight_snapshots")->whereIn("rating_event_id",$deleteIds)->count()!==0)
   throw new RuntimeException("Stale reputation snapshot purge failed");
 $removedRatings=$db::table("rating_events")->where("target_entity_id",$cj)->whereIn("id",$deleteIds)->delete();
 if($removedRatings!==53)throw new RuntimeException("Exact 53 physical DELETE affected wrong count");

 // No integrity trigger may remain changed when the transaction commits.
 foreach($guardDefinitions as $name=>$definition){
  $db::unprepared($definition);
  $row=$db::selectOne("SELECT md5(pg_get_functiondef(p.oid)) actual_hash
    FROM pg_proc p WHERE p.oid=to_regprocedure(?)",["public.".$name."()"]);
  if(!$row||$row->actual_hash!==$guardHashes[$name])
   throw new RuntimeException("Immutable guard restoration failed: ".$name);
 }
 $triggers=$db::table("pg_trigger")->whereIn("tgname",[
  "rum_rating_event_evidence_guard_trigger","rum_rating_review_evidence_guard_trigger",
  "rum_rating_event_revision_guard_trigger","rum_rating_dimension_value_guard_trigger"
 ])->get(["tgname","tgenabled"]);
 if($triggers->count()!==4||$triggers->where("tgenabled","!=","O")->count()!==0)
   throw new RuntimeException("Rating immutability triggers not fully enabled");
 echo "RATCJ_IMMUTABILITY_GUARDS_RESTORED_IN_TRANSACTION=true\n";

 $eventState=App\Models\RatingEventAdmissionState::query()->whereKey($keep)->lockForUpdate()->firstOrFail();
 if($eventState->state!=="not_eligible"||(int)$eventState->keep_votes!==0||
   (int)$eventState->sin_bin_votes!==0||(int)$eventState->score!==0)
   throw new RuntimeException("Protected admission state changed during purge");
 $eventState->forceFill(["state"=>"needs_verdict","keep_votes"=>0,"sin_bin_votes"=>0,"score"=>0,
   "sin_bin_entered_at"=>null,"review_ends_at"=>null])->save();
 App\Models\OutboxEvent::query()->create([
  "aggregate_type"=>"rating_event","aggregate_id"=>$keep,"event_type"=>"rating_event.admission_state_changed.v1",
  "payload"=>["ratingEventId"=>$keep,"from"=>"not_eligible","to"=>"needs_verdict",
   "source"=>"owner_authorised_dev_maintenance","sourceId"=>$operationId,"score"=>0,"manualScoreEdit"=>false],
  "occurred_at"=>now(),"attempts"=>0
 ]);
 App\Models\OutboxEvent::query()->create([
  "aggregate_type"=>"entity","aggregate_id"=>$cj,"event_type"=>"rating.dev_permanent_cleanup_completed.v1",
  "payload"=>["operationId"=>$operationId,"keptRatingId"=>$keep,"deletedCount"=>53,
    "deletedIdsSha256"=>hash("sha256",implode("|",$deleteIds)),
    "deletedAdmissionVotes"=>17,"deletedRatingOutboxEvents"=>134,"deletedJourneyEntries"=>51,
    "invalidatedReputationSnapshots"=>1,"admissionTo"=>"needs_verdict","liveTouched"=>false],
  "occurred_at"=>now(),"attempts"=>0
 ]);
 if($db::table("rating_events")->where("target_entity_id",$cj)->count()!==1||
    $db::table("rating_events")->where("target_entity_id",$cj)->value("id")!==$keep||
    $db::table("rating_event_admission_states")->where("rating_event_id",$keep)->value("state")!=="needs_verdict"||
    $db::table("rating_event_admission_votes")->where("rating_event_id",$keep)->count()!==0||
    $db::table("rating_events")->whereIn("id",$deleteIds)->count()!==0||
    $db::table("rating_event_admission_votes")->whereIn("rating_event_id",$deleteIds)->count()!==0||
    $db::table("rating_journey_entries")->whereIn("rating_event_id",$deleteIds)->count()!==0||
    $db::table("rating_weight_snapshots")->whereIn("rating_event_id",$deleteIds)->count()!==0||
    $db::table("outbox_events")->where("aggregate_type","rating_event")->whereIn("aggregate_id",$deleteIds)->count()!==0||
    $db::table("entity_reputation_snapshots")->where("entity_id",$cj)->count()!==0)
   throw new RuntimeException("Final physical deletion/canonical admission state invariant failed");
 if($db::table("rating_events")->where("target_entity_id","!=",$cj)->orderBy("id")->get()->toJson()!==$otherRatingsBefore||
    $db::table("entity_reputation_snapshots")->where("entity_id","!=",$cj)->orderBy("id")->get()->toJson()!==$otherSnapshotsBefore||
    $db::table("entity_profile_revisions")->where("entity_id",$cj)->orderBy("id")->get()->toJson()!==$editHistoryBefore||
    hash("sha256",json_encode(app(App\Services\ThingImageService::class)->publicImages($entity),JSON_UNESCAPED_SLASHES))!==$imageBefore||
    $db::table("rating_reviews")->where("rating_event_id",$keep)->first()?->body!==$reviewBefore?->body||
    $db::table("rating_event_revisions")->where("rating_event_id",$keep)->orderBy("id")->get()->toJson()!==$originalRevisionBefore)
   throw new RuntimeException("Unrelated or protected evidence changed");
 $selectedAfter=$db::table("rating_events")->where("id",$keep)->first();
 if(json_encode($selectedAfter,JSON_UNESCAPED_SLASHES)!==json_encode($selectedBefore,JSON_UNESCAPED_SLASHES))
   throw new RuntimeException("Protected original rating row changed");
 echo "RATCJ_HARD_DELETE_TRANSACTION_PASS=".json_encode([
  "operationId"=>$operationId,"deletedRatings"=>$removedRatings,"retainedId"=>$keep,
  "remainingRatings"=>1,"admissionState"=>"needs_verdict","keepVotes"=>0,
  "sinBinVotes"=>0,"netVotes"=>0,"deletedJourneyEntries"=>$deletedJourney,
  "removedOutboxEvents"=>$deletedOutbox,"invalidatedSnapshots"=>$removedSnapshots,
  "deletedAdmissionVotes"=>17,"otherThingRatingsPreserved"=>80,
  "selectedEvidencePreserved"=>true,"historicalDataNotCopied"=>true],JSON_UNESCAPED_SLASHES)."\n";
 return ["status"=>"applied","deleted"=>$removedRatings,"kept"=>$keep];
},3);
echo "RATCJ_HARD_DELETE_FINAL=".json_encode($out,JSON_UNESCAPED_SLASHES)."\n";

RATCJ_HARD_DELETE_PHP
)"
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- timeout 135s php -r "$php"
echo RATCJ_HARD_DELETE_SCRIPT_COMPLETED=true
