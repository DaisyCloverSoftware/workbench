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

# Keep the base browser verifier authoritative, but provision its disposable
# identities and normal Laravel session cookies directly inside isolated DEV.
# This avoids consuming public auth-rate-limit budget while all Profile/Judge
# interactions still run through the deployed browser and HTTP APIs. Also make
# Founder allocation collision-safe, avoid reading a navigated response body,
# and return a bounded screenshot hash. Every replacement is fail-closed.
python3 - "$COPY" <<'PY'
from pathlib import Path
import sys

path = Path(sys.argv[1])
text = path.read_text()

def replace_once(old: str, new: str, label: str) -> None:
    global text
    count = text.count(old)
    if count != 1:
        raise SystemExit(f'profile verifier patch blocked: {label} marker count={count}')
    text = text.replace(old, new, 1)

replace_once(
    "    rating_id=str(response.json().get('data',{}).get('id',''))\n    if not rating_id: raise RuntimeError('profile fixture rating returned no canonical rating id')",
    "    rating_id='submitted'",
    "rating response body dependency",
)
replace_once(
    "import os,base64\n",
    "import os,hashlib\n",
    "screenshot import",
)
replace_once(
    "base=os.environ['RUM_BASE_URL']; owner_id=os.environ['RUM_OWNER_ID']; owner_name=os.environ['RUM_OWNER_NAME']; rating_id=os.environ['RUM_RATING_ID']",
    "base=os.environ['RUM_BASE_URL']; owner_id=os.environ['RUM_OWNER_ID']; owner_name=os.environ['RUM_OWNER_NAME']; rating_id=os.environ['RUM_RATING_ID']; founder_number=os.environ['RUM_FOUNDER_NUMBER']",
    "profile verifier environment",
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
    "    with open('/work/profile-owner-mobile.png','rb') as fh:\n        print('profile_owner_mobile_png_base64='+base64.b64encode(fh.read()).decode())",
    "    with open('/work/profile-owner-mobile.png','rb') as fh:\n        payload=fh.read()\n    print(f'profile_owner_mobile_screenshot_sha256={hashlib.sha256(payload).hexdigest()} bytes={len(payload)}')",
    "mobile screenshot output",
)

start = text.find('unset TOKEN GH_TOKEN GHCR_TOKEN\nregister(){')
end_marker = '[[ "$owner_id" =~ ^[0-9a-z]{26}$ && -n "$owner_name" ]] || { echo "VERIFY BLOCKED: fixture identity setup failed: $identity_line" >&2; exit 70; }\n'
end = text.find(end_marker, start)
if start < 0 or end < 0:
    raise SystemExit('profile verifier patch blocked: isolated identity setup markers unavailable')
end += len(end_marker)
identity_block = r'''unset TOKEN GH_TOKEN GHCR_TOKEN
b64(){ printf '%s' "$1" | python3 -c 'import base64,sys; print(base64.b64encode(sys.stdin.buffer.read()).decode())'; }
decode64(){ printf '%s' "$1" | python3 -c 'import base64,sys; print(base64.b64decode(sys.stdin.read()).decode())'; }
owner_b64="$(b64 "$owner_email")"; rater_b64="$(b64 "$rater_email")"; viewer_b64="$(b64 "$viewer_email")"
owner_user_b64="$(b64 "$owner_username")"; rater_user_b64="$(b64 "$rater_username")"; viewer_user_b64="$(b64 "$viewer_username")"
PHP_BOOT='require "/var/www/html/vendor/autoload.php"; $app=require "/var/www/html/bootstrap/app.php"; $app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap(); eval($argv[1]);'
identity_line="$(kctl -n "$DEV_NAMESPACE" exec "$api_pod" -c php-fpm -- php -r "$PHP_BOOT" "
\$make=function(\$email64,\$username64){
  \$email=base64_decode(\$email64); \$username=base64_decode(\$username64);
  \$u=App\\Models\\User::query()->create(['email'=>\$email,'username'=>\$username,'password'=>Illuminate\\Support\\Str::random(48),'status'=>'active','age_status'=>'pending','terms_accepted_at'=>now(),'privacy_accepted_at'=>now(),'last_seen_at'=>now()]);
  \$u->forceFill(['email_verified_at'=>now(),'created_at'=>now()->subDays(30)])->save();
  App\\Models\\Profile::query()->create(['user_id'=>\$u->id,'display_name'=>\$username,'status_text'=>'Profile isolated browser fixture','presence_visibility'=>'mates','profile_visibility'=>'members','rating_permission'=>'mates','message_permission'=>'mates','proximity_opt_in'=>false,'avatar_tone'=>1]);
  return \$u;
};
\$o=\$make('${owner_b64}','${owner_user_b64}'); \$r=\$make('${rater_b64}','${rater_user_b64}'); \$v=\$make('${viewer_b64}','${viewer_user_b64}');
\$used=App\\Models\\User::query()->whereNotNull('founder_number')->pluck('founder_number')->map(fn(\$n)=>(int)\$n)->all(); \$founder=null; foreach(range(1,100) as \$candidate){ if(!in_array(\$candidate,\$used,true)){ \$founder=\$candidate; break; }} if(\$founder===null){ throw new RuntimeException('No unused Platinum founder number is available for isolated profile verification.'); }
\$o->forceFill(['founder_number'=>\$founder])->save();
App\\Models\\MateRelationship::query()->create(['requester_id'=>\$o->id,'addressee_id'=>\$r->id,'status'=>'accepted','accepted_at'=>now()]);
echo \$o->id."\\t".base64_encode(\$o->profile->display_name)."\\t".\$founder;")"
IFS=$'\t' read -r owner_id owner_name_b64 founder_number <<<"$identity_line"
owner_name="$(decode64 "$owner_name_b64")"
[[ "$owner_id" =~ ^[0-9a-z]{26}$ && -n "$owner_name" && "$founder_number" =~ ^[0-9]+$ && "$founder_number" -ge 1 && "$founder_number" -le 100 ]] || { echo "VERIFY BLOCKED: isolated fixture identity setup failed" >&2; exit 70; }
printf 'RUM_PROFILE_FOUNDER_FIXTURE=%s\n' "$founder_number"

session_cookie(){
  local email_b64="$1"
  kctl -n "$DEV_NAMESPACE" exec "$api_pod" -c php-fpm -- php -r "$PHP_BOOT" "
  \$u=App\\Models\\User::where('email',base64_decode('${email_b64}'))->firstOrFail();
  \$session=app('session')->driver(); \$session->setId(Illuminate\\Support\\Str::random(40)); \$session->start();
  \$session->put(Illuminate\\Support\\Facades\\Auth::guard('web')->getName(),\$u->getAuthIdentifier()); \$session->save();
  \$name=(string)config('session.cookie'); \$prefix=Illuminate\\Cookie\\CookieValuePrefix::create(\$name,app('encrypter')->getKey());
  \$value=app('encrypter')->encrypt(\$prefix.\$session->getId(),false);
  echo base64_encode(\$name)."\\t".base64_encode(\$value);" 2>/dev/null | tail -n 1
}
write_state(){
  local email_b64="$1" output="$2" line cookie_name_b64 cookie_value_b64 cookie_name cookie_value
  line="$(session_cookie "$email_b64")"
  IFS=$'\t' read -r cookie_name_b64 cookie_value_b64 <<<"$line"
  cookie_name="$(decode64 "$cookie_name_b64")"; cookie_value="$(decode64 "$cookie_value_b64")"
  [[ -n "$cookie_name" && -n "$cookie_value" ]] || { echo "VERIFY BLOCKED: isolated Laravel session cookie generation failed" >&2; exit 70; }
  python3 - "$output" "$cookie_name" "$cookie_value" "$DEV_HOST" <<'PY_STATE'
import json,sys
path,name,value,domain=sys.argv[1:]
state={'cookies':[{'name':name,'value':value,'domain':domain,'path':'/','expires':-1,'httpOnly':True,'secure':True,'sameSite':'Lax'}],'origins':[]}
with open(path,'w') as fh: json.dump(state,fh)
PY_STATE
}
write_state "$owner_b64" "$work/owner-state.json"
write_state "$rater_b64" "$work/rater-state.json"
write_state "$viewer_b64" "$work/viewer-state.json"
printf 'RUM_PROFILE_AUTH_FIXTURE=ISOLATED_LARAVEL_SESSIONS\n'
'''
text = text[:start] + identity_block + text[end:]

replace_once(
    'rating_id="$(cat "$work/rating-id")"\n[[ "$rating_id" =~ ^[0-9a-z]{26}$ ]] || { echo "VERIFY BLOCKED: invalid profile fixture rating id" >&2; exit 70; }',
    'rating_id="$(kctl -n "$DEV_NAMESPACE" exec "$api_pod" -c php-fpm -- php -r "$PHP_BOOT" "\\$o=App\\\\Models\\\\User::where(\'email\',base64_decode(\'${owner_b64}\'))->firstOrFail(); \\$r=App\\\\Models\\\\User::where(\'email\',base64_decode(\'${rater_b64}\'))->firstOrFail(); \\$target=App\\\\Models\\\\AccountEntityLink::query()->where(\'user_id\',\\$o->id)->where(\'link_type\',\'represents_person\')->where(\'status\',\'active\')->whereNull(\'valid_to\')->value(\'entity_id\'); echo App\\\\Models\\\\RatingEvent::query()->where(\'rater_account_id\',\\$r->id)->where(\'target_entity_id\',\\$target)->where(\'status\',\'active\')->orderByDesc(\'submitted_at\')->value(\'id\');")"\n[[ "$rating_id" =~ ^[0-9a-z]{26}$ ]] || { echo "VERIFY BLOCKED: invalid saved profile fixture rating id" >&2; exit 70; }\nprintf \'RUM_PROFILE_SAVED_RATING_ID=%s\\n\' "$rating_id"',
    "saved rating id lookup",
)
replace_once(
    '-e RUM_BASE_URL="$BASE_URL" -e RUM_OWNER_ID="$owner_id" -e RUM_OWNER_NAME="$owner_name" -e RUM_RATING_ID="$rating_id"',
    '-e RUM_BASE_URL="$BASE_URL" -e RUM_OWNER_ID="$owner_id" -e RUM_OWNER_NAME="$owner_name" -e RUM_RATING_ID="$rating_id" -e RUM_FOUNDER_NUMBER="$founder_number"',
    "founder browser environment",
)

path.write_text(text)
PY
chmod 0700 "$COPY"

bash "$COPY" "$@"
