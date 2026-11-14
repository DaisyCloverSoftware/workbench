#!/usr/bin/env bash
set -euo pipefail
[[ $# -eq 1 && "$1" =~ ^[a-z0-9]{26}$ ]] || exit 64
id="$1"
kctl() { if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
[[ "$(kctl -n rum-dev-isolated get ingress -o jsonpath='{.items[0].spec.rules[0].host}')" == 'dev-rum.daisycloversoftware.uk' ]] || exit 78
probe='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";
$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$id=$argv[1];
$entity=App\Models\Entity::query()->with("type")->find($id);
if(!$entity) {echo "RAT170_ENTITY_ELIGIBILITY={\"entity_present\":false}\n";exit(2);}
$discovery=app(App\Services\RatDiscoveryPolicy::class);
$policy=app(App\Services\RatCommunityContributionPolicy::class);
$assertPublic=true;$reason="";
try{$policy->assertPublicThing($entity);}catch(\Throwable $e){$assertPublic=false;$reason=get_class($e);}
echo "RAT170_ENTITY_ELIGIBILITY=".json_encode([
 "entity_present"=>true,
 "id"=>$entity->id,
 "canonical_name"=>$entity->canonical_name,
 "status"=>$entity->status,
 "visibility"=>$entity->visibility,
 "publication_state"=>$entity->publication_state,
 "type"=>$entity->type?->key,
 "edit_policy"=>$entity->edit_policy,
 "has_creator"=>($entity->created_by_user_id!==null),
 "discovery_allows"=>$discovery->allows($entity),
 "contribution_policy_allows"=>$assertPublic,
 "policy_failure_class"=>$reason,
 "claims_count"=>$entity->claims()->count(),
],JSON_THROW_ON_ERROR)."\n";
'
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$probe" -- "$id"
echo 'RAT170_ENTITY_DIAGNOSTIC_READ_ONLY=true'
