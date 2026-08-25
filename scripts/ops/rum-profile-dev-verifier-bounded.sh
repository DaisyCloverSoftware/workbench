#!/usr/bin/env bash
set -euo pipefail
umask 077

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SOURCE="$ROOT/scripts/ops/rum-profile-dev-verifier.sh"
[[ -f "$SOURCE" && ! -L "$SOURCE" ]] || {
  echo "VERIFY BLOCKED: base profile verifier unavailable" >&2
  exit 78
}

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT HUP INT TERM
COPY="$work/rum-profile-dev-verifier.sh"
cp "$SOURCE" "$COPY"

# Keep the base verifier authoritative, but make its isolated Founder fixture
# collision-safe, avoid relying on a navigated-away Playwright response body,
# and keep screenshot proof relay-safe. Every replacement is fail-closed so a
# future base-verifier edit cannot silently weaken coverage.
python3 - "$COPY" <<'PY'
from pathlib import Path
import sys

path = Path(sys.argv[1])
text = path.read_text()

def replace_once(old: str, new: str, label: str) -> None:
    global text
    if text.count(old) != 1:
        raise SystemExit(f'profile verifier patch blocked: {label} marker count={text.count(old)}')
    text = text.replace(old, new, 1)

replace_once(
    "import os,base64\n",
    "import os,hashlib\n",
    "screenshot import",
)
replace_once(
    "base=os.environ['RUM_BASE_URL']; owner_id=os.environ['RUM_OWNER_ID']; owner_name=os.environ['RUM_OWNER_NAME']; rating_id=os.environ['RUM_RATING_ID']",
    "base=os.environ['RUM_BASE_URL']; owner_id=os.environ['RUM_OWNER_ID']; owner_name=os.environ['RUM_OWNER_NAME']; rating_id=os.environ['RUM_RATING_ID']; founder_number=os.environ['RUM_FOUNDER_NUMBER']",
    "verifier environment",
)
replace_once(
    "owner.get_by_text('F #27', exact=True)",
    "owner.get_by_text(f'F #{founder_number}', exact=True)",
    "founder mini chip",
)
replace_once(
    "owner.get_by_text('Founder #27 · Platinum', exact=True)",
    "owner.get_by_text(f'Founder #{founder_number} · Platinum', exact=True)",
    "founder badge",
)
replace_once(
    r"\$o->forceFill(['founder_number'=>27])->save();",
    r"\$used=App\\Models\\User::query()->whereNotNull('founder_number')->pluck('founder_number')->map(fn(\$n)=>(int)\$n)->all(); \$founder=null; foreach(range(1,100) as \$candidate){ if(!in_array(\$candidate,\$used,true)){ \$founder=\$candidate; break; }} if(\$founder===null){ throw new RuntimeException('No unused Platinum founder number is available for isolated profile verification.'); } \$o->forceFill(['founder_number'=>\$founder])->save();",
    "founder fixture assignment",
)
replace_once(
    r"echo \$o->id.'|'.\$o->profile->display_name;",
    r"echo \$o->id.'|'.\$o->profile->display_name.'|'.\$founder;",
    "fixture identity output",
)
replace_once(
    'owner_id="${identity_line%%|*}"; owner_name="${identity_line#*|}"',
    'owner_id="${identity_line%%|*}"; identity_rest="${identity_line#*|}"; founder_number="${identity_rest##*|}"; owner_name="${identity_rest%|*}"',
    "fixture identity parsing",
)
replace_once(
    '[[ "$owner_id" =~ ^[0-9a-z]{26}$ && -n "$owner_name" ]] || { echo "VERIFY BLOCKED: fixture identity setup failed: $identity_line" >&2; exit 70; }',
    '[[ "$owner_id" =~ ^[0-9a-z]{26}$ && -n "$owner_name" && "$founder_number" =~ ^[0-9]+$ && "$founder_number" -ge 1 && "$founder_number" -le 100 ]] || { echo "VERIFY BLOCKED: fixture identity setup failed: $identity_line" >&2; exit 70; }\nprintf \'RUM_PROFILE_FOUNDER_FIXTURE=%s\\n\' "$founder_number"',
    "fixture identity validation",
)
replace_once(
    "    rating_id=str(response.json().get('data',{}).get('id',''))\n    if not rating_id: raise RuntimeError('profile fixture rating returned no canonical rating id')",
    "    rating_id='submitted'",
    "rating response body dependency",
)
replace_once(
    'rating_id="$(cat "$work/rating-id")"\n[[ "$rating_id" =~ ^[0-9a-z]{26}$ ]] || { echo "VERIFY BLOCKED: invalid profile fixture rating id" >&2; exit 70; }',
    'rating_id="$(kctl -n "$DEV_NAMESPACE" exec "$api_pod" -c php-fpm -- php -r "$PHP_BOOT" "\\$o=App\\\\Models\\\\User::where(\'email\',base64_decode(\'${owner_b64}\'))->firstOrFail(); \\$r=App\\\\Models\\\\User::where(\'email\',base64_decode(\'${rater_b64}\'))->firstOrFail(); \\$target=App\\\\Models\\\\AccountEntityLink::query()->where(\'user_id\',\\$o->id)->where(\'link_type\',\'represents_person\')->where(\'status\',\'active\')->whereNull(\'valid_to\')->value(\'entity_id\'); echo App\\\\Models\\\\RatingEvent::query()->where(\'rater_account_id\',\\$r->id)->where(\'target_entity_id\',\\$target)->where(\'status\',\'active\')->orderByDesc(\'submitted_at\')->value(\'id\');")"\n[[ "$rating_id" =~ ^[0-9a-z]{26}$ ]] || { echo "VERIFY BLOCKED: invalid saved profile fixture rating id: $rating_id" >&2; exit 70; }\nprintf \'RUM_PROFILE_SAVED_RATING_ID=%s\\n\' "$rating_id"',
    "saved rating id lookup",
)
replace_once(
    '-e RUM_BASE_URL="$BASE_URL" -e RUM_OWNER_ID="$owner_id" -e RUM_OWNER_NAME="$owner_name" -e RUM_RATING_ID="$rating_id"',
    '-e RUM_BASE_URL="$BASE_URL" -e RUM_OWNER_ID="$owner_id" -e RUM_OWNER_NAME="$owner_name" -e RUM_RATING_ID="$rating_id" -e RUM_FOUNDER_NUMBER="$founder_number"',
    "browser verifier environment",
)
replace_once(
    "    with open('/work/profile-owner-mobile.png','rb') as fh:\n        print('profile_owner_mobile_png_base64='+base64.b64encode(fh.read()).decode())",
    "    with open('/work/profile-owner-mobile.png','rb') as fh:\n        payload=fh.read()\n    print(f'profile_owner_mobile_screenshot_sha256={hashlib.sha256(payload).hexdigest()} bytes={len(payload)}')",
    "mobile screenshot output",
)

path.write_text(text)
PY
chmod 0700 "$COPY"

bash "$COPY" "$@"
