#!/usr/bin/env bash
set -euo pipefail
# Read-only follow-up to PR169 exact-head browser acceptance, isolated DEV only.
# No passwords, token reads, personal/third-party identities, data writes,
# migrations, reseeds, deployments, or production access.
[[ $# -eq 4 ]] || { echo 'usage: <entity-id> <classification-id> <classification-key> <journey-id>' >&2; exit 64; }
entity="$1"; classid="$2"; key="$3"; journey="$4"
[[ "$entity" =~ ^[0-9a-z]{26}$ && "$classid" =~ ^[0-9a-z]{26}$ && "$key" =~ ^[a-z0-9_-]{2,100}$ && "$journey" =~ ^[0-9a-z]{26}$ ]] || exit 64
kctl() { if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
[[ "$(kctl -n rum-dev-isolated get ingress -o jsonpath='{.items[0].spec.rules[0].host}')" == 'dev-rum.daisycloversoftware.uk' ]] || exit 78
[[ "$(kctl -n rum-dev-isolated get configmap rum-config -o jsonpath='{.data.APP_URL}')" == 'https://dev-rum.daisycloversoftware.uk' ]] || exit 78
probe='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";
$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
[$eid,$cid,$key,$jid]=array_slice($argv,1);
$db=Illuminate\Support\Facades\DB::class;
$e=$db::table("entities")->where("id",$eid)->first();
$c=$db::table("classifications")->where("id",$cid)->first();
$ck=$db::table("classifications")->where("key",$key)->first();
$a=$db::table("entity_classifications")->where("entity_id",$eid)->get();
$j=$db::table("rating_journeys")->where("id",$jid)->where("root_entity_id",$eid)->first();
$et=$db::table("entity_types")->where("key","digital_resource")->first();
$location=$db::table("entity_types")->where("key","business_location")->first();
$org=$db::table("entity_types")->where("key","organisation")->first();
$businessKeys=["local_business","food_drink","pub","bar","cafe","coffee_shop","restaurant","retail","supermarket","pharmacy","garage","hotel"];
$businessRows=$db::table("classifications")->whereIn("key",$businessKeys)->get(["key","entity_type_id"]);
$wrongBusiness=$businessRows->filter(fn($row) => !$location || (string)$row->entity_type_id!==(string)$location->id)->pluck("key")->values()->all();
$collective=$db::table("classifications")->where("key","creator_collective")->first();
$assignedById=$a->first(fn($row)=>(string)$row->classification_id===$cid);
$assignedByKey=$ck ? $a->first(fn($row)=>(string)$row->classification_id===(string)$ck->id) : null;
$assignedIds=$a->pluck("classification_id")->all();
$joined=$db::table("entity_classifications as ec")->join("classifications as c","c.id","=","ec.classification_id")->where("ec.entity_id",$eid)->get(["c.key","c.entity_type_id","ec.is_primary","ec.verification_state"]);
$out=[
 "actual_namespace"=>"rum-dev-isolated",
 "entity_exists"=>(bool)$e,
 "entity_published"=>($e && $e->publication_state==="published"),
 "entity_structural_kind_correct"=>(bool)($e && $et && (string)$e->entity_type_id===(string)$et->id),
 "classification_found_by_created_id"=>(bool)$c,
 "classification_found_by_created_key"=>(bool)$ck,
 "classification_id_matches_key"=>(bool)($c && $ck && (string)$c->id===(string)$ck->id),
 "classification_key_at_id"=>$c ? (string)$c->key : null,
 "expected_classification_key"=>$key,
 "classification_kind_correct"=>(bool)($c && $et && (string)$c->entity_type_id===(string)$et->id),
 "assignment_count"=>$a->count(),
 "assigned_classification_ids"=>$assignedIds,
 "assignment_to_created_id"=>(bool)$assignedById,
 "assignment_to_created_key"=>(bool)$assignedByKey,
 "assignment_to_created_id_primary"=>($assignedById ? (bool)$assignedById->is_primary:false),
 "joined_assignments"=>$joined->toArray(),
 "journey_exists"=>(bool)$j,
 "journey_state"=>$j ? (string)$j->state : null,
 "rating_event_count"=>$db::table("rating_events")->where("target_entity_id",$eid)->count(),
 "business_mappings_present"=>$businessRows->count(),
 "business_mappings_wrong"=>$wrongBusiness,
 "creator_collective_to_organisation"=>(bool)($collective && $org && (string)$collective->entity_type_id===(string)$org->id),
 "migration_recorded"=>$db::table("migrations")->where("migration","2026_11_13_000001_map_thing_classifications_to_entity_types")->exists(),
 "classification_count"=>$db::table("classifications")->count(),
];
echo "RAT169_CANONICAL_DEV_DEEP_READBACK=".json_encode($out,JSON_THROW_ON_ERROR)."\n";
'
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$probe" -- "$entity" "$classid" "$key" "$journey"
echo 'RAT169_DB_QUERY_READ_ONLY=true'
