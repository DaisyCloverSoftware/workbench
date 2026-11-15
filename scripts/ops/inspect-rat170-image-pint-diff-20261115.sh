#!/usr/bin/env bash
set -euo pipefail
# Exact-source read-only formatting diagnostic in disposable clone.
[[ $# -eq 1 && "$1" == '9b2add5ce9ce39567fc097a5b3281d2446e1dc49' ]] || exit 78
command -v composer >/dev/null || exit 69
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q
git -C "$tmp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$tmp" fetch -q --depth=1 origin "$1"
git -C "$tmp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$tmp" rev-parse HEAD)" == "$1" ]] || exit 78
cd "$tmp/apps/api"
COMPOSER_ALLOW_SUPERUSER=1 composer install --no-interaction --prefer-dist --no-progress --no-scripts --ignore-platform-reqs >/dev/null 2>&1
vendor/bin/pint app/Http/Controllers/Api/V1/Admin/AdminThingEditReviewController.php app/Http/Resources/UniversalEntityResource.php app/Services/SafeThingImageFetcher.php app/Services/ThingEditService.php app/Services/ThingImageService.php tests/Feature/Entity/ThingImagePublishingTest.php >/dev/null
cd "$tmp"
echo "RAT170_EXACT_SOURCE=$1"
echo "RAT170_PINT_DIFF_BEGIN"
git --no-pager diff --unified=1 -- apps/api/app/Http/Controllers/Api/V1/Admin/AdminThingEditReviewController.php apps/api/app/Http/Resources/UniversalEntityResource.php apps/api/app/Services/SafeThingImageFetcher.php apps/api/app/Services/ThingEditService.php apps/api/app/Services/ThingImageService.php apps/api/tests/Feature/Entity/ThingImagePublishingTest.php
echo "RAT170_PINT_DIFF_END"
