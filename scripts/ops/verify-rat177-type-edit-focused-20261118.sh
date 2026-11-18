#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only exact PR177 source validation; no application/DB mutations.
[[ $# -eq 1 && "$1" == '6cf8f1e921e9f9b0c35369ab380d4103886a935e' ]] || exit 64
src="$1"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
git -C "$work" init -q
git -C "$work" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$work" fetch -q --depth=1 origin "$src"
git -C "$work" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$work" rev-parse HEAD)" == "$src" ]] || exit 78
for f in \
 apps/api/app/Services/ThingTypeChangeService.php \
 apps/api/app/Services/ThingEditService.php \
 apps/api/app/Services/RatThingTypeCatalogueService.php \
 apps/api/app/Http/Controllers/Api/V1/ThingEditController.php \
 apps/api/tests/Feature/Entity/ThingTypeEditTest.php;do
 if command -v php >/dev/null 2>&1;then php -l "$work/$f"
 else
   if command -v k3s >/dev/null 2>&1;then
     cat "$work/$f" | sudo -n k3s kubectl -n rum-dev-isolated exec -i deployment/rum-api -c php-fpm -- php -l
   else
     cat "$work/$f" | kubectl -n rum-dev-isolated exec -i deployment/rum-api -c php-fpm -- php -l
   fi
 fi
done
echo RAT177_PHP_SOURCE_LINT_PASS=true
cd "$work/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >/dev/null
npx vitest run src/ratThingTypeEditing.test.ts src/thingEditFormLayout.test.ts src/thingTypeApi.test.ts src/ratJudgeWorkspace.test.ts src/ratRatingRoute.test.ts
npx tsc -b --pretty false
echo RAT177_FOCUSED_RAT_TYPE_AND_RATING_REGRESSIONS_PASS=true
