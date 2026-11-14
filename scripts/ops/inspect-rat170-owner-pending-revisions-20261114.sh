#!/usr/bin/env bash
set -euo pipefail
# Read only: exact public DEV Thing's pending image suggestions. Do not print URLs,
# note bodies, emails, login IDs or raw private evidence.
[[ $# -eq 0 ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
[[ "$(kctl -n rum-dev-isolated get ingress -o jsonpath='{range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)" == 'dev-rum.daisycloversoftware.uk' ]] || exit 78
php='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";
$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
$eid="01m7bswfh0a3evf271ye4taaf0";
$entity=$db::table("entities")->where("id",$eid)->first();
if(!$entity || $entity->canonical_name!=="The Professional Idiots") exit(78);
$rows=$db::table("entity_profile_revisions")->where("entity_id",$eid)->where("field_key","image_url")
 ->orderBy("occurred_at")->get();
$safe=[];
foreach($rows as $r){
 $text=(string)($r->proposed_value??"");
 $evidence=$r->evidence===null?[]:json_decode((string)$r->evidence,true);
 $safe[]=[
  "revisionId"=>(string)$r->id,
  "proposedValueHash12"=>substr(hash("sha256",$text),0,12),
  "proposedValueLength"=>strlen($text),
  "hasHTTPSProposal"=>str_starts_with($text,"https://"),
  "evidenceProvided"=>is_array($evidence)&&count($evidence)>0,
  "moderationStatus"=>$r->moderation_status,
  "outcome"=>$r->outcome,
  "source"=>$r->source,
  "occurredAt"=>$r->occurred_at,
  "editorMatchesFirst"=>$rows->count()>0&&(string)$r->editor_user_id===(string)$rows[0]->editor_user_id,
  "metadataPresent"=>$r->metadata!==null,
 ];
}
$pending=$rows->filter(fn($r)=>$r->outcome==="pending")->values();
$first=$pending->first();$second=$pending->skip(1)->first();
echo "RAT170_OWNER_PENDING_REVISIONS_READ_ONLY=".json_encode([
 "entityPresent"=>true,"publicPublished"=>$entity->visibility==="public"&&$entity->publication_state==="published",
 "imageRevisions"=>$safe,"pendingImageCount"=>$pending->count(),
 "firstTwoPendingSameNormalisedValue"=>$first&&$second?trim((string)$first->proposed_value)===trim((string)$second->proposed_value):null,
 "firstTwoPendingHaveDistinctIds"=>$first&&$second?(string)$first->id!==(string)$second->id:null,
 "noMutation"=>true,
],JSON_THROW_ON_ERROR)."\n";
';
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$php"
