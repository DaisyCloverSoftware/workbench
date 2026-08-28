#!/usr/bin/env bash
set -euo pipefail

RUM_REPO="https://github.com/DaisyCloverSoftware/rum.git"
TARGET_BRANCH="sprint-1-rat-foundation-search-explorer-20260826"
TRANSFER_BRANCH="ops/rat-owner-assets-transfer-20260828"
EXPECTED_HEAD="8884806dc14d7047ffe70c5eef77824d3964e05e"
EXPECTED_OLD_BLOB="04a98a4791aa66f4c82a18f0317404c18e840116"
EXPECTED_NEW_SHA256="38643e3cad4e1e9e22b1726a9fdbc8cb415e6630ce60db9601c9c734627c853c"
EXPECTED_NEW_BYTES="14232"
EXPECTED_NEW_BLOB="ad234fbada71f3be1829a33157928dfda333afb8"
TARGET_PATH="apps/rate-anything/public/brand/rum-thumb.webp"

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

for i in 00 01 02 03; do
  git show "origin/$TRANSFER_BRANCH:.rat-transfer/thumb.$i" > "$tmp/thumb.$i"
done

python3 - "$tmp" "$tmp/rum-thumb.webp" <<'PY'
import base64
import hashlib
from pathlib import Path
import sys

root = Path(sys.argv[1])
out = Path(sys.argv[2])
expected = {
    "00": (4800, "0b33492c80d1d80d98ce0a627e58db7dde79c5007f5521abf02cea048394a0e8"),
    "01": (4800, "7858bb1361230846d48d65a7a31a42a719f03f407ddcc46a8dea6bfabfa59dbd"),
    "02": (4800, "7826ebc7928abd81082ceec8a297ea2facfa1eaf0b229e68004b30351809c4e0"),
    "03": (4576, "1bd6f94c82dae5f380dcca2a256ee56626d896d296897f680d358393ead228ba"),
}
parts = []
for suffix, (expected_len, expected_sha) in expected.items():
    raw = (root / f"thumb.{suffix}").read_bytes()
    actual_sha = hashlib.sha256(raw).hexdigest()
    if len(raw) != expected_len:
        raise SystemExit(f"ERROR: thumb.{suffix} length mismatch: expected {expected_len} got {len(raw)}")
    if actual_sha != expected_sha:
        raise SystemExit(f"ERROR: thumb.{suffix} sha256 mismatch: expected {expected_sha} got {actual_sha}")
    try:
        text = raw.decode("ascii")
    except UnicodeDecodeError as exc:
        raise SystemExit(f"ERROR: thumb.{suffix} is not ASCII: {exc}")
    parts.append(text)

joined = "".join(parts)
if len(joined) != 18976:
    raise SystemExit(f"ERROR: assembled base64 length mismatch: {len(joined)}")
try:
    decoded = base64.b64decode(joined, validate=True)
except Exception as exc:
    raise SystemExit(f"ERROR: strict base64 decode failed: {exc}")

expected_bytes = 14232
expected_sha = "38643e3cad4e1e9e22b1726a9fdbc8cb415e6630ce60db9601c9c734627c853c"
actual_sha = hashlib.sha256(decoded).hexdigest()
if len(decoded) != expected_bytes:
    raise SystemExit(f"ERROR: decoded thumb size mismatch: expected {expected_bytes} got {len(decoded)}")
if actual_sha != expected_sha:
    raise SystemExit(f"ERROR: decoded thumb sha256 mismatch: expected {expected_sha} got {actual_sha}")
if decoded[:4] != b"RIFF" or decoded[8:12] != b"WEBP":
    raise SystemExit(f"ERROR: decoded thumb is not RIFF/WEBP: {decoded[:12].hex()}")
out.write_bytes(decoded)
print(f"assembled_base64_chars={len(joined)}")
print(f"decoded_bytes={len(decoded)}")
print(f"decoded_sha256={actual_sha}")
print(f"decoded_magic12={decoded[:12].hex()}")
PY

bytes="$(wc -c < "$tmp/rum-thumb.webp" | tr -d ' ')"
sha256="$(sha256sum "$tmp/rum-thumb.webp" | awk '{print $1}')"
[[ "$bytes" == "$EXPECTED_NEW_BYTES" ]] || {
  echo "ERROR: decoded thumb size mismatch after Python validation" >&2
  exit 1
}
[[ "$sha256" == "$EXPECTED_NEW_SHA256" ]] || {
  echo "ERROR: decoded thumb SHA changed after Python validation" >&2
  exit 1
}
if command -v file >/dev/null 2>&1; then
  mime="$(file -b --mime-type "$tmp/rum-thumb.webp")"
  [[ "$mime" == "image/webp" ]] || {
    echo "ERROR: decoded thumb MIME is $mime, expected image/webp" >&2
    exit 1
  }
fi

install -m 0644 "$tmp/rum-thumb.webp" "$TARGET_PATH"
new_blob="$(git hash-object "$TARGET_PATH")"
[[ "$new_blob" == "$EXPECTED_NEW_BLOB" ]] || {
  echo "ERROR: installed thumb blob mismatch: expected $EXPECTED_NEW_BLOB got $new_blob" >&2
  exit 1
}

git add "$TARGET_PATH"
status="$(git status --porcelain)"
[[ "$status" == " M $TARGET_PATH" || "$status" == "M  $TARGET_PATH" ]] || {
  echo "ERROR: unexpected candidate working tree after thumb install: $status" >&2
  exit 1
}

git config user.name "RUM Development"
git config user.email "rum-development@daisycloversoftware.uk"
git commit --quiet -m "fix(rat): use owner-approved RUM thumb"
new_head="$(git rev-parse HEAD)"

# Fail closed if the remote candidate moved after our initial fetch.
remote_head="$(git ls-remote origin "refs/heads/$TARGET_BRANCH" | awk '{print $1}')"
[[ "$remote_head" == "$EXPECTED_HEAD" ]] || {
  echo "ERROR: RAT candidate moved before push: expected $EXPECTED_HEAD got $remote_head" >&2
  exit 1
}

git push --quiet origin "HEAD:$TARGET_BRANCH"

printf 'candidate_old=%s\n' "$EXPECTED_HEAD"
printf 'candidate_new=%s\n' "$new_head"
printf 'thumb_bytes=%s\n' "$bytes"
printf 'thumb_sha256=%s\n' "$sha256"
printf 'thumb_blob=%s\n' "$new_blob"
printf 'target_branch=%s\n' "$TARGET_BRANCH"
