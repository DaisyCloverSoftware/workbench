#!/usr/bin/env bash
set -Eeuo pipefail
# PR178 bounded read-only canonical taxonomy and public linked graph inventory.
[[ $# -eq 0 ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
api="$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')"
[[ "$api" == 'ghcr.io/daisycloversoftware/rum-api@sha256:c68f9e4902b7ab71abab2cf0ea50244bf80dd93809385add84b764ff1fc34d0c' ]] || exit 78
php='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
$all=$db::table("classifications as c")->leftJoin("entity_types as t","t.id","=","c.entity_type_id")
 ->leftJoin("classifications as p","p.id","=","c.parent_id")->orderBy("t.key")->orderBy("c.canonical_name")
 ->get(["c.key","c.canonical_name","c.status","t.key as type_key","p.key as parent_key"]);
$types=$db::table("entity_types")->orderBy("key")->get(["key","display_name","status"]);
$groups=$all->groupBy(fn($c)=>$c->type_key??"unmapped")->map(fn($r)=>["count"=>$r->count(),"keys"=>$r->map(fn($c)=>["key"=>$c->key,"name"=>$c->canonical_name,"status"=>$c->status,"parent"=>$c->parent_key])->values()->all()]);
$digital=$db::table("entities as e")->join("entity_types as t","t.id","=","e.entity_type_id")
 ->leftJoin("entity_classifications as a",function($q){$q->on("a.entity_id","=","e.id")->whereNull("a.valid_to")->where("a.is_primary","=",true);})
 ->leftJoin("classifications as c","c.id","=","a.classification_id")
 ->where("t.key","digital_resource")->where("e.visibility","public")->whereNotIn("e.status",["deleted","merged"])
 ->selectRaw("coalesce(c.key, ? ) as class_key,coalesce(c.canonical_name, ?) as class_name, count(distinct e.id) as entity_count",["[none]","[none]"])
 ->groupBy("c.key","c.canonical_name")->orderByDesc("entity_count")->get();
$names=["The Professional Idiots","The Unprofessional Idiots"];
$things=$db::table("entities as e")->join("entity_types as t","t.id","=","e.entity_type_id")
 ->whereIn("e.canonical_name",$names)->select(["e.id","e.canonical_name","e.visibility","e.publication_state","e.rateability","e.status","t.key as type"])
 ->get()->map(function($e)use($db){$e->active_classes=$db::table("entity_classifications as a")->join("classifications as c","c.id","=","a.classification_id")
 ->where("a.entity_id",$e->id)->whereNull("a.valid_to")->get(["a.is_primary","c.key","c.canonical_name"]);return $e;});
$rel=[];
if($things->count()===2){
 $p=$things->first(fn($e)=>$e->canonical_name==="The Professional Idiots");
 $u=$things->first(fn($e)=>$e->canonical_name==="The Unprofessional Idiots");
 if($p&&$u)$rel=$db::table("entity_relationships as r")->join("relationship_types as t","t.id","=","r.relationship_type_id")
 ->where(function($q)use($p,$u){$q->where(fn($z)=>$z->where("r.source_entity_id",$p->id)->where("r.target_entity_id",$u->id))
 ->orWhere(fn($z)=>$z->where("r.source_entity_id",$u->id)->where("r.target_entity_id",$p->id));})
 ->get(["r.id","r.source_entity_id","r.target_entity_id","t.key as relation_type","r.visibility","r.valid_to"])->all();
}
echo "RAT_SEMANTIC_TAXONOMY=".json_encode(["types"=>$types,"classificationCount"=>$all->count(),"classificationsByKind"=>$groups,"publicDigitalResources"=>$digital,"targetThings"=>$things,"relationships"=>$rel],JSON_UNESCAPED_SLASHES)."\n";
';
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$php"
echo RAT_SEMANTIC_TAXONOMY_READONLY_PASS=true
