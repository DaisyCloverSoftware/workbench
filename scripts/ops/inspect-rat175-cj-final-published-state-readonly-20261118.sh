#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only final CJ actual stored media/identity/revision integrity, plus QA rollback.
[[ $# -eq 0 ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "ghcr.io/daisycloversoftware/rum-api@sha256:76731a87574a11547938a398dc15ecf87ced56d63b6622cd50bab5836dfeee71" ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == "ghcr.io/daisycloversoftware/rum-rate-anything@sha256:3938ed73de97546fccc527f9025cd5b7b21fb56d69bd4eb9be3a7ebabffb3919" ]] || exit 78
code='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;$cid="01m07d2f9fj05f4ypdbwp087qj";
$cj=App\Models\Entity::query()->findOrFail($cid);
$p=app(App\Services\RatCommunityContributionPolicy::class);
$rev=App\Models\EntityProfileRevision::query()->where("entity_id",$cid)->where("field_key","profile_image")->get();
$media=App\Models\MediaAsset::query()->findOrFail("01m7q2fbv84465vcd0jsng1vbp");
$img=app(App\Services\ThingImageService::class)->publicImages($cj)["profile"];
if($cj->canonical_name!=="CJ Investigates"||$cj->status!=="unverified"||$p->isClaimed($cj)
 ||$cj->edit_locked_at!==null||$rev->count()!==1
 ||$rev[0]->id!=="01m7q2fdfz4kb104tbgbjr6veh"||$rev[0]->outcome!=="applied"
 ||$rev[0]->moderation_status!=="approved"||$media->status!=="ready"
 ||$media->sha256!=="55124575ce8c175d9def68e6918a269b876f5b88e4ceff7498b902011684e1a9"
 ||!is_array($img)||!str_contains($img["url"],(string)$media->id))exit(78);
$counts=$db::table("rating_events")->where("target_entity_id",$cid)->count();
if($counts!==54||$db::table("entities")->where("canonical_name","CJ Investigates")->count()!==1)exit(78);
$qa=App\Models\Entity::query()->findOrFail("01m784s54kww2vqw891mnmce71");
$qaImage=app(App\Services\ThingImageService::class)->publicImages($qa)["profile"];
$qaRevs=App\Models\EntityProfileRevision::query()->where("entity_id",$qa->id)->where("field_key","profile_image")->get();
if($qa->canonical_name!=="RAT Demo Drone Creator"||$p->isClaimed($qa)||$qaImage!==null
 ||$qaRevs->count()!==2||$qaRevs->where("source","rollback")->count()!==1)exit(78);
echo "RAT175_FINAL_PERSISTED_STATE=".json_encode(["CJ"=>["id"=>$cid,"name"=>$cj->canonical_name,
 "unclaimed"=>true,"status"=>$cj->status,"revisionId"=>$rev[0]->id,"revisionOutcome"=>$rev[0]->outcome,
 "imagePublished"=>true,"mediaId"=>$media->id,"imageRevisions"=>1,"ratings"=>$counts,"duplicateEntities"=>false],
 "QA"=>["id"=>$qa->id,"imagePublished"=>false,"profileRevisions"=>$qaRevs->count(),"rollbackPresent"=>true]],JSON_UNESCAPED_SLASHES)."\n";
';
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$code"
echo RAT175_FINAL_STORED_IMAGE_AND_QA_ROLLBACK_VERIFIED=true
