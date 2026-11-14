#!/usr/bin/env bash
set -euo pipefail
# Disposable exact-source Pint diagnostic. No cluster, application DB or repo mutation.
[[ $# -eq 1 && "$1" == "7cd4c2b12a5e26629c51b7351c4d1997182432f0" ]] || exit 78
command -v composer >/dev/null || { echo 'COMPOSER_NOT_AVAILABLE'; exit 69; }
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q
git -C "$tmp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$tmp" fetch -q --depth=1 origin "$1"
git -C "$tmp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$tmp" rev-parse HEAD)" == "$1" ]] || exit 78
cd "$tmp/apps/api"
COMPOSER_ALLOW_SUPERUSER=1 composer install --no-interaction --prefer-dist --no-progress --no-scripts --ignore-platform-reqs >/dev/null 2>&1
vendor/bin/pint app/Services/ThingEditService.php app/Http/Controllers/Api/V1/ThingEditController.php app/Http/Controllers/Api/V1/Admin/AdminThingEditReviewController.php tests/Feature/Entity/ThingEditModerationJourneyTest.php >/dev/null
cd "$tmp"
printf 'RAT170_PINT_EXACT_SOURCE=%s\n' "$1"
printf 'RAT170_PINT_CHANGED_FILES\n'
git diff --name-only -- apps/api/app/Services/ThingEditService.php apps/api/app/Http/Controllers/Api/V1/ThingEditController.php apps/api/app/Http/Controllers/Api/V1/Admin/AdminThingEditReviewController.php apps/api/tests/Feature/Entity/ThingEditModerationJourneyTest.php
printf 'RAT170_PINT_DIFF_BEGIN\n'
git --no-pager diff --unified=2 -- apps/api/app/Services/ThingEditService.php apps/api/app/Http/Controllers/Api/V1/ThingEditController.php apps/api/app/Http/Controllers/Api/V1/Admin/AdminThingEditReviewController.php apps/api/tests/Feature/Entity/ThingEditModerationJourneyTest.php
printf 'RAT170_PINT_DIFF_END\n'
