#!/usr/bin/env bash
set -euo pipefail

RUM_REPO="DaisyCloverSoftware/rum"
TRANSFER_BRANCH="ops/rat-owner-assets-transfer-20260828"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

git clone --quiet --filter=blob:none --no-checkout "git@github.com:${RUM_REPO}.git" "$tmp/rum"
cd "$tmp/rum"
git fetch --quiet --depth=1 origin "$TRANSFER_BRANCH"

for i in 00 01 02 03 04 05 06 07 08; do
  git show "FETCH_HEAD:.rat-transfer/mascotv10.$i" > "$tmp/part.$i"
done

python3 - "$tmp" <<'PY'
import base64, hashlib, pathlib, struct, sys
root=pathlib.Path(sys.argv[1])
parts=[]
for i in range(9):
    p=root/f"part.{i:02d}"
    raw=p.read_bytes()
    text=b''.join(raw.split())
    print(f"part={i:02d} chars={len(text)} sha256={hashlib.sha256(raw).hexdigest()}")
    base64.b64decode(text, validate=True)
    parts.append(text)
joined=b''.join(parts)
data=base64.b64decode(joined, validate=True)
out=root/'mascot.webp'
out.write_bytes(data)
print(f"base64_chars={len(joined)}")
print(f"bytes={len(data)}")
print(f"sha256={hashlib.sha256(data).hexdigest()}")
print(f"magic12={data[:12].hex()}")
if data[:4] != b'RIFF' or data[8:12] != b'WEBP':
    raise SystemExit('ERROR: reconstructed mascot is not RIFF/WEBP')
# Parse VP8X dimensions when present.
if data[12:16] == b'VP8X' and len(data) >= 30:
    w=1+int.from_bytes(data[24:27],'little')
    h=1+int.from_bytes(data[27:30],'little')
    print(f"dimensions={w}x{h}")
PY

git_hash="$(git hash-object "$tmp/mascot.webp")"
mime="$(file -b --mime-type "$tmp/mascot.webp" 2>/dev/null || true)"
printf 'git_blob=%s\n' "$git_hash"
printf 'mime=%s\n' "$mime"
