#!/usr/bin/env bash
set -euo pipefail
# Creates exactly one harmless, allowlisted claimed BUSINESS in canonical DEV
# for real contribution-permission acceptance. Uses canonical Eloquent and
# BusinessEntityClaimService, not raw SQL updates, migrations, seeds, or resets.
# The approved claim is SYNTHETIC TEST VERIFICATION only; it does not prove DNS.
[[ $# -eq 2 ]] || exit 64
rid="$1"; username="$2"
[[ "$rid" =~ ^[a-z0-9-]{8,42}$ ]] || exit 64
[[ "$username" =~ ^rat169qa[a-z0-9]{6,24}$ ]] || exit 64
kctl() { if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
[[ "$(kctl -n rum-dev-isolated get ingress -o jsonpath='{.items[0].spec.rules[0].host}')" == 'dev-rum.daisycloversoftware.uk' ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get ingress -o jsonpath='{.items[0].spec.rules[0].host}')" == 'dev-rum-ra.daisycloversoftware.uk' ]] || exit 78
probe='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";
$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
[$rid,$username]=array_slice($argv,1);
$claimant=App\Models\User::query()->where("username",$username)->first();
if(!$claimant || !$claimant->hasVerifiedEmail() || !str_starts_with($username,"rat169qa")) throw new RuntimeException("Only the previous explicitly approved DEV test contributor can be the synthetic claimant.");
$name="RATDEV Claimed Business Fixture ".$rid;
$normal=app(App\Services\EntityNameNormalizer::class)->normalise($name);
$type=App\Models\EntityType::query()->where("key","business")->firstOrFail();
$entity=App\Models\Entity::query()->where("normalised_name",$normal)->first();
if(!$entity){
 $entity=App\Models\Entity::query()->create([
  "entity_type_id"=>$type->id,"canonical_name"=>$name,"normalised_name"=>$normal,
  "canonical_slug"=>"ratdev-claim-".substr(hash("sha256",$rid),0,20),
  "description"=>"Separate DEV business fixture for claimed community correction acceptance.",
  "status"=>"active","visibility"=>"public","publication_state"=>"published",
  "rateability"=>"open","reviewability"=>"closed","sensitivity"=>"normal",
  "verification_state"=>"unverified","origin"=>"user_created","completeness"=>"complete",
  "edit_policy"=>"community_governed","created_by_user_id"=>$claimant->id,
 ]);
}
if($entity->canonical_name!==$name || $entity->type?->key!=="business"
 || $entity->created_by_user_id!==$claimant->id || $entity->visibility!=="public"
 || $entity->publication_state!=="published")
 throw new RuntimeException("Fixture identity or permissions changed; refusing claim.");
$service=app(App\Services\BusinessEntityClaimService::class);
$claim=$service->current($claimant,$entity,"owner");
if(!$claim){
 $claim=$service->submit($claimant,$entity,"owner");
}
if($claim->status!=="approved"){
 if($claim->status!=="pending") throw new RuntimeException("Claim was not safely approvable.");
 $claim=$service->approve($claim,null,2);
}
$claim->refresh();
$manager=App\Models\EntityManager::query()->where("entity_id",$entity->id)->where("user_id",$claimant->id)
 ->where("status","active")->whereNull("revoked_at")->first();
if($claim->status!=="approved" || $claim->verification_level<2 || !$manager)
 throw new RuntimeException("Synthetic claim did not establish approved management authority.");
echo "RAT170_SYNTHETIC_CLAIM_FIXTURE=".json_encode([
 "id"=>$entity->id,"name"=>$name,"createdByUserId"=>$entity->created_by_user_id,
 "claimantUserId"=>$claimant->id,"approvedClaimId"=>$claim->id,
 "claimStatus"=>$claim->status,"verificationLevel"=>$claim->verification_level,
 "managerActive"=>(bool)$manager,
 "editPolicy"=>$entity->edit_policy,"syntheticDevTestClaim"=>true,
 "actualDnsVerificationProven"=>false,
 "ratingEventsAdded"=>0,"noReseedOrReset"=>true
],JSON_THROW_ON_ERROR)."\n";
'
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$probe" -- "$rid" "$username"
