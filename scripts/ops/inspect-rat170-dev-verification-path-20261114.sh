#!/usr/bin/env bash
set -euo pipefail
# Read only: isolated DEV mail-delivery readiness + synthetic-only claimant eligibility.
# Never read application logs, email addresses, verification links, tokens, credentials or LIVE.
[[ $# -eq 0 ]] || { echo 'usage: inspect-rat170-dev-verification-path-20261114.sh (no args)' >&2; exit 64; }
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
[[ "$(kctl -n rum-dev-isolated get ingress -o jsonpath='{.items[0].spec.rules[0].host}')" == 'dev-rum.daisycloversoftware.uk' ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get ingress -o jsonpath='{.items[0].spec.rules[0].host}')" == 'dev-rum-ra.daisycloversoftware.uk' ]] || exit 78
php='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";
$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$state=app(App\Services\MailDeliveryConfiguration::class)->status();
$user=App\Models\User::query()
 ->where("username","like","rat169qa%")->orderBy("username")
 ->get(["username","email_verified_at","status"]);
$accounts=$user->map(fn($v)=>[
 "username" => $v->username,
 "emailVerified" => $v->hasVerifiedEmail(),
 "active" => $v->status === "active",
 "eligibleForExistingFixture" => preg_match("/^rat169qa[a-z0-9]{6,24}$/", $v->username)===1
     && $v->hasVerifiedEmail() && $v->status === "active"
])->all();
echo "RAT170_DEV_VERIFICATION_READ_ONLY=".json_encode([
 "environmentProduction" => app()->environment("production"),
 "mailer" => $state["mailer"],
 "mailDeliveryConfigured" => $state["configured"],
 "mailDeliveryReason" => $state["reason"],
 "previewRegistrationEnabled" => (bool) config("features.rat_preview_accounts"),
 "eligibleSyntheticClaimants" => count(array_filter($accounts,fn($a)=>$a["eligibleForExistingFixture"])),
 "approvedSyntheticAccountStatuses" => $accounts,
 "noCredentialOrVerificationLinkRead" => true,
 "noMutation" => true
],JSON_THROW_ON_ERROR)."\n";
';
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$php"
for ns in rum-dev-isolated rum-rate-anything-preview; do
 capture="$(kctl -n "$ns" get service mailpit --ignore-not-found -o name)"
 [[ "$capture" == '' || "$capture" == 'service/mailpit' ]] || exit 78
 [[ -n "$capture" ]] || capture='absent'
 printf 'RAT170_MAILPIT_SERVICE_%s=%s\n' "$ns" "$capture"
done
