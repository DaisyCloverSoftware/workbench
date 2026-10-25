#!/usr/bin/env bash
set -euo pipefail
umask 077

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SOURCE="$ROOT/scripts/ops/rum-owner-rejection-corrections.sh"
[[ -f "$SOURCE" && ! -L "$SOURCE" ]] || { echo "PATCH BLOCKED: owner correction source unavailable" >&2; exit 78; }

COPY="$(mktemp "$ROOT/scripts/ops/.rum-owner-rejection-corrections-current.XXXXXX.sh")"
trap 'rm -f "$COPY"' EXIT HUP INT TERM
cp "$SOURCE" "$COPY"

python3 - "$COPY" <<'PY'
from pathlib import Path
import sys
path = Path(sys.argv[1])
text = path.read_text()
old = "color: #431f0f;"
new = "color: #431f0e;"
if text.count(old) != 1:
    raise SystemExit(f"PATCH BLOCKED: bronze marker count={text.count(old)}")
path.write_text(text.replace(old, new, 1))
PY
chmod 0700 "$COPY"
printf 'RUM_OWNER_CORRECTION_WRAPPER=BRONZE_MARKER_CURRENT\n'
bash "$COPY" "$@"
