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
import base64, hashlib, pathlib, re, sys
root=pathlib.Path(sys.argv[1])
parts=[]
for i in range(9):
    p=root/f"part.{i:02d}"
    raw=p.read_bytes()
    text=b''.join(raw.split())
    if not re.fullmatch(rb'[A-Za-z0-9+/=]*', text):
        raise SystemExit(f'ERROR: part {i:02d} contains non-base64 characters')
    print(f"part={i:02d} chars={len(text)} sha256={hashlib.sha256(raw).hexdigest()}")
    parts.append(text)
joined=b''.join(parts)
if len(joined) % 4:
    raise SystemExit(f'ERROR: combined base64 length is not divisible by 4: {len(joined)}')
data=base64.b64decode(joined, validate=True)
out=root/'mascot.webp'
out.write_bytes(data)
print(f"base64_chars={len(joined)}")
print(f"bytes={len(data)}")
print(f"sha256={hashlib.sha256(data).hexdigest()}")
print(f"magic12={data[:12].hex()}")
if data[:4] != b'RIFF' or data[8:12] != b'WEBP':
    raise SystemExit('ERROR: reconstructed mascot is not RIFF/WEBP')
if data[12:16] == b'VP8X' and len(data) >= 30:
    w=1+int.from_bytes(data[24:27],'little')
    h=1+int.from_bytes(data[27:30],'little')
    print(f"dimensions={w}x{h}")
PY

git_hash="$(git hash-object "$tmp/mascot.webp")"
mime="$(file -b --mime-type "$tmp/mascot.webp" 2>/dev/null || true)"
printf 'git_blob=%s\n' "$git_hash"
printf 'mime=%s\n' "$mime"
