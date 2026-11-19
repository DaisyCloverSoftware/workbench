#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only eligibility check on clearly synthetic pre-existing DEV QA accounts. No votes, ratings or changes.
[[ $# -eq 0 ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == 'ghcr.io/daisycloversoftware/rum-api@sha256:a23e36323b55f3bce46d5b229146e2e7a9d70bdff9e16ea366380940e0629430' ]] || exit 78
php="$(cat <<'RATCJ_SYNTHETIC_JUDGE_ELIGIBILITY_PHP'

require "vendor/autoload.php";$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$prefixes=["rumcheck","rumowner","rumrate","rumcj","rumagain","ratowner"];
$users=App\Models\User::query()->where(function($q)use($prefixes){foreach($prefixes as $p)$q->orWhere("username","like",$p."%");})
 ->where("status","active")->orderBy("created_at")->limit(80)->get();
$eligibility=app(App\Services\JudgeEligibility::class);
$features=app(App\Services\FeatureAccessService::class);
$result=[];
foreach($users as $u){
 $email=(string)$u->email;if(!str_ends_with(mb_strtolower($email),"@example.com"))continue;
 $reason=null;
 try{
  $eligibility->assertCanBrowse($u);
  if($eligibility->accountTooNew($u))throw new RuntimeException("account_too_new");
  $features->assertAllowed($u,"voting");
 }catch(Throwable $e){$reason=get_class($e);}
 $result[]=["id"=>$u->id,"username"=>$u->username,"eligible"=>$reason===null,
    "reason"=>$reason,"emailVerified"=>$u->hasVerifiedEmail(),
    "adultAccessible"=>$u->hasCurrentAdultAssurance(),"accountAgeOlder24h"=>$u->created_at?->isBefore(now()->subDay())];
}
echo "RATCJ_DEV_SYNTHETIC_JUDGE_ELIGIBILITY=".json_encode(["candidateCount"=>count($result),
 "eligible"=>array_values(array_filter($result,fn($x)=>$x["eligible"])),
 "ineligibleCount"=>count(array_filter($result,fn($x)=>!$x["eligible"]))],JSON_UNESCAPED_SLASHES)."\n";

RATCJ_SYNTHETIC_JUDGE_ELIGIBILITY_PHP
)"
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- timeout 55s php -r "$php"
echo RATCJ_SYNTHETIC_ELIGIBILITY_READONLY_COMPLETE=true
