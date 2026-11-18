#!/usr/bin/env bash
set -Eeuo pipefail
# Syntax and release guards only. Does not deploy or edit any Thing.
[[ $# -eq 0 ]] || exit 64
for path in \
 scripts/ops/deploy-rat177-thing-type-api-isolated-20261118.sh \
 scripts/ops/deploy-rat177-type-editor-frontend-isolated-20261118.sh \
 scripts/ops/verify-rat177-cj-thing-type-browser-20261118.sh; do
 [[ -f "$path" && ! -L "$path" ]] || exit 66
 bash -n "$path"
done
tmp="$(mktemp --suffix=.mjs)"
trap 'rm -f "$tmp"' EXIT
sed -n '/^import assert from /,/^NODE$/p' scripts/ops/verify-rat177-cj-thing-type-browser-20261118.sh | sed '$d' > "$tmp"
[[ -s "$tmp" ]] || exit 65
node --check "$tmp"
python3 - <<'PY'
from pathlib import Path
api=Path('scripts/ops/deploy-rat177-thing-type-api-isolated-20261118.sh').read_text()
rat=Path('scripts/ops/deploy-rat177-type-editor-frontend-isolated-20261118.sh').read_text()
browser=Path('scripts/ops/verify-rat177-cj-thing-type-browser-20261118.sh').read_text()
for path,src in [
 ('scripts/ops/deploy-rat177-thing-type-api-isolated-20261118.sh',api),
 ('scripts/ops/deploy-rat177-type-editor-frontend-isolated-20261118.sh',rat),
 ('scripts/ops/verify-rat177-cj-thing-type-browser-20261118.sh',browser),
]:
 for block in src.split("<<'PY'\n")[1:]:
  compile(block.split('\nPY\n',1)[0],path,'exec')
 assert '049c9e304823e040a087ab4760bd932a98e93217' in src
assert 'ghcr.io/daisycloversoftware/rum-api@sha256:76731a87574a11547938a398dc15ecf87ced56d63b6622cd50bab5836dfeee71' in api
assert 'ghcr.io/daisycloversoftware/rum-rate-anything@sha256:448807a70db93b91bb0fd714b854253c93157ad4d89d96255a385b94bf1258bc' in rat
assert 'rollback()' in api and 'rollback()' in rat
assert 'protected_snapshot()' in rat
assert "__RAT177_API_DIGEST_PENDING__" in api and "__RAT177_RAT_DIGEST_PENDING__" in rat
print('RAT177_GUARDED_SOURCE_SYNTAX_AND_RELEASE_PINS_PASS=true')
PY
echo RAT177_RELEASE_BROWSER_SYNTAX_PASS=true