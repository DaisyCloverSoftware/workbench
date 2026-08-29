#!/usr/bin/env bash
set -euo pipefail
RUM_REPO="DaisyCloverSoftware/rum"
TRANSFER_BRANCH="ops/rat-owner-assets-transfer-20260828"
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
git clone --quiet --filter=blob:none --no-checkout "git@github.com:${RUM_REPO}.git" "$tmp/rum"
cd "$tmp/rum"; git fetch --quiet --depth=1 origin "$TRANSFER_BRANCH"
python3 - "$tmp" <<'PY'
import base64, hashlib, pathlib, re, subprocess, sys
root=pathlib.Path(sys.argv[1])
variants={'mascotv2':range(3),'mascotv10':range(9)}
for name, rng in variants.items():
    parts=[]
    print(f'variant={name}')
    for i in rng:
        raw=subprocess.check_output(['git','show',f'FETCH_HEAD:.rat-transfer/{name}.{i:02d}'])
        text=b''.join(raw.split())
        print(f' part={i:02d} chars={len(text)} sha256={hashlib.sha256(raw).hexdigest()}')
        if not re.fullmatch(rb'[A-Za-z0-9+/=]*',text):
            print(' result=INVALID_CHARS'); parts=[]; break
        parts.append(text)
    if not parts: continue
    joined=b''.join(parts)
    print(f' base64_chars={len(joined)} mod4={len(joined)%4}')
    try:
        data=base64.b64decode(joined,validate=True)
    except Exception as exc:
        print(f' result=DECODE_FAIL error={exc}')
        continue
    out=root/f'{name}.webp'; out.write_bytes(data)
    print(f' bytes={len(data)} sha256={hashlib.sha256(data).hexdigest()} magic12={data[:12].hex()}')
    if data[:4]==b'RIFF' and data[8:12]==b'WEBP':
        print(' result=RIFF_WEBP_OK')
    else:
        print(' result=NOT_WEBP')
PY
