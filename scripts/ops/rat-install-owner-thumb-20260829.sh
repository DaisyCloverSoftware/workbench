#!/usr/bin/env bash
set -euo pipefail

RUM_REPO="https://github.com/DaisyCloverSoftware/rum.git"
TARGET_BRANCH="sprint-1-rat-foundation-search-explorer-20260826"
TRANSFER_BRANCH="ops/rat-owner-assets-transfer-20260828"
EXPECTED_HEAD="8884806dc14d7047ffe70c5eef77824d3964e05e"
EXPECTED_OLD_BLOB="04a98a4791aa66f4c82a18f0317404c18e840116"
EXPECTED_NEW_SHA256="38643e3cad4e1e9ae0b1015cb6e42645145ada0b6b5132077d038e9b899d6e78c"
EXPECTED_NEW_BYTES="14232"
TARGET_PATH="apps/rate-anything/public/brand/rum-thumb.webp"
TRANSFER_PATH=".rat-transfer/thumb-384-q94.b64"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

git clone --quiet --no-checkout "$RUM_REPO" "$tmp/rum"
cd "$tmp/rum"
git fetch --quiet origin "$TARGET_BRANCH" "$TRANSFER_BRANCH"
git checkout --quiet -B "$TARGET_BRANCH" "origin/$TARGET_BRANCH"

actual_head="$(git rev-parse HEAD)"
[[ "$actual_head" == "$EXPECTED_HEAD" ]] || {
  echo "ERROR: RAT candidate moved: expected $EXPECTED_HEAD got $actual_head" >&2
  exit 1
}

old_blob="$(git hash-object "$TARGET_PATH")"
[[ "$old_blob" == "$EXPECTED_OLD_BLOB" ]] || {
  echo "ERROR: current thumb blob changed: expected $EXPECTED_OLD_BLOB got $old_blob" >&2
  exit 1
}

git show "origin/$TRANSFER_BRANCH:$TRANSFER_PATH" | tr -d '\r\n\t ' | base64 -d > "$tmp/rum-thumb.webp"

bytes="$(wc -c < "$tmp/rum-thumb.webp" | tr -d ' ')"
sha256="$(sha256sum "$tmp/rum-thumb.webp" | awk '{print $1}')"
magic="$(od -An -tx1 -N12 "$tmp/rum-thumb.webp" | tr -d ' \n')"
[[ "$bytes" == "$EXPECTED_NEW_BYTES" ]] || {
  echo "ERROR: decoded thumb size mismatch: expected $EXPECTED_NEW_BYTES got $bytes" >&2
  exit 1
}
[[ "$sha256" == "$EXPECTED_NEW_SHA256" ]] || {
  echo "ERROR: decoded thumb sha256 mismatch: expected $EXPECTED_NEW_SHA256 got $sha256" >&2
  exit 1
}
[[ "$magic" == 52494646*57454250 ]] || {
  echo "ERROR: decoded thumb is not RIFF/WEBP: $magic" >&2
  exit 1
}

install -m 0644 "$tmp/rum-thumb.webp" "$TARGET_PATH"
new_blob="$(git hash-object "$TARGET_PATH")"
[[ "$new_blob" == "ad234fbada71f3be1829a33157928dfda333afb8" ]] || {
  echo "ERROR: installed thumb blob mismatch: $new_blob" >&2
  exit 1
}

git add "$TARGET_PATH"
[[ -n "$(git status --porcelain)" ]] || {
  echo "ERROR: owner thumb produced no candidate change" >&2
  exit 1
}

git config user.name "RUM Development"
git config user.email "rum-development@daisycloversoftware.uk"
git commit --quiet -m "fix(rat): use owner-approved RUM thumb"
new_head="$(git rev-parse HEAD)"
git push --quiet origin "HEAD:$TARGET_BRANCH"

printf 'candidate_old=%s\n' "$EXPECTED_HEAD"
printf 'candidate_new=%s\n' "$new_head"
printf 'thumb_bytes=%s\n' "$bytes"
printf 'thumb_sha256=%s\n' "$sha256"
printf 'thumb_blob=%s\n' "$new_blob"
printf 'target_branch=%s\n' "$TARGET_BRANCH"
