#!/usr/bin/env bash
set -Eeuo pipefail
# Exact-head source-only check; never changes application DB, deployed workloads or LIVE.
[[ $# -eq 1 && "$1" == 'a8aa3a5dcd8a60c154ff8e7c0b4055b50ecede18' ]] || exit 64
src="$1"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
git -C "$work" init -q
git -C "$work" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$work" fetch -q --depth=1 origin "$src"
git -C "$work" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$work" rev-parse HEAD)" == "$src" ]] || exit 78
for file in \
 apps/api/app/Services/RatCommunityContributionPolicy.php \
 apps/api/app/Services/SafeThingImageFetcher.php \
 apps/api/tests/Feature/Entity/ThingImagePublishingTest.php; do
 if command -v php >/dev/null 2>&1; then
  php -l "$work/$file" | sed "s|$work/||g"
 else
  cat "$work/$file" | { if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl -n rum-dev-isolated exec -i deployment/rum-api -c php-fpm -- php -l; else kubectl -n rum-dev-isolated exec -i deployment/rum-api -c php-fpm -- php -l; fi; }
 fi
done
echo RAT175_CJ_PHP_SYNTAX_PASS=true
cd "$work/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >/dev/null
npx vitest run src/thingEditErrorMessage.test.ts src/thingEditFormLayout.test.ts
npx tsc -b --pretty false
echo RAT175_CJ_FOCUSED_FRONTEND_TESTS_AND_TYPECHECK_PASS=true
