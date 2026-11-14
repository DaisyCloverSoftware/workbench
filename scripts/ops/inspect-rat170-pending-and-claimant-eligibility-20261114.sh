#!/usr/bin/env bash
set -euo pipefail
# Read-only, exact-scoped DEV test-fixture status. No user email, secret, writes, LIVE, or owner data.
[[ $# -eq 0 ]] || exit 64
kctl() { if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
[[ "$(kctl -n rum-dev-isolated get ingress -o jsonpath='{.items[0].spec.rules[0].host}')" == 'dev-rum.daisycloversoftware.uk' ]] || exit 78
php='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";
$app->make(Illuminate\\Contracts\\Console\\Kernel::class)->bootstrap();
$db=Illuminate\\Support\\Facades\\DB::class;
$entity=$db::table("entities")->where("id","01m7dbsjhn6a9j373va2yz42yx")->first();
$journey=$db::table("rating_journeys")->where("id","01m7dbsjjbf2b94kzq4mm1cf3c")->first();
$creator=$entity ? $db::table("users")->where("id",$entity->created_by_user_id)->first(["username","email_verified_at"]) : null;
$qa=$db::table("users")->where("username","like","rat169qa%")->orderBy("username")->limit(40)->get(["username","email_verified_at"])->map(fn($u)=>["username"=>$u->username,"emailVerified"=>$u->email_verified_at!==null])->all();
$root="01m7d2emwsnd0nxw6new8khbje"; $pending="01m7dbsjhn6a9j373va2yz42yx";
$edges=$db::table("entity_relationships")->whereNull("valid_to")->where("source_entity_id",$root)->where("target_entity_id",$pending)->count();
$ratings=$db::table("rating_events")->where("target_entity_id",$pending)->count();
echo "RAT170_TEST_FIXTURE_READ_ONLY=".json_encode([
 "pendingThingPresent"=>(bool)$entity,
 "pendingThingState"=>$entity?->publication_state,
 "pendingThingVisibility"=>$entity?->visibility,
 "creatorUsername"=>$creator?->username,
 "creatorEmailVerified"=>$creator?->email_verified_at!==null,
 "journeyPresent"=>(bool)$journey,
 "journeyState"=>$journey?->state,
 "journeyRoot"=>$journey?->root_entity_id,
 "pendingRatingEventCount"=>$ratings,
 "pendingActiveSourceConnections"=>$edges,
 "previousApprovedQaAccounts"=>$qa,
 "noMutation"=>true,
],JSON_THROW_ON_ERROR)."\\n";
';
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$php"
