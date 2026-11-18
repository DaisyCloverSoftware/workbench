#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# -eq 0 ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "ghcr.io/daisycloversoftware/rum-api@sha256:76731a87574a11547938a398dc15ecf87ced56d63b6622cd50bab5836dfeee71" ]] || exit 78
code='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;$id="01m07d2f9fj05f4ypdbwp087qj";
$links=$db::table("account_entity_links")->where("entity_id",$id)->get(["link_type","status","visibility","valid_to"])->map(static fn($r)=>["kind"=>$r->link_type,"status"=>$r->status,"visibility"=>$r->visibility,"active"=>$r->valid_to===null]);
$rows=$db::table("entity_relationships as r")->join("relationship_types as t","t.id","=","r.relationship_type_id")
 ->whereNull("r.valid_to")->where(fn($q)=>$q->where("r.source_entity_id",$id)->orWhere("r.target_entity_id",$id))
 ->get(["r.source_entity_id","r.target_entity_id","t.key","t.source_type_constraints","t.target_type_constraints","r.visibility"]);
$rels=$rows->map(static fn($r)=>["direction"=>$r->source_entity_id===$id?"outgoing":"incoming","key"=>$r->key,
 "sourceConstraint"=>$r->source_type_constraints,"targetConstraint"=>$r->target_type_constraints,"visibility"=>$r->visibility]);
$components=["location_components","organisation_components","product_components"];
$counts=[];foreach($components as $table)$counts[$table]=$db::table($table)->where("entity_id",$id)->count();
echo "RAT177_CJ_TRANSITION_LINKS=".json_encode(["activeAccountLinks"=>$links->where("active",true)->count(),"accountLinks"=>$links,
"activeRelationships"=>$rels,"components"=>$counts],JSON_UNESCAPED_SLASHES)."\n";
';
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$code"
echo RAT177_LINKED_IDENTITY_READONLY_PASS=true
