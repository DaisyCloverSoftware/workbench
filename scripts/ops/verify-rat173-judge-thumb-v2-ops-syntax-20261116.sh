#!/usr/bin/env bash
set -euo pipefail
[[ $# -eq 0 ]] || exit 64
for script in scripts/ops/deploy-rat173-judge-thumbs-frontend-isolated-v2.sh scripts/ops/verify-rat173-judge-thumb-parity-browser-v2-20261116.sh; do
  [[ -f "$script" && ! -L "$script" ]] || exit 66
  bash -n "$script"
done
tmp="$(mktemp --suffix=.mjs)"
trap 'rm -f "$tmp"' EXIT
sed -n '/^import assert from /,/^NODE$/p' scripts/ops/verify-rat173-judge-thumb-parity-browser-v2-20261116.sh | sed '$d' > "$tmp"
[[ -s "$tmp" ]] || exit 65
node --check "$tmp"
python3 - <<'PYCHECK'
from pathlib import Path
scripts = [
 'scripts/ops/deploy-rat173-judge-thumbs-frontend-isolated-v2.sh',
 'scripts/ops/verify-rat173-judge-thumb-parity-browser-v2-20261116.sh',
]
for path in scripts:
    text = Path(path).read_text()
    blocks = text.split("<<'PY'\n")[1:]
    assert blocks, f"no Python heredoc in {path}"
    for block in blocks:
        source = block.split("\nPY\n", 1)[0]
        assert "\nassert " not in source, f"literal newline escape in {path}"
        compile(source, path + ":embedded-python", "exec")
print("RAT173_EMBEDDED_PYTHON_SYNTAX_PASS=true")
PYCHECK
echo RAT173_JUDGE_THUMBS_V2_OPS_SYNTAX_PASS=true
