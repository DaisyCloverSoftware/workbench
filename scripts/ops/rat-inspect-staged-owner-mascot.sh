#!/usr/bin/env bash
set -euo pipefail

RUM_REPO="https://github.com/DaisyCloverSoftware/rum.git"
TRANSFER_BRANCH="ops/rat-owner-assets-transfer-20260828"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

git clone --quiet --depth=1 --branch "$TRANSFER_BRANCH" "$RUM_REPO" "$tmp/rum"
cd "$tmp/rum"

cat .rat-transfer/mascot.00 .rat-transfer/mascot.01 .rat-transfer/mascot.02 .rat-transfer/mascot.03 .rat-transfer/mascot.04 .rat-transfer/mascot.05 | base64 -d > "$tmp/mascot.webp"

printf 'bytes=%s\n' "$(wc -c < "$tmp/mascot.webp")"
printf 'sha256=%s\n' "$(sha256sum "$tmp/mascot.webp" | awk '{print $1}')"
printf 'magic12=%s\n' "$(od -An -tx1 -N12 "$tmp/mascot.webp" | tr -d ' \n')"
if command -v file >/dev/null 2>&1; then
  printf 'mime=%s\n' "$(file -b --mime-type "$tmp/mascot.webp")"
fi
if command -v python3 >/dev/null 2>&1; then
  python3 - "$tmp/mascot.webp" <<'PY'
import sys
p=sys.argv[1]
try:
    from PIL import Image
    im=Image.open(p)
    print(f'pil={im.format}|{im.width}x{im.height}|{im.mode}')
except Exception as exc:
    print(f'pil_error={type(exc).__name__}:{exc}')
PY
fi
