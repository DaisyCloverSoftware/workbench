#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only canonical service and real-record audience audit. No impersonation,
# credentials, database writes, LIVE traffic, or modified source.
[[ $# -eq 1 && "$1" == '061beefd23bd71cf5902ff2a7724ff78acb515f6' ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == 'ghcr.io/daisycloversoftware/rum-api@sha256:7411cc02eb6568e6c6d228b551683294bfac49a6f89c320078af5550adce5d56' ]] || exit 78
code='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$person=App\Models\Entity::query()->with("type")->findOrFail("01m0mw9hkbynssr7kbc0gk534j");
$channel=App\Models\Entity::query()->with("type","classificationAssignments.classification")->findOrFail("01m7x6ez57zpsgbv486y9t0y9h");
$rum=app(App\Services\RumEntityAudiencePolicy::class);
$rat=app(App\Services\RatDiscoveryPolicy::class);
$search=app(App\Services\UniversalEntitySearchService::class);
$reads=app(App\Services\UniversalEntityReadService::class);
$checks=[];
function check($name,$pass) {global $checks;if(!$pass){echo "RAT180_RUM_FAIL=".$name."\n";exit(78);} $checks[]=$name;echo "RAT180_RUM_PASS=".$name."\n";}
check("person-entity-and-public", $person->canonical_name==="Colin Furze"&&$person->type?->key==="person"&&$person->publication_state==="published"&&$person->visibility==="public");
check("channel-entity-and-public", $channel->canonical_name==="Colin Furze"&&$channel->type?->key==="digital_resource"&&$channel->publication_state==="published"&&$channel->visibility==="public");
check("RUM-person-allowed", $rum->allows($person));
check("RUM-youtube-channel-denied", !$rum->allows($channel));
check("RAT-person-allowed", $rat->allows($person));
check("RAT-channel-allowed", $rat->allows($channel));
check("RUM-person-direct-positive-control", $reads->findPublic((string)$person->id) instanceof App\Models\Entity);
check("RUM-channel-direct-policy-denial", $reads->findPublic((string)$channel->id) instanceof App\Models\Entity && !$rum->allows($channel));
$r=$search->search(query:"Colin Furze",viewer:null,limit:20);
$hits=array_map(static fn($h)=>(string)$h["entity"]->id,$r["results"]);
check("canonical-person-search-positive-control",in_array((string)$person->id,$hits,true));
$visible=array_values(array_filter($r["results"], fn($h)=>$rum->allows($h["entity"])));
$visibleIds=array_map(static fn($h)=>(string)$h["entity"]->id,$visible);
check("RUM-general-search-person-present",in_array((string)$person->id,$visibleIds,true));
check("RUM-general-search-channel-absent",!in_array((string)$channel->id,$visibleIds,true));
$person->load("outgoingRelationships.targetEntity","incomingRelationships.sourceEntity");
$rum->filterRelatedEntities($person);
$edges=$person->outgoingRelationships->concat($person->incomingRelationships)->filter(fn($e)=>
 (string)$e->source_entity_id===(string)$channel->id||(string)$e->target_entity_id===(string)$channel->id);
check("RUM-person-relationship-channel-hidden",$edges->isEmpty());
$private=App\Models\Entity::query()->with("type")->where("visibility","private")->whereHas("type",fn($q)=>$q->where("key","person"))->first();
if($private instanceof App\Models\Entity)check("private-RUM-person-not-in-RAT",$rat->allows($private)===false);
else echo "RAT180_RUM_PRIVATE_PERSON_ACTUAL_FIXTURE=ABSENT\n";
echo "RAT180_RUM_AUDIENCE_SUMMARY=".json_encode(["status"=>"PASS_SERVER_POLICY","checks"=>$checks,"personId"=>(string)$person->id,"channelId"=>(string)$channel->id,"publicSearchHits"=>count($hits),"allowedSearchHits"=>count($visibleIds)],JSON_UNESCAPED_SLASHES)."\n";
';
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$code"
