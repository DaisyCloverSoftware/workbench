#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only source validation pinned to current PR #179 HEAD; no DEV DB writes, build/deploy, or LIVE changes.
[[ $# -eq 1 && "$1" =~ ^[0-9a-f]{40}$ ]] || exit 64
src="$1"
tmp="$(mktemp -d)";trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q
git -C "$tmp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$tmp" fetch -q --depth=1 origin "$src"
git -C "$tmp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$tmp" rev-parse HEAD)" == "$src" ]] || exit 78
for file in \
 apps/api/app/Http/Controllers/Api/V1/RatCatalogueController.php \
 apps/api/app/Services/RatPublicRatingsService.php \
 apps/api/app/Services/RatThingJudgeService.php \
 apps/api/app/Services/CanonicalJudgeService.php \
 apps/api/app/Services/Reputation/RatingWeightService.php \
 apps/api/app/Services/Reputation/ReputationOutboxDispatcher.php \
 apps/api/app/Services/Reputation/PublicEntityReputationSummaryService.php \
 apps/api/tests/Feature/RatingEvents/RatPublicRatingCardsTest.php;do
 if command -v php >/dev/null 2>&1;then
  php -l "$tmp/$file" | sed "s|$tmp/||"
 else
  if command -v k3s >/dev/null 2>&1;then
   sudo -n k3s kubectl -n rum-dev-isolated exec -i deployment/rum-api -c php-fpm -- php -l < "$tmp/$file"
  else kubectl -n rum-dev-isolated exec -i deployment/rum-api -c php-fpm -- php -l < "$tmp/$file";fi
 fi
done
echo RATCJ_PR179_EXACT_SOURCE_PHP_SYNTAX_PASS=true
cd "$tmp/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >/dev/null
npx vitest run src/ratPublicRatings.test.ts
npx tsc -b --pretty false
echo RATCJ_PR179_EXACT_SOURCE_FRONTEND_TYPE_AND_FOCUSED_TESTS_PASS=true
