#!/usr/bin/env bash
set -euo pipefail
# Strictly read-only canonical DEV SQL verification of a separate public test
# Thing edited and connected by a different authenticated DEV test account.
[[ $# -eq 4 ]] || exit 64
for id in "$@"; do [[ "$id" =~ ^[a-z0-9]{26}$ ]] || exit 64; done
root="$1"; target="$2"; contributor="$3"; edge="$4"
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
hosts="$(kctl -n rum-dev-isolated get ingress -o jsonpath='{range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)"
[[ "$hosts" == 'dev-rum.daisycloversoftware.uk' ]] || exit 78
probe='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";
$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
[$root,$target,$user,$edge]=array_slice($argv,1);
$db=Illuminate\Support\Facades\DB::class;
$r=$db::table("entities")->where("id",$root)->first(["publication_state","visibility","created_by_user_id","description"]);
$t=$db::table("entities")->where("id",$target)->first(["publication_state","visibility"]);
$e=$db::table("entity_relationships")->where("id",$edge)->whereNull("valid_to")->first();
$direct=$db::table("entity_profile_revisions")->where("entity_id",$root)
 ->where("editor_user_id",$user)->where("outcome","applied")->where("moderation_status","approved")->count();
$images=$db::table("entity_profile_revisions")->where("entity_id",$root)
 ->where("editor_user_id",$user)->where("field_key","image_url")
 ->where("moderation_status","pending")->where("outcome","pending")->count();
$pair=$db::table("entity_relationships")->whereNull("valid_to")
 ->where("relationship_type_id",$e?->relationship_type_id)
 ->where(function($q)use($root,$target){$q->where(fn($w)=>$w->where("source_entity_id",$root)->where("target_entity_id",$target))
  ->orWhere(fn($w)=>$w->where("source_entity_id",$target)->where("target_entity_id",$root));})->count();
$result=[
 "rootPublicPublished"=>(bool)($r&&$r->publication_state==="published"&&$r->visibility==="public"),
 "differentFromCreator"=>(bool)($r&&$r->created_by_user_id!==$user),
 "targetPublicPublished"=>(bool)($t&&$t->publication_state==="published"&&$t->visibility==="public"),
 "directApprovedRevisionsFromContributor"=>$direct,
 "pendingUnpublishedImageSuggestions"=>$images,
 "relationshipPresent"=>(bool)$e,
 "relationshipSourceIsRoot"=>(bool)($e&&$e->source_entity_id===$root),
 "relationshipTargetIsIntended"=>(bool)($e&&$e->target_entity_id===$target),
 "relationshipUnverified"=>(bool)($e&&$e->verification_state==="unverified"),
 "relationshipPublic"=>(bool)($e&&$e->visibility==="public"),
 "provenanceIsContributor"=>(bool)($e&&$e->created_by_user_id===$user&&$e->provenance==="user_submitted"),
 "activeConnectionsBetweenPair"=>$pair,
 "entityClaimCount"=>$db::table("entity_claims")->where("entity_id",$root)->count(),
 "rootRatingEventCount"=>$db::table("rating_events")->where("target_entity_id",$root)->count(),
];
echo "RAT170_CANONICAL_COMMUNITY_SQL=".json_encode($result,JSON_THROW_ON_ERROR)."\n";
$pass=$result["rootPublicPublished"]&&$result["targetPublicPublished"]&&$result["differentFromCreator"]
 &&$result["directApprovedRevisionsFromContributor"]>=1&&$result["pendingUnpublishedImageSuggestions"]>=1
 &&$result["relationshipSourceIsRoot"]&&$result["relationshipTargetIsIntended"]
 &&$result["relationshipUnverified"]&&$result["relationshipPublic"]&&$result["provenanceIsContributor"]
 &&$result["activeConnectionsBetweenPair"]===1&&$result["entityClaimCount"]===0;
if(!$pass)exit(2);
'
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$probe" -- "$root" "$target" "$contributor" "$edge"
echo RAT170_SQL_READ_ONLY=true
