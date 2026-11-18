#!/usr/bin/env bash
set -Eeuo pipefail
# Exact-image, DEV-only, read-only canonical relationship and fixture provenance audit.
[[ $# -eq 0 ]] || { echo "No arguments accepted"; exit 64; }
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
api='ghcr.io/daisycloversoftware/rum-api@sha256:a23e36323b55f3bce46d5b229146e2e7a9d70bdff9e16ea366380940e0629430'
current="$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')"
[[ "$current" == "$api" ]] || { echo "Refusing: wrong DEV API image"; exit 78; }
php='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
$id="01m07d2f9fj05f4ypdbwp087qj";
$cj=$db::table("entities")->where("id",$id)->first(["id","canonical_name","origin","status","publication_state","deleted_at"]);
if(!$cj||$cj->canonical_name!=="CJ Investigates"||$cj->deleted_at!==null)exit(78);
$r=$db::table("entity_relationships as r")
 ->join("relationship_types as rt","rt.id","=","r.relationship_type_id")
 ->leftJoin("entities as se","se.id","=","r.source_entity_id")
 ->leftJoin("entities as te","te.id","=","r.target_entity_id")
 ->leftJoin("entity_types as st","st.id","=","se.entity_type_id")
 ->leftJoin("entity_types as tt","tt.id","=","te.entity_type_id")
 ->where(function($q)use($id){$q->where("r.source_entity_id",$id)->orWhere("r.target_entity_id",$id);})
 ->orderBy("r.created_at")->orderBy("r.id")
 ->get(["r.id","r.source_entity_id","r.target_entity_id","r.relationship_type_id","rt.key as relationship_type",
 "rt.is_symmetric","r.valid_from","r.valid_to","r.verification_state","r.visibility","r.source",
 "r.provenance","r.confidence","r.attributes","r.created_by_user_id","r.created_at","r.updated_at",
 "se.canonical_name as source_name","se.origin as source_origin","se.created_at as source_created_at",
 "st.key as source_type","te.canonical_name as target_name","te.origin as target_origin",
 "te.created_at as target_created_at","tt.key as target_type"]);
if($r->count()>500)exit(79);
$otherIds=$r->map(fn($x)=>$x->source_entity_id===$id?$x->target_entity_id:$x->source_entity_id)->unique()->values()->all();
$entities=$db::table("entities as e")->leftJoin("entity_types as t","t.id","=","e.entity_type_id")
 ->whereIn("e.id",$otherIds)->orderBy("e.id")
 ->get(["e.id","e.canonical_name","e.description","e.origin","e.status","e.visibility","e.publication_state",
 "e.verification_state","e.created_by_user_id","e.created_at","e.updated_at","e.deleted_at","t.key as type"]);
$identifiers=$db::table("entity_identifiers")->whereIn("entity_id",$otherIds)->orderBy("entity_id")->orderBy("id")
 ->get(["id","entity_id","namespace","value","source","verification_state","valid_from","valid_to","created_at"]);
$events=$db::table("outbox_events")->where("aggregate_type","entity_relationship")
 ->whereIn("aggregate_id",$r->pluck("id")->all())->orderBy("occurred_at")
 ->get(["aggregate_id","event_type","payload","occurred_at","created_at"]);
if($identifiers->count()>500||$events->count()>1000)exit(79);
echo "CJ_QA_RELATIONSHIP_AUDIT=".json_encode([
 "environment"=>"rum-dev-isolated","entity"=>$cj,"totalRelationships"=>$r->count(),
 "activeRelationships"=>$r->whereNull("valid_to")->count(),"relationships"=>$r->values(),
 "relatedEntities"=>$entities,"identifiers"=>$identifiers,"relationshipEvents"=>$events,
 "cjRatingEvents"=>$db::table("rating_events")->where("target_entity_id",$id)->count()
],JSON_UNESCAPED_SLASHES|JSON_INVALID_UTF8_SUBSTITUTE)."\n";
';
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$php"
echo CJ_QA_RELATIONSHIP_AUDIT_COMPLETE=true
