#!/usr/bin/env bash
set -euo pipefail
umask 077
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SOURCE="$ROOT/scripts/ops/rum-owner-rejection-profile-dev-verifier.sh"
[[ -f "$SOURCE" && ! -L "$SOURCE" ]] || { echo "VERIFY BLOCKED: profile verifier source unavailable" >&2; exit 78; }
work="$(mktemp -d)"; trap 'rm -rf "$work"' EXIT HUP INT TERM
COPY="$work/profile-owner-rejection-current2.sh"; cp "$SOURCE" "$COPY"
python3 - "$COPY" <<'PY'
from pathlib import Path
import sys
p=Path(sys.argv[1]); text=p.read_text()
def once(old,new,label):
    global text
    c=text.count(old)
    if c != 1: raise SystemExit(f'{label} marker count={c}')
    text=text.replace(old,new,1)
once('Storage::disk($m->storage_disk)', 'Illuminate\\Support\\Facades\\Storage::disk($m->storage_disk)', 'storage facade')
once("base=os.environ['RUM_BASE_URL']; owner_name=os.environ['RUM_OWNER_NAME']; avatar_media_id=os.environ['RUM_AVATAR_MEDIA_ID']", "base=os.environ['RUM_BASE_URL']; owner_name=os.environ['RUM_OWNER_NAME']; avatar_media_id=os.environ['RUM_AVATAR_MEDIA_ID']; baseline=os.environ['RUM_BASELINE_CSP']", 'baseline environment')
once("require(labels==['Overall Public Rating','Rate My Rating','Mates Only Rating'], f'unexpected principal metric labels: {labels}')", "require([label.casefold() for label in labels]==['overall public rating','rate my rating','mates only rating'], f'unexpected principal metric labels: {labels}')", 'rendered metric label casing')
once("if console_errors: raise RuntimeError('console errors: '+' | '.join(console_errors[:5]))\nprint('profile_unexpected_console_errors=0')", "known=[msg for msg in console_errors if baseline in msg and \"script-src 'self'\" in msg]\nunexpected=[msg for msg in console_errors if msg not in known]\nprint(f'profile_known_live_baseline_csp_console_errors={len(known)}')\nif unexpected: raise RuntimeError('unexpected console errors: '+' | '.join(unexpected[:5]))\nprint('profile_unexpected_console_errors=0')", 'console baseline filter')
once('"$runtime" run --rm --ipc=host -v "$work:/work" -e RUM_BASE_URL="$BASE_URL" -e RUM_OWNER_NAME="$owner_name" -e RUM_AVATAR_MEDIA_ID="$avatar_media_id" "$image" python /work/verify.py', '"$runtime" run --rm --ipc=host -v "$work:/work" -e RUM_BASE_URL="$BASE_URL" -e RUM_OWNER_NAME="$owner_name" -e RUM_AVATAR_MEDIA_ID="$avatar_media_id" -e RUM_BASELINE_CSP="sha256-A0FJyCgxFUPhG7nac5LcQPwVRK5So9ZNz7x5ubsD9kU=" "$image" python /work/verify.py', 'browser environment')
p.write_text(text)
PY
chmod 0700 "$COPY"
bash "$COPY" "$@"
