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
for name, count in [('mascotv2',3),('mascotv10',9)]:
    parts=[]; print(f'variant={name}')
    for i in range(count):
        raw=subprocess.check_output(['git','show',f'FETCH_HEAD:.rat-transfer/{name}.{i:02d}'])
        text=b''.join(raw.split())
        print(f' part={i:02d} chars={len(text)} sha256={hashlib.sha256(raw).hexdigest()}')
        if not re.fullmatch(rb'[A-Za-z0-9+/=]*',text):
            print(' result=INVALID_CHARS'); parts=[]; break
        parts.append(text)
    if not parts: continue
    joined=b''.join(parts)
    print(f' base64_chars={len(joined)} mod4={len(joined)%4} equals={joined.count(b"=")}')
    candidates=[('strict',joined)]
    if len(joined)%4:
        candidates.append(('tail_padded',joined + b'='*((-len(joined))%4)))
    for label,candidate in candidates:
        try:
            data=base64.b64decode(candidate,validate=True)
        except Exception as exc:
            print(f' {label}=DECODE_FAIL error={exc}')
            continue
        out=root/f'{name}-{label}.webp'; out.write_bytes(data)
        magic=data[:12].hex(); sha=hashlib.sha256(data).hexdigest()
        print(f' {label}=DECODE_OK bytes={len(data)} sha256={sha} magic12={magic}')
        print(f' {label}_webp={data[:4]==b"RIFF" and data[8:12]==b"WEBP"}')
PY
for f in "$tmp"/*.webp; do
  [[ -f "$f" ]] || continue
  printf 'file=%s mime=%s git_blob=%s\n' "$(basename "$f")" "$(file -b --mime-type "$f" 2>/dev/null || true)" "$(git hash-object "$f")"
done
