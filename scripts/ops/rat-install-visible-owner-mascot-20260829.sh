#!/usr/bin/env bash
set -euo pipefail

RUM_REPO="https://github.com/DaisyCloverSoftware/rum.git"
TARGET_BRANCH="sprint-1-rat-foundation-search-explorer-20260826"
TRANSFER_BRANCH="ops/rat-owner-assets-transfer-20260828"
EXPECTED_HEAD="cf92e170c8b8728cb59c5b22c424e6472d048b49"
TARGET_PATH="apps/rate-anything/public/brand/rat-pirate-mascot.webp"
BRAND_ASSETS_PATH="apps/rate-anything/src/brandAssets.ts"
EXPECTED_OLD_BLOB="0c4ade813ec3e41a5520add8e8cfc818f890ad8a"
EXPECTED_OLD_URL="/brand/rat-pirate-mascot.webp?v=0c4ade813ec3e41a"
EXPECTED_NEW_URL="/brand/rat-pirate-mascot.webp?v=7ab886eafde87205"
EXPECTED_NEW_SHA256="d66164326c0e042c182b307fa8755564d74d6dbc643448888e916e5a293b938c"
EXPECTED_NEW_BYTES="55392"
EXPECTED_NEW_BLOB="7ab886eafde872051803ca0a4229f4d0af404b71"

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
  echo "ERROR: current mascot blob changed: expected $EXPECTED_OLD_BLOB got $old_blob" >&2
  exit 1
}

grep -Fq "$EXPECTED_OLD_URL" "$BRAND_ASSETS_PATH" || {
  echo "ERROR: current mascot asset URL changed or is missing" >&2
  exit 1
}

for i in 00 01 02 03 04; do
  git show "origin/$TRANSFER_BRANCH:.rat-transfer/mascotfinal.$i" > "$tmp/mascotfinal.$i"
done

python3 - "$tmp" "$tmp/rat-pirate-mascot.webp" <<'PY'
import base64
import hashlib
from pathlib import Path
import sys

root = Path(sys.argv[1])
out = Path(sys.argv[2])
expected = {
    "00": (16000, "51db1cb7960028e2186006940cb54ed402adc1abafd1dc54f6cf72f11f564894"),
    "01": (16000, "252f1fc5a04df62281ab3ecd0dfcd99a43fa70f7498ddf6a5c46a916b0656230"),
    "02": (16000, "2fc18fb1aa493e42e3542dc1267d73b897c7600dbbac39811463d063980b74ac"),
    "03": (16000, "69534eee78ee8e575772702c858f9ad9d9616db127ad0ed0206e5e9d86ef5fb0"),
    "04": (9856, "6ee4c45b560b2198e0e1fc1d0f8139b42285af203c3935231cd8171ade907394"),
}
parts = []
for suffix, (expected_len, expected_sha) in expected.items():
    raw = (root / f"mascotfinal.{suffix}").read_bytes()
    actual_sha = hashlib.sha256(raw).hexdigest()
    if len(raw) != expected_len:
        raise SystemExit(f"ERROR: mascotfinal.{suffix} length mismatch: expected {expected_len} got {len(raw)}")
    if actual_sha != expected_sha:
        raise SystemExit(f"ERROR: mascotfinal.{suffix} sha256 mismatch: expected {expected_sha} got {actual_sha}")
    try:
        text = raw.decode("ascii")
    except UnicodeDecodeError as exc:
        raise SystemExit(f"ERROR: mascotfinal.{suffix} is not ASCII: {exc}")
    parts.append(text)

joined = "".join(parts)
if len(joined) != 73856:
    raise SystemExit(f"ERROR: assembled base64 length mismatch: {len(joined)}")
try:
    decoded = base64.b64decode(joined, validate=True)
except Exception as exc:
    raise SystemExit(f"ERROR: strict base64 decode failed: {exc}")

expected_bytes = 55392
expected_sha = "d66164326c0e042c182b307fa8755564d74d6dbc643448888e916e5a293b938c"
actual_sha = hashlib.sha256(decoded).hexdigest()
if len(decoded) != expected_bytes:
    raise SystemExit(f"ERROR: decoded mascot size mismatch: expected {expected_bytes} got {len(decoded)}")
if actual_sha != expected_sha:
    raise SystemExit(f"ERROR: decoded mascot sha256 mismatch: expected {expected_sha} got {actual_sha}")
if decoded[:4] != b"RIFF" or decoded[8:12] != b"WEBP":
    raise SystemExit(f"ERROR: decoded mascot is not RIFF/WEBP: {decoded[:12].hex()}")
declared_bytes = int.from_bytes(decoded[4:8], "little") + 8
if declared_bytes != len(decoded):
    raise SystemExit(f"ERROR: RIFF declared length mismatch: declared {declared_bytes} actual {len(decoded)}")
if decoded[12:16] != b"VP8X":
    raise SystemExit(f"ERROR: expected VP8X WebP, got chunk {decoded[12:16]!r}")
width = 1 + int.from_bytes(decoded[24:27], "little")
height = 1 + int.from_bytes(decoded[27:30], "little")
if (width, height) != (360, 450):
    raise SystemExit(f"ERROR: mascot dimensions mismatch: expected 360x450 got {width}x{height}")
out.write_bytes(decoded)
print(f"assembled_base64_chars={len(joined)}")
print(f"decoded_bytes={len(decoded)}")
print(f"decoded_sha256={actual_sha}")
print(f"decoded_magic12={decoded[:12].hex()}")
print(f"riff_declared_bytes={declared_bytes}")
print(f"dimensions={width}x{height}")
PY

bytes="$(wc -c < "$tmp/rat-pirate-mascot.webp" | tr -d ' ')"
sha256="$(sha256sum "$tmp/rat-pirate-mascot.webp" | awk '{print $1}')"
[[ "$bytes" == "$EXPECTED_NEW_BYTES" ]] || { echo "ERROR: decoded mascot size changed" >&2; exit 1; }
[[ "$sha256" == "$EXPECTED_NEW_SHA256" ]] || { echo "ERROR: decoded mascot SHA changed" >&2; exit 1; }
if command -v file >/dev/null 2>&1; then
  mime="$(file -b --mime-type "$tmp/rat-pirate-mascot.webp")"
  [[ "$mime" == "image/webp" ]] || { echo "ERROR: decoded mascot MIME is $mime" >&2; exit 1; }
fi

install -m 0644 "$tmp/rat-pirate-mascot.webp" "$TARGET_PATH"
new_blob="$(git hash-object "$TARGET_PATH")"
[[ "$new_blob" == "$EXPECTED_NEW_BLOB" ]] || {
  echo "ERROR: installed mascot blob mismatch: expected $EXPECTED_NEW_BLOB got $new_blob" >&2
  exit 1
}

python3 - "$BRAND_ASSETS_PATH" "$EXPECTED_OLD_URL" "$EXPECTED_NEW_URL" <<'PY'
from pathlib import Path
import sys
p=Path(sys.argv[1]); old=sys.argv[2]; new=sys.argv[3]
text=p.read_text()
if text.count(old) != 1:
    raise SystemExit(f"ERROR: expected exactly one old mascot URL, found {text.count(old)}")
p.write_text(text.replace(old,new))
PY

git add "$TARGET_PATH" "$BRAND_ASSETS_PATH"
status="$(git status --porcelain)"
printf 'staged_status=%s\n' "$status"
[[ "$(printf '%s\n' "$status" | wc -l | tr -d ' ')" == "2" ]] || {
  echo "ERROR: unexpected number of changed paths" >&2
  exit 1
}

git config user.name "RUM Development"
git config user.email "rum-development@daisycloversoftware.uk"
git commit --quiet -m "fix(rat): restore visible owner pirate mascot"
new_head="$(git rev-parse HEAD)"

remote_head="$(git ls-remote origin "refs/heads/$TARGET_BRANCH" | awk '{print $1}')"
[[ "$remote_head" == "$EXPECTED_HEAD" ]] || {
  echo "ERROR: RAT candidate moved before push: expected $EXPECTED_HEAD got $remote_head" >&2
  exit 1
}

git push --quiet origin "HEAD:$TARGET_BRANCH"

printf 'candidate_old=%s\n' "$EXPECTED_HEAD"
printf 'candidate_new=%s\n' "$new_head"
printf 'mascot_bytes=%s\n' "$bytes"
printf 'mascot_sha256=%s\n' "$sha256"
printf 'mascot_blob=%s\n' "$new_blob"
printf 'mascot_url=%s\n' "$EXPECTED_NEW_URL"
printf 'target_branch=%s\n' "$TARGET_BRANCH"
