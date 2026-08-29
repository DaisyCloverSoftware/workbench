#!/usr/bin/env bash
set -euo pipefail

RUM_REPO="https://github.com/DaisyCloverSoftware/rum.git"
TARGET_BRANCH="sprint-1-rat-foundation-search-explorer-20260826"
EXPECTED_HEAD="60cab55d5bd868913da833e60c15cbe938afa494"

RAT_BRAND_SHA="9893244c5f0c146b128ec5e09d22daf438f96637"
SEARCH_SHA="cb2b19d0eb81a3fee3f37e6bf913fe61f8d40f0e"
EXPLORER_TEST_SHA="11441e60626f5774c759484f48eb861d819cdf1d"
PREVIEW_TEST_SHA="003bbdb2ce128131ed1cc919d2b91472148a8690"
VERIFIER_SHA="fcff5e8ef261d4191c7b5db9a2cf7f5b5fae006e"

THUMB_URL="/brand/rum-thumb.webp?v=ad234fbada71f3be"
MASCOT_URL="/brand/rat-pirate-mascot.webp?v=0c4ade813ec3e41a"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

git clone --quiet --no-checkout "$RUM_REPO" "$tmp/rum"
cd "$tmp/rum"
git fetch --quiet origin "$TARGET_BRANCH"
git checkout --quiet -B "$TARGET_BRANCH" "origin/$TARGET_BRANCH"

actual_head="$(git rev-parse HEAD)"
[[ "$actual_head" == "$EXPECTED_HEAD" ]] || {
  echo "ERROR: RAT candidate moved: expected $EXPECTED_HEAD got $actual_head" >&2
  exit 1
}

check_blob() {
  local path="$1" expected="$2" actual
  actual="$(git hash-object "$path")"
  [[ "$actual" == "$expected" ]] || {
    echo "ERROR: source blob changed for $path: expected $expected got $actual" >&2
    exit 1
  }
}

check_blob apps/rate-anything/src/RatBrand.tsx "$RAT_BRAND_SHA"
check_blob apps/rate-anything/src/SearchExperience.tsx "$SEARCH_SHA"
check_blob apps/rate-anything/src/ratExplorer.test.ts "$EXPLORER_TEST_SHA"
check_blob apps/rate-anything/src/previewIsolation.test.ts "$PREVIEW_TEST_SHA"
check_blob scripts/verify-rate-anything-preview-ui.sh "$VERIFIER_SHA"
[[ ! -e apps/rate-anything/src/brandAssets.ts ]] || {
  echo "ERROR: brandAssets.ts already exists unexpectedly" >&2
  exit 1
}

cat > apps/rate-anything/src/brandAssets.ts <<EOF
// These query keys are content identities, not release identities. The RAT preview
// serves static images with a long immutable cache lifetime, so changing an asset
// must also change its URL or an intermediary may legitimately serve old bytes.
export const RUM_THUMB_ASSET = '$THUMB_URL'
export const RAT_PIRATE_MASCOT_ASSET = '$MASCOT_URL'
EOF

python3 - <<'PY'
from pathlib import Path


def replace_exact(path: str, old: str, new: str, count: int = 1) -> None:
    p = Path(path)
    text = p.read_text(encoding='utf-8')
    actual = text.count(old)
    if actual != count:
        raise SystemExit(f'ERROR: {path}: expected {count} occurrence(s) of {old!r}, found {actual}')
    p.write_text(text.replace(old, new), encoding='utf-8')

# Route every RUM Thumb use through the content-versioned URL.
replace_exact(
    'apps/rate-anything/src/RatBrand.tsx',
    'export function RatBrand',
    "import { RUM_THUMB_ASSET } from './brandAssets'\n\nexport function RatBrand",
)
replace_exact(
    'apps/rate-anything/src/RatBrand.tsx',
    'src="/brand/rum-thumb.webp"',
    'src={RUM_THUMB_ASSET}',
    3,
)

# Route both home and results mascot uses through the same content identity.
replace_exact(
    'apps/rate-anything/src/SearchExperience.tsx',
    "import { RatBrand, RumThumbRating } from './RatBrand'\n",
    "import { RatBrand, RumThumbRating } from './RatBrand'\nimport { RAT_PIRATE_MASCOT_ASSET } from './brandAssets'\n",
)
replace_exact(
    'apps/rate-anything/src/SearchExperience.tsx',
    'src="/brand/rat-pirate-mascot.webp"',
    'src={RAT_PIRATE_MASCOT_ASSET}',
    2,
)

# Lock the cache-version contract in source tests.
replace_exact(
    'apps/rate-anything/src/ratExplorer.test.ts',
    "    const css = read('./rat-explorer.css')\n    const rendered = `${brand}\\n${experience}\\n${css}`.toLowerCase()\n\n    expect(brand).toContain('/brand/rum-thumb.webp')\n",
    "    const css = read('./rat-explorer.css')\n    const assets = read('./brandAssets.ts')\n    const rendered = `${brand}\\n${experience}\\n${css}`.toLowerCase()\n\n    expect(brand).toContain(\"import { RUM_THUMB_ASSET } from './brandAssets'\")\n    expect(brand).toContain('src={RUM_THUMB_ASSET}')\n    expect(experience).toContain(\"import { RAT_PIRATE_MASCOT_ASSET } from './brandAssets'\")\n    expect(experience).toContain('src={RAT_PIRATE_MASCOT_ASSET}')\n    expect(assets).toContain(\"RUM_THUMB_ASSET = '/brand/rum-thumb.webp?v=ad234fbada71f3be'\")\n    expect(assets).toContain(\"RAT_PIRATE_MASCOT_ASSET = '/brand/rat-pirate-mascot.webp?v=0c4ade813ec3e41a'\")\n    expect(brand).not.toContain('src=\"/brand/rum-thumb.webp\"')\n    expect(experience).not.toContain('src=\"/brand/rat-pirate-mascot.webp\"')\n",
)

replace_exact(
    'apps/rate-anything/src/previewIsolation.test.ts',
    "    const ratBrand = readProjectFile('./RatBrand.tsx')\n    const renderedSources = [searchExperience, entityExperience, managementExperience, compatibilityBrand, ratBrand].join('\\n')\n",
    "    const ratBrand = readProjectFile('./RatBrand.tsx')\n    const brandAssets = readProjectFile('./brandAssets.ts')\n    const renderedSources = [searchExperience, entityExperience, managementExperience, compatibilityBrand, ratBrand, brandAssets].join('\\n')\n",
)
replace_exact(
    'apps/rate-anything/src/previewIsolation.test.ts',
    "    expect(ratBrand).toContain('/brand/rum-thumb.webp')\n",
    "    expect(ratBrand).toContain(\"import { RUM_THUMB_ASSET } from './brandAssets'\")\n    expect(searchExperience).toContain(\"import { RAT_PIRATE_MASCOT_ASSET } from './brandAssets'\")\n    expect(brandAssets).toContain(\"RUM_THUMB_ASSET = '/brand/rum-thumb.webp?v=ad234fbada71f3be'\")\n    expect(brandAssets).toContain(\"RAT_PIRATE_MASCOT_ASSET = '/brand/rat-pirate-mascot.webp?v=0c4ade813ec3e41a'\")\n    expect(ratBrand).not.toContain('src=\"/brand/rum-thumb.webp\"')\n    expect(searchExperience).not.toContain('src=\"/brand/rat-pirate-mascot.webp\"')\n",
)

# The deployed verifier must request exactly the content-versioned URLs emitted by
# the bundle; probing the old unversioned URLs would only test a stale CDN cache key.
replace_exact(
    'scripts/verify-rate-anything-preview-ui.sh',
    'EXPECTED_MASCOT="/brand/rat-pirate-mascot.webp"\nEXPECTED_THUMB="/brand/rum-thumb.webp"',
    'EXPECTED_MASCOT="/brand/rat-pirate-mascot.webp?v=0c4ade813ec3e41a"\nEXPECTED_THUMB="/brand/rum-thumb.webp?v=ad234fbada71f3be"',
)
replace_exact(
    'scripts/verify-rate-anything-preview-ui.sh',
    '  name="$(basename "$asset")"\n  status="$(curl -sS -o "$asset_tmp/$name" -w \'%{http_code}\' \\\n    -H \'Cache-Control: no-cache, no-store\' --max-time 30 \\\n    "https://${PREVIEW_HOST}${asset}?rat-preview-verify=${cache_buster}")"',
    '  name="$(basename "${asset%%\\?*}")"\n  status="$(curl -sS -o "$asset_tmp/$name" -w \'%{http_code}\' \\\n    -H \'Cache-Control: no-cache, no-store\' --max-time 30 \\\n    "https://${PREVIEW_HOST}${asset}")"',
)
PY

# Static checks before commit.
grep -Fq "RUM_THUMB_ASSET = '$THUMB_URL'" apps/rate-anything/src/brandAssets.ts
grep -Fq "RAT_PIRATE_MASCOT_ASSET = '$MASCOT_URL'" apps/rate-anything/src/brandAssets.ts
! grep -Fq 'src="/brand/rum-thumb.webp"' apps/rate-anything/src/RatBrand.tsx
! grep -Fq 'src="/brand/rat-pirate-mascot.webp"' apps/rate-anything/src/SearchExperience.tsx
grep -Fq "EXPECTED_THUMB=\"$THUMB_URL\"" scripts/verify-rate-anything-preview-ui.sh
grep -Fq "EXPECTED_MASCOT=\"$MASCOT_URL\"" scripts/verify-rate-anything-preview-ui.sh
bash -n scripts/verify-rate-anything-preview-ui.sh

git add \
  apps/rate-anything/src/brandAssets.ts \
  apps/rate-anything/src/RatBrand.tsx \
  apps/rate-anything/src/SearchExperience.tsx \
  apps/rate-anything/src/ratExplorer.test.ts \
  apps/rate-anything/src/previewIsolation.test.ts \
  scripts/verify-rate-anything-preview-ui.sh

status="$(git status --porcelain)"
change_count="$(printf '%s\n' "$status" | sed '/^$/d' | wc -l | tr -d ' ')"
[[ "$change_count" == "6" ]] || {
  echo "ERROR: expected exactly 6 changed paths, got $change_count" >&2
  printf '%s\n' "$status" >&2
  exit 1
}

git config user.name "RUM Development"
git config user.email "rum-development@daisycloversoftware.uk"
git commit --quiet -m "fix(rat): version immutable brand asset URLs"
new_head="$(git rev-parse HEAD)"

remote_head="$(git ls-remote origin "refs/heads/$TARGET_BRANCH" | awk '{print $1}')"
[[ "$remote_head" == "$EXPECTED_HEAD" ]] || {
  echo "ERROR: RAT candidate moved before push: expected $EXPECTED_HEAD got $remote_head" >&2
  exit 1
}

git push --quiet origin "HEAD:$TARGET_BRANCH"

printf 'candidate_old=%s\n' "$EXPECTED_HEAD"
printf 'candidate_new=%s\n' "$new_head"
printf 'thumb_url=%s\n' "$THUMB_URL"
printf 'mascot_url=%s\n' "$MASCOT_URL"
printf 'changed_paths=%s\n' "$change_count"
