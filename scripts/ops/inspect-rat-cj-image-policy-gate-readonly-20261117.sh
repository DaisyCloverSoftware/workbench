#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only isolated DEV proof: exact CJ controller policy exception and safe host eligibility.
[[ $# -eq 0 ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
expected='ghcr.io/daisycloversoftware/rum-api@sha256:5fc614b352a596e41d35a33f55b9f4b01d4c6c2bfbe2d7dda700d92267761a13'
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$expected" ]] || exit 78
code='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$e=App\Models\Entity::query()->findOrFail("01m07d2f9fj05f4ypdbwp087qj");
try{app(App\Services\RatCommunityContributionPolicy::class)->assertPublicThing($e);
 echo "CJ_CURRENT_POLICY_GATE=allow\n";
}catch(Throwable $error){
 $status=($error instanceof Symfony\Component\HttpKernel\Exception\HttpExceptionInterface)?$error->getStatusCode():null;
 echo "CJ_CURRENT_POLICY_GATE=".json_encode(["type"=>get_class($error),"status"=>$status,"message"=>$error->getMessage()])."\n";
}
try{$host=app(App\Services\SafeThingImageFetcher::class)->validate("https://yt3.googleusercontent.com/placeholder.jpg");
 echo "CJ_GOOGLE_PUBLIC_HOST_VALIDATION=accepted host=".$host."\n";
}catch(Throwable $error){echo "CJ_GOOGLE_PUBLIC_HOST_VALIDATION=reject type=".get_class($error)."\n";}
'
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$code"
echo CJ_POLICY_GATE_PROBE_READONLY=true
