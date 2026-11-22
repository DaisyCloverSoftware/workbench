#!/usr/bin/env bash
set -Eeuo pipefail
mode=${1:-};shift || exit 64
case "$mode" in source) [[ $# == 1 ]]||exit 64; entry=test-rat181-source-browser.sh;; images) [[ $# == 4 ]]||exit 64;entry=verify-rat181-disposable-image-browser.sh;; *)exit 64;;esac
ops=$(cd "$(dirname "$0")" && pwd);t=$(mktemp -d);trap 'rm -rf "$t"' EXIT
for f in test-rat181-source-browser.sh verify-rat181-disposable-image-browser.sh rat181-fixture-probe-20261122.sh rat181-fixture-loopback.mjs rat181-acceptance-20261122.mjs;do cp "$ops/$f" "$t/$f";done
python3 - "$t/rat181-acceptance-20261122.mjs" <<'PY'
from pathlib import Path
import sys
p=Path(sys.argv[1]);s=p.read_text();old="const option=form.getByRole('option').filter({has:page.getByText(label,{exact:true})}).first();await option.getByRole('button').click();";new="const option=form.locator('li[role=option] > button').filter({hasText:new RegExp('^'+label+'\\\\s*(?:General Thing kind|Existing type)')});await expect(option).toHaveCount(1);await option.click();";assert s.count(old)==1;p.write_text(s.replace(old,new))
PY
node --check "$t/rat181-acceptance-20261122.mjs";bash "$t/$entry" "$@"
