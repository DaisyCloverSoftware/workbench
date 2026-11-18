#!/usr/bin/env bash
set -Eeuo pipefail
# Exact-head source-only checks. Does not run application tests against DEV data.
[[ $# -eq 1 && "$1" == 'e54abfaabde5bbd69b0b97e55e8770754fea3401' ]] || exit 64
source_sha="$1"
work="$(mktemp -d)";trap 'rm -rf "$work"' EXIT
git -C "$work" init -q
git -C "$work" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$work" fetch -q --depth=1 origin "$source_sha"
git -C "$work" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$work" rev-parse HEAD)" == "$source_sha" ]] || exit 78
for file in apps/api/app/Services/RatExplorerService.php apps/api/tests/Feature/Entity/RatExplorerPrivacyTest.php;do
 if command -v php >/dev/null 2>&1;then php -l "$work/$file"
 else
  if command -v k3s >/dev/null 2>&1;then cat "$work/$file" | sudo -n k3s kubectl -n rum-dev-isolated exec -i deployment/rum-api -c php-fpm -- php -l
  else cat "$work/$file" | kubectl -n rum-dev-isolated exec -i deployment/rum-api -c php-fpm -- php -l
  fi
 fi
done
python3 - "$work" <<'PY'
from pathlib import Path
import sys
base=Path(sys.argv[1])
source=(base/'apps/api/app/Services/RatExplorerService.php').read_text()
test=(base/'apps/api/tests/Feature/Entity/RatExplorerPrivacyTest.php').read_text()
assert "$type === 'digital_resource' => $this->digitalResourceCategory($entity)" in source
assert "'key' => 'channels', 'label' => 'Channels'" in source
assert "'key' => 'other-digital-resources', 'label' => 'Other digital resources'" in source
assert "source_entity_id" in test and "The Professional Idiots" in test
assert "test_digital_resources_use_explicit_canonical_keys_with_truthful_unknown_fallback" in test
assert "array_slice($linked, 0, 3)" in source
assert "$this->policy->allows($related)" in source
print("RAT178_FOCUSED_SOURCE_GUARDS_PASS=true")
PY
echo RAT178_EXACT_SOURCE_SYNTAX_PASS=true
