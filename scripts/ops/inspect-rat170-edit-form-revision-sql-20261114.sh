#!/usr/bin/env bash
set -euo pipefail
# Read-only canonical DEV persistence proof for one unrelated user's isolated Thing-edit QA.
[[ $# -eq 3 ]] || exit 64
user="$1"; revision="$2"; runid="$3"
[[ "$user" =~ ^[a-z0-9]{26}$ && "$revision" =~ ^[a-z0-9]{26}$ && "$runid" =~ ^[a-z0-9-]{8,42}$ ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
[[ "$(kctl -n rum-dev-isolated get ingress -o jsonpath='{range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)" == 'dev-rum.daisycloversoftware.uk' ]] || exit 78
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == 'ghcr.io/daisycloversoftware/rum-api@sha256:e0f3de7f64cc0e54d7a0a812b47eccfbd9f2c8c97f623ac2aa72749665be9983' ]] || exit 78
php='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";
$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
[$user,$revisionId,$runid]=array_slice($argv,1);
$db=Illuminate\Support\Facades\DB::class;
$root="01m7d2emwsnd0nxw6new8khbje";
$entity=$db::table("entities")->where("id",$root)->first();
$member=$db::table("users")->where("id",$user)->first(["username"]);
$rev=$db::table("entity_profile_revisions")->where("id",$revisionId)->where("entity_id",$root)->first();
$expected="DEV layout acceptance edit on an unrelated synthetic Thing only. ".$runid;
$ev=$rev&&$rev->evidence!==null?json_decode((string)$rev->evidence,true):[];
$expectedNote="Harmless supporting evidence for isolated DEV acceptance. ".$runid;
$eventCount=$db::table("rating_events")->where("target_entity_id",$root)->count();
$result=[
 "entityPublicPublished"=>(bool)($entity&&$entity->publication_state==="published"&&$entity->visibility==="public"),
 "entityNameUnchanged"=>(bool)($entity&&$entity->canonical_name==="Carmine Nebula Archive rat170-community-a01"),
 "editorDifferentFromCreator"=>(bool)($entity&&$entity->created_by_user_id!==$user),
 "syntheticContributor"=>(bool)($member&&str_starts_with($member->username,"ratlayout")),
 "descriptionPersisted"=>(bool)($entity&&$entity->description===$expected),
 "revisionEditorMatches"=>(bool)($rev&&$rev->editor_user_id===$user),
 "revisionValueMatches"=>(bool)($rev&&$rev->proposed_value===$expected),
 "revisionFieldDescription"=>(bool)($rev&&$rev->field_key==="description"),
 "revisionCommunitySource"=>(bool)($rev&&$rev->source==="community"),
 "revisionApprovedApplied"=>(bool)($rev&&$rev->moderation_status==="approved"&&$rev->outcome==="applied"),
 "evidenceUrlAndNotePersisted"=>(bool)(is_array($ev)&&count($ev)===1
  &&($ev[0]["url"]??null)==="https://example.org/rat-qa-evidence"
  &&($ev[0]["note"]??null)===$expectedNote),
 "revisionCountForContributor"=>$db::table("entity_profile_revisions")->where("entity_id",$root)->where("editor_user_id",$user)->count(),
 "rootClaimCount"=>$db::table("entity_claims")->where("entity_id",$root)->count(),
 "rootRatingEventCount"=>$eventCount,
 "noMutation"=>true,
];
echo "RAT170_EDIT_FORM_CANONICAL_SQL=".json_encode($result,JSON_THROW_ON_ERROR)."\n";
if(!$result["entityPublicPublished"]||!$result["entityNameUnchanged"]
 ||!$result["editorDifferentFromCreator"]||!$result["syntheticContributor"]
 ||!$result["descriptionPersisted"]||!$result["revisionEditorMatches"]||!$result["revisionValueMatches"]
 ||!$result["revisionFieldDescription"]||!$result["revisionCommunitySource"]||!$result["revisionApprovedApplied"]
 ||!$result["evidenceUrlAndNotePersisted"]||$result["revisionCountForContributor"]!==1
 ||$result["rootClaimCount"]!==0||$eventCount<1)exit(2);
';
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$php" -- "$user" "$revision" "$runid"
echo RAT170_EDIT_FORM_SQL_READ_ONLY_PASS=true
