#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 0 ]]||exit 64
ops=$(cd "$(dirname "$0")" && pwd);t=$(mktemp -d);trap 'rm -rf "$t"' EXIT
python3 - "$ops/prepare-rat-connected-thumbnails-20261125.sh" "$t/prepare.sh" <<'PY'
from pathlib import Path
import sys
s=Path(sys.argv[1]).read_text()
for old,new in [("className: 'existing-size', fallback }))", "className: 'existing-size', ...(fallback === undefined ? {} : { fallback }) }))"),("render({ ...entity, images: undefined })", "render({ id: entity.id, name: entity.name })")]:
 assert s.count(old)==1;s=s.replace(old,new)
Path(sys.argv[2]).write_text(s)
PY
bash -n "$t/prepare.sh";bash "$t/prepare.sh"
