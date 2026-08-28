#!/usr/bin/env bash
set -euo pipefail

RUM_REPO="https://github.com/DaisyCloverSoftware/rum.git"
TRANSFER_BRANCH="ops/rat-owner-assets-transfer-20260828"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

git clone --quiet --depth=1 --branch "$TRANSFER_BRANCH" "$RUM_REPO" "$tmp/rum"
cd "$tmp/rum"

python3 - "$tmp/mascot.webp" <<'PY'
import base64, hashlib, pathlib, re, sys
out = pathlib.Path(sys.argv[1])
parts = [pathlib.Path(f'.rat-transfer/mascot.{i:02d}') for i in range(6)]
raw = bytearray()
for p in parts:
    text = p.read_text(encoding='ascii')
    compact = ''.join(text.split())
    bad = sorted(set(re.sub(r'[A-Za-z0-9+/=]', '', compact)))
    print(f'part={p.name} file_bytes={p.stat().st_size} compact_chars={len(compact)} mod4={len(compact)%4} bad={bad!r} head={compact[:12]} tail={compact[-12:]}')
    try:
        decoded = base64.b64decode(compact, validate=True)
        print(f'part_decode={p.name}:strict_ok bytes={len(decoded)} sha256={hashlib.sha256(decoded).hexdigest()}')
    except Exception as exc:
        print(f'part_decode={p.name}:strict_error {type(exc).__name__}:{exc}')
        decoded = base64.b64decode(compact + '=' * ((-len(compact)) % 4), validate=False)
        print(f'part_decode={p.name}:lenient bytes={len(decoded)} sha256={hashlib.sha256(decoded).hexdigest()}')
    raw.extend(decoded)
out.write_bytes(raw)
print(f'combined_bytes={len(raw)}')
print(f'combined_sha256={hashlib.sha256(raw).hexdigest()}')
print(f'combined_magic12={raw[:12].hex()}')
if raw[:4] == b'RIFF' and raw[8:12] == b'WEBP':
    print('combined_container=webp')
else:
    print('combined_container=not_webp')
PY

if command -v file >/dev/null 2>&1; then
  printf 'mime=%s\n' "$(file -b --mime-type "$tmp/mascot.webp")"
fi
