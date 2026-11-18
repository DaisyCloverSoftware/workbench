#!/usr/bin/env bash
set -Eeuo pipefail
# Guarded, audited, atomic DEV-only QA relationship closure. No entity/rating deletion.
[[ $# -eq 0 ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == 'ghcr.io/daisycloversoftware/rum-api@sha256:a23e36323b55f3bce46d5b229146e2e7a9d70bdff9e16ea366380940e0629430' ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == 'ghcr.io/daisycloversoftware/rum-rate-anything@sha256:ec11bfec432ff10d508850a0a071b3af3787bff656d2decca90eed5951cd0003' ]] || exit 78
php='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
$cj="01m07d2f9fj05f4ypdbwp087qj";
$expected=[
 "01m0rbf00dqe2c6m3pdcasskmk"=>["target"=>"01m0rbezxvrvg20b27tp3mvcmd","name"=>"RUM DEV linked check 1787523713-e513","created"=>"2026-08-23 22:22:07+00"],
 "01m0rdqk3e525xx7xnay0c9qbq"=>["target"=>"01m0rdqk1zw87q4wq7a51h35rx","name"=>"RUM DEV linked check 1787526089-4e3e","created"=>"2026-08-23 23:01:46+00"],
 "01m0rds40zqqthtn092dc5jfff"=>["target"=>"01m0rds3whewhqw8fw19pt4ss4","name"=>"RUM DEV linked check 1787526140-d04b","created"=>"2026-08-23 23:02:36+00"],
 "01m0rdwqk8wk3gsyj09b4eadp0"=>["target"=>"01m0rdwqhwchd5qzh5bgzvmetw","name"=>"RUM DEV linked check 1787526256-3a96","created"=>"2026-08-23 23:04:34+00"],
 "01m0rp1hkjep2w6wk739nc4t9w"=>["target"=>"01m0rp1hj88v6thnsms6kxgpdt","name"=>"Quillstone17875348100c22Zeta","created"=>"2026-08-24 01:27:00+00"],
 "01m0s1qnzrep92yjv1mzzn2c6d"=>["target"=>"01m0s1qnxtjekzb5gmd2afdmkw","name"=>"Quillstone17875470673812Zeta","created"=>"2026-08-24 04:51:20+00"],
 "01m0tpcvznwvn87be1f52kh86s"=>["target"=>"01m0tpcvx62yw72ph3hy9ggfn6","name"=>"Quillstone17876022847672Zeta","created"=>"2026-08-24 20:11:40+00"],
 "01m0tq4wxhzvay8jrx96q7qmgv"=>["target"=>"01m0tq4ww6baks74wwfwax6rn2","name"=>"Quillstone1787603071e9ecZeta","created"=>"2026-08-24 20:24:48+00"],
 "01m0trz2djk8ybfh92987597rj"=>["target"=>"01m0trz2azpax5revkcn56p756","name"=>"Quillstone1787604977af8bZeta","created"=>"2026-08-24 20:56:34+00"],
 "01m0v0qmv6bdbxs8dz8cp9nsq8"=>["target"=>"01m0v0qmryf26t3myet46h463c","name"=>"Quillstone17876131227c7eZeta","created"=>"2026-08-24 23:12:19+00"],
 "01m0w8rsmexg0a00ycjsbqv578"=>["target"=>"01m0w8rska67b22rdy9ge4hy1b","name"=>"Quillstone1787655101fea4Zeta","created"=>"2026-08-25 10:52:00+00"],
 "01m0wazxps8928jba931ntf0az"=>["target"=>"01m0wazxn81jsdcpzcwv7yv6js","name"=>"Quillstone1787657433967cZeta","created"=>"2026-08-25 11:30:51+00"],
 "01m0wb6n41yyp19snkam3a65p3"=>["target"=>"01m0wb6n1xzhyrz6mjck3aqnjb","name"=>"Quillstone1787657654a4fdZeta","created"=>"2026-08-25 11:34:31+00"],
 "01m0wcxw7vxhdm40vfwj5rd959"=>["target"=>"01m0wcxw1qarphbadw4qfftx7n","name"=>"Quillstone17876594647776Zeta","created"=>"2026-08-25 12:04:41+00"],
 "01m0x2bte78m7dq0eaxtm93y49"=>["target"=>"01m0x2btczmwkctxqdge2kk7bc","name"=>"Quillstone1787681937bd86Zeta","created"=>"2026-08-25 18:19:18+00"],
 "01m0x7phmqwz7sa22f1an0t7er"=>["target"=>"01m0x7phjrysr6z73s7hk1mypz","name"=>"Quillstone1787687531ae4dZeta","created"=>"2026-08-25 19:52:32+00"],
 "01m0xdpd28j8g91a74r785p4d5"=>["target"=>"01m0xdpd1kxdrb3w0dhvb9eww6","name"=>"Quillstone178769382133faZeta","created"=>"2026-08-25 21:37:19+00"],
 "01m0xkk3y97es91vzybfk11qgk"=>["target"=>"01m0xkk3xd86q8vmbpjkyef51e","name"=>"Quillstone1787700000e7deZeta","created"=>"2026-08-25 23:20:23+00"],
 "01m0xsy7cx1wtse1wm0q2hdg6a"=>["target"=>"01m0xsy7b9ceywhn1mf93q40kr","name"=>"Quillstone1787706657a8adZeta","created"=>"2026-08-26 01:11:18+00"],
 "01m0yqmp7mq1pd325bvazd68pf"=>["target"=>"01m0yqmp6dhyr5wzqpdvr61k0s","name"=>"Quillstone178773780395f3Zeta","created"=>"2026-08-26 09:50:23+00"],
 "01m0yrp4wqfkdetgchbpjxsg3n"=>["target"=>"01m0yrp4v04bnsczfp43k5bpk6","name"=>"Quillstone17877388894e70Zeta","created"=>"2026-08-26 10:08:39+00"],
 "01m0z0xwqx6nxbpvz9xwwt3sxg"=>["target"=>"01m0z0xwprqmexgpntwcfpg2n5","name"=>"Quillstone1787747541773cZeta","created"=>"2026-08-26 12:32:42+00"],
];
$descriptions=[
 "Disposable linked Thing created only in isolated Rate Ur Mate DEV architecture verification.",
 "Disposable linked Thing created only in isolated Rate Ur Mate DEV owner-flow verification.",
 "Disposable linked Thing created only in isolated DEV to verify the LIVE-derived rating flow."
];
if(count($expected)!==22||count(array_unique(array_column($expected,"target")))!==22)throw new RuntimeException("Frozen QA inventory invalid");
$out=$db::transaction(function()use($db,$cj,$expected,$descriptions){
 $entity=App\Models\Entity::query()->with("type")->findOrFail($cj);
 if($entity->canonical_name!=="CJ Investigates"||$entity->type?->key!=="digital_resource"||$entity->deleted_at!==null)throw new RuntimeException("CJ identity mismatch");
 $rows=App\Models\EntityRelationship::query()->with(["type","sourceEntity","targetEntity.type"])
  ->where(function($q)use($cj){$q->where("source_entity_id",$cj)->orWhere("target_entity_id",$cj);})
  ->lockForUpdate()->orderBy("id")->get();
 $actual=$rows->pluck("id")->all();sort($actual);$wanted=array_keys($expected);sort($wanted);
 if($actual!==$wanted||$rows->count()!==22)throw new RuntimeException("Relationship set changed: no cleanup was applied");
 foreach($rows as $r){
  $e=$expected[$r->id]??null;$target=$r->targetEntity;
  if($e===null||$r->source_entity_id!==$cj||$r->target_entity_id!==$e["target"]||
   $r->valid_to!==null||$r->type?->key!=="related_to"||$r->source!=="rating_journey"||
   $r->provenance!=="rating_journey"||$r->visibility!=="public"||
   $r->verification_state!=="unverified"||$r->created_at?->format("Y-m-d H:i:s")!==substr($e["created"],0,19)||
   $r->sourceEntity?->canonical_name!=="CJ Investigates"||
   $target?->canonical_name!==$e["name"]||$target?->type?->key!=="product"||
   $target?->origin!=="user_created"||$target?->status!=="active"||
   $target?->visibility!=="public"||$target?->publication_state!=="published"||
   !in_array($target?->description,$descriptions,true)||
   $target?->created_by_user_id!==$r->created_by_user_id||
   $target?->created_at?->format("Y-m-d H:i:s")!==$r->created_at?->format("Y-m-d H:i:s")||
   !is_array($r->attributes)||empty($r->attributes["ratingJourneyId"]))
   throw new RuntimeException("QA evidence guard failed for relationship ".$r->id);
 }
 $entityIds=array_merge([$cj],array_column($expected,"target"));
 $entitiesBefore=$db::table("entities")->whereIn("id",$entityIds)->orderBy("id")->get()->toJson();
 $ratingsBefore=$db::table("rating_events")->where("target_entity_id",$cj)->orderBy("id")->get()->toJson();
 $revisionsBefore=$db::table("entity_profile_revisions")->where("entity_id",$cj)->orderBy("id")->get()->toJson();
 if($db::table("rating_events")->where("target_entity_id",$cj)->count()!==54)throw new RuntimeException("CJ rating baseline mismatch");
 $service=app(App\Services\EntityRelationshipService::class);$endedIds=[];
 foreach($rows as $r){$ended=$service->end($r);if($ended->valid_to===null)throw new RuntimeException("Failed to end ".$r->id);$endedIds[]=$r->id;}
 $active=$db::table("entity_relationships")->where(function($q)use($cj){$q->where("source_entity_id",$cj)->orWhere("target_entity_id",$cj);})->whereNull("valid_to")->count();
 if($active!==0||$db::table("entity_relationships")->whereIn("id",$endedIds)->whereNotNull("valid_to")->count()!==22)throw new RuntimeException("Postcondition active relationships remain");
 if($db::table("outbox_events")->where("aggregate_type","entity_relationship")->whereIn("aggregate_id",$endedIds)->where("event_type","entity.relationship_ended.v1")->count()!==22)throw new RuntimeException("Missing relationship-ended audit events");
 if($db::table("entities")->whereIn("id",$entityIds)->orderBy("id")->get()->toJson()!==$entitiesBefore||
  $db::table("rating_events")->where("target_entity_id",$cj)->orderBy("id")->get()->toJson()!==$ratingsBefore||
  $db::table("entity_profile_revisions")->where("entity_id",$cj)->orderBy("id")->get()->toJson()!==$revisionsBefore)
  throw new RuntimeException("Preservation guard failed");
 return ["endedCount"=>count($endedIds),"endedIds"=>$endedIds,"activeRemaining"=>$active,
  "entityRowsPreserved"=>count($entityIds),"cjRatingEventsPreserved"=>54,"profileRevisionsPreserved"=>true,
  "relationshipEndedOutboxEvents"=>22];
},3);
echo "CJ_QA_RELATIONSHIP_END_RESULT=".json_encode($out,JSON_UNESCAPED_SLASHES)."\n";
'
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$php"
echo CJ_QA_RELATIONSHIP_END_COMPLETE=true
