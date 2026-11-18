#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only canonical identity, relationship, classification and CJ history audit.
[[ $# -eq 1 && "$1" == '__RAT178_API_IMAGE_PENDING__' ]] || exit 78
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$1" ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == 'ghcr.io/daisycloversoftware/rum-rate-anything@sha256:ec11bfec432ff10d508850a0a071b3af3787bff656d2decca90eed5951cd0003' ]] || exit 78
php='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
$ids=["01m7bswfh0a3evf271ye4taaf0"=>"The Professional Idiots","01m7jss48cpff8c59j6mdbbb7f"=>"The Unprofessional Idiots"];
$key="community_788e6ae3bfcd391d111e3ca31a1332ebf92906d0d0868e7cf7bb";
$things=[];
foreach($ids as $id=>$name){
 $e=App\Models\Entity::query()->with(["type","classificationAssignments.classification"])->findOrFail($id);
 $primary=$e->classificationAssignments->first(fn($a)=>(bool)$a->is_primary&&$a->valid_to===null);
 if($e->canonical_name!==$name||$e->type?->key!=="digital_resource"||$e->visibility!=="public"||$e->publication_state!=="published"||$e->rateability!=="open"||$primary?->classification?->key!==$key||$db::table("entities")->where("canonical_name",$name)->count()!==1)exit(78);
 $things[]=["id"=>$id,"name"=>$name,"primary"=>$primary->classification->canonical_name];
}
$rel=$db::table("entity_relationships as r")->join("relationship_types as t","t.id","=","r.relationship_type_id")->where("r.id","01m7jswzw09b94n00am5003dej")->first(["r.source_entity_id","r.target_entity_id","r.visibility","r.valid_to","t.key"]);
if(!$rel||$rel->source_entity_id!=="01m7bswfh0a3evf271ye4taaf0"||$rel->target_entity_id!=="01m7jss48cpff8c59j6mdbbb7f"||$rel->visibility!=="public"||$rel->valid_to!==null||$rel->key!=="related_to")exit(78);
$cj=App\Models\Entity::query()->with("type")->findOrFail("01m07d2f9fj05f4ypdbwp087qj");
$rev=$db::table("entity_profile_revisions")->where("entity_id",$cj->id)->where("field_key","thing_type")->get(["id","outcome","editor_user_id"]);
$pics=app(App\Services\ThingImageService::class)->publicImages($cj);
if($cj->canonical_name!=="CJ Investigates"||$cj->type?->key!=="digital_resource"||$db::table("rating_events")->where("target_entity_id",$cj->id)->count()!==54||$rev->count()!==1||$rev[0]->id!=="01m7qsxdjs1tg298py83xk2y49"||$rev[0]->outcome!=="applied"||!$rev[0]->editor_user_id||$pics["profile"]===null||$pics["cover"]===null)exit(78);
echo "RAT178_CANONICAL_FINAL=".json_encode(["channels"=>$things,"relationId"=>"01m7jswzw09b94n00am5003dej","relationPreserved"=>true,"cjType"=>"digital_resource","cjTypeRevision"=>$rev[0]->id,"cjRatingEvents"=>54,"cjProfileCoverPreserved"=>true],JSON_UNESCAPED_SLASHES)."\n";
';
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$php"
echo RAT178_READONLY_CANONICAL_PRESERVATION_PASS=true
