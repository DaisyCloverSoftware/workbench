#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 0 ]]||exit 64
ops=$(cd "$(dirname "$0")" && pwd);for f in accept-rat181-isolated-browser.sh rat181-fixture-probe-20261122.sh deploy-rat181-linked-things-isolated.sh;do bash -n "$ops/$f";done;node --check "$ops/rat181-acceptance-20261122.mjs"
k(){ if command -v k3s >/dev/null;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; };[[ $(k -n rum-dev-isolated get ingress -o 'jsonpath={.items[*].spec.rules[*].host}') == dev-rum.daisycloversoftware.uk ]] || exit 78
k -n rum-dev-isolated exec -i deployment/rum-api -c php-fpm -- php <<'PHP'
<?php
require 'vendor/autoload.php';$app=require 'bootstrap/app.php';$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();echo 'FIXTURE_POLICY='.json_encode(['environment'=>config('app.env'),'recognisedFixtureEmailAllowed'=>!App\Support\TestFixtureAccount::isBlockedEnvironment(),'ratPreviewAccounts'=>(bool)config('features.rat_preview_accounts')]).PHP_EOL;
PHP
printf 'SHELL_AND_JS_SYNTAX_PASS\n'
