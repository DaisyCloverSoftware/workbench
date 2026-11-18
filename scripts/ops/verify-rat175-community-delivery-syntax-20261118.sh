#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only syntax for the two guarded, fail-closed API/RAT-only rollout scripts.
[[ $# -eq 0 ]] || exit 64
for script in scripts/ops/deploy-rat175-community-api-isolated-20261118.sh scripts/ops/deploy-rat175-community-frontend-isolated-20261118.sh; do
 [[ -f "$script" && ! -L "$script" ]] || exit 66
 bash -n "$script"
done
python3 - <<'PY'
from pathlib import Path
for path in ("scripts/ops/deploy-rat175-community-api-isolated-20261118.sh", "scripts/ops/deploy-rat175-community-frontend-isolated-20261118.sh"):
  source=Path(path).read_text()
  for block in source.split("<<'PY'\n")[1:]:
    compile(block.split("\nPY\n",1)[0],path,'exec')
  assert "e4a474e2dbcac2a92272156c7cb8446ea7b74483" in source
  assert "rollback()" in source
print("RAT175_COMMUNITY_DELIVERY_PYTHON_SYNTAX_PASS=true")
PY
echo RAT175_COMMUNITY_DELIVERY_SYNTAX_PASS=true
