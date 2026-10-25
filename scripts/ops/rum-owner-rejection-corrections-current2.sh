#!/usr/bin/env bash
set -euo pipefail
umask 077

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SOURCE="$ROOT/scripts/ops/rum-owner-rejection-corrections.sh"
[[ -f "$SOURCE" && ! -L "$SOURCE" ]] || { echo "PATCH BLOCKED: owner correction source unavailable" >&2; exit 78; }

COPY="$(mktemp "$ROOT/scripts/ops/.rum-owner-rejection-corrections-current2.XXXXXX.sh")"
trap 'rm -f "$COPY"' EXIT HUP INT TERM
cp "$SOURCE" "$COPY"

python3 - "$COPY" <<'PY'
from pathlib import Path
import sys
path = Path(sys.argv[1])
text = path.read_text()
replacements = [
    ("color: #431f0f;", "color: #431f0e;", "bronze colour marker"),
    (
        '''    "  if (isVerified(item)) return 'positive';\\n  if (item.claim.status === 'pending') return 'warning';",''',
        '''    "  if (isVerified(item)) return 'positive';\\n  if (item.claim.verificationState === 'pending' || item.claim.verificationState === 'failed' || item.claim.status === 'pending') return 'warning';",''',
        "My Identities current tone marker",
    ),
]
for old, new, label in replacements:
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"PATCH BLOCKED: {label} count={count}")
    text = text.replace(old, new, 1)
path.write_text(text)
PY
chmod 0700 "$COPY"
printf 'RUM_OWNER_CORRECTION_WRAPPER=CURRENT2\n'
bash "$COPY" "$@"
