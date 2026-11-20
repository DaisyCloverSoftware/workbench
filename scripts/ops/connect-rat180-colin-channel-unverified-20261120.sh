#!/usr/bin/env bash
set -Eeuo pipefail
# One-off, narrowly scoped canonical graph relationship. No entity creation,
# ratings, claims, merges, migrations, schema changes or LIVE access.
[[ $# -eq 0 ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
api='ghcr.io/daisycloversoftware/rum-api@sha256:7411cc02eb6568e6c6d228b551683294bfac49a6f89c320078af5550adce5d56'
rat='ghcr.io/daisycloversoftware/rum-rate-anything@sha256:9a33ea315f52d5e14a5ece0e57895aae8d4eb4d7d7e9955110d49fc9676c64d9'
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api" ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == "$rat" ]] || exit 78
code='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
$pid="01m0mw9hkbynssr7kbc0gk534j";
$cid="01m7x6ez57zpsgbv486y9t0y9h";
$jid="01m7x6ez8pj1k53cj4y8nnw4q6";
$person=App\Models\Entity::query()->with("type")->findOrFail($pid);
$channel=App\Models\Entity::query()->with("type","classificationAssignments.classification","identifiers")->findOrFail($cid);
$journey=App\Models\RatingJourney::query()->findOrFail($jid);
if($person->canonical_name!=="Colin Furze"||$person->type?->key!=="person"||$person->visibility!=="public"||$person->publication_state!=="published")exit(78);
if($channel->canonical_name!=="Colin Furze"||$channel->type?->key!=="digital_resource"||$channel->visibility!=="public"||$channel->publication_state!=="published"||$channel->rateability!=="open")exit(78);
if(!$channel->classificationAssignments->contains(fn($a)=>$a->valid_to===null&&$a->is_primary&&$a->classification?->key==="community_788e6ae3bfcd391d111e3ca31a1332ebf92906d0d0868e7cf7bb"))exit(78);
if(!$channel->identifiers->contains(fn($i)=>$i->valid_to===null&&$i->namespace==="youtube_channel_id"&&$i->value==="UCp68_FLety0O-n9QU6phsgw"))exit(78);
if((string)$journey->root_entity_id!==$cid||(string)$journey->user_id!=="01m7x6eyapaft7xxmk4cv0vj6x")exit(78);
$personRatings=$db::table("rating_events")->where("target_entity_id",$pid)->count();
$channelRatings=$db::table("rating_events")->where("target_entity_id",$cid)->count();
if($personRatings!==0||$channelRatings!==1)exit(78);
$existing=App\Models\EntityRelationship::query()->whereNull("valid_to")
 ->where(fn($q)=>$q->where(fn($p)=>$p->where("source_entity_id",$cid)->where("target_entity_id",$pid))
 ->orWhere(fn($p)=>$p->where("source_entity_id",$pid)->where("target_entity_id",$cid)))->get();
if($existing->count()>0){
 $r=$existing->first();
 echo "RAT180_EDGE_ALREADY_EXISTS_NO_WRITE=".json_encode(["id"=>$r->id,"verificationState"=>$r->verification_state,"visibility"=>$r->visibility,"source"=>$r->source],JSON_UNESCAPED_SLASHES)."\n";
 exit($existing->count()===1&&$r->verification_state==="unverified"&&$r->visibility==="public"?0:78);
}
$actor=App\Models\User::query()->findOrFail("01m7x6eyapaft7xxmk4cv0vj6x");
$result=app(App\Services\RatCommunityConnectionService::class)->connect($actor,$channel,$person,"related_to","outgoing");
$r=$result["relationship"];
if(!$result["created"]||$r->verification_state!=="unverified"||$r->visibility!=="public"||(string)$r->source_entity_id!==$cid||(string)$r->target_entity_id!==$pid)exit(78);
echo "RAT180_UNVERIFIED_CANONICAL_EDGE_CREATED=".json_encode([
 "id"=>(string)$r->id,"sourceId"=>(string)$r->source_entity_id,"targetId"=>(string)$r->target_entity_id,
 "verificationState"=>$r->verification_state,"visibility"=>$r->visibility,
 "source"=>$r->source,"attributes"=>$r->attributes,"actorId"=>(string)$r->created_by_user_id,
 "personRatingEvents"=>$personRatings,"channelRatingEvents"=>$channelRatings],JSON_UNESCAPED_SLASHES)."\n";
';
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$code"
echo RAT180_UNVERIFIED_RELATIONSHIP_OPERATION_COMPLETE=true
