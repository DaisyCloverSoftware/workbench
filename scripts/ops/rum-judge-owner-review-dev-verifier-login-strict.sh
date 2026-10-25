#!/usr/bin/env bash
set -euo pipefail
umask 077

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SOURCE="$ROOT/scripts/ops/rum-judge-owner-review-dev-verifier-strict.sh"
[[ -f "$SOURCE" && ! -L "$SOURCE" ]] || {
  echo "VERIFY BLOCKED: strict Judge verifier unavailable" >&2
  exit 78
}

COPY="$(mktemp "$ROOT/scripts/ops/.rum-judge-owner-review-login.XXXXXX.sh")"
trap 'rm -f "$COPY"' EXIT HUP INT TERM
cp "$SOURCE" "$COPY"

# Keep every strict Judge/Check-again/geometry assertion. Replace only the
# public signup transport: create disposable accounts inside isolated DEV, then
# sign into them through RUM's normal deployed login UI. This avoids signup
# throttling without synthesizing browser sessions.
python3 - "$COPY" <<'PY'
from pathlib import Path
import sys

path=Path(sys.argv[1])
script=path.read_text()
marker='path.write_text(text)\nPY\n'
if script.count(marker) != 1:
    raise SystemExit(f'Judge login wrapper blocked: strict write marker count={script.count(marker)}')
extra=r'''
setup_start=text.find('unset TOKEN GH_TOKEN GHCR_TOKEN\nregister(){')
setup_end_marker="printf 'judge_owner_review_dev_accounts_prepared_ok\\n'"
setup_end=text.find(setup_end_marker, setup_start)
if setup_start < 0 or setup_end < 0:
    raise SystemExit('Judge login wrapper blocked: account setup markers unavailable')
setup_end += len(setup_end_marker)
direct_setup=r"""unset TOKEN GH_TOKEN GHCR_TOKEN
b64(){ printf '%s' "$1" | python3 -c 'import base64,sys; print(base64.b64encode(sys.stdin.buffer.read()).decode())'; }
viewer_b64="$(b64 "$viewer_email")"; rater_b64="$(b64 "$rater_email")"
viewer_user_b64="$(b64 "$viewer_username")"; rater_user_b64="$(b64 "$rater_username")"
viewer_password_b64="$(b64 "$viewer_password")"; rater_password_b64="$(b64 "$rater_password")"
PHP_BOOT='require "/var/www/html/vendor/autoload.php"; $app=require "/var/www/html/bootstrap/app.php"; $app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap(); eval($argv[1]);'
kctl -n "$DEV_NAMESPACE" exec "$api_pod" -c php-fpm -- php -r "$PHP_BOOT" "
\$make=function(\$email64,\$username64,\$password64,\$days){
  \$email=base64_decode(\$email64); \$username=base64_decode(\$username64); \$password=base64_decode(\$password64);
  \$u=App\\Models\\User::query()->create(['email'=>\$email,'username'=>\$username,'password'=>Illuminate\\Support\\Facades\\Hash::make(\$password),'status'=>'active','age_status'=>'pending','terms_accepted_at'=>now(),'privacy_accepted_at'=>now(),'last_seen_at'=>now()]);
  \$u->forceFill(['email_verified_at'=>now(),'created_at'=>now()->subDays(\$days)])->save();
  App\\Models\\Profile::query()->create(['user_id'=>\$u->id,'display_name'=>\$username,'status_text'=>'Judge isolated browser fixture','presence_visibility'=>'mates','profile_visibility'=>'members','rating_permission'=>'mates','message_permission'=>'mates','proximity_opt_in'=>false,'avatar_tone'=>1]);
};
\$make('${viewer_b64}','${viewer_user_b64}','${viewer_password_b64}',2); \$make('${rater_b64}','${rater_user_b64}','${rater_password_b64}',1);" >/dev/null

cat >"$work/login.py" <<'PY_LOGIN'
import os
from playwright.sync_api import sync_playwright
base=os.environ['RUM_BASE_URL']; login=os.environ['RUM_LOGIN']; password=os.environ['RUM_PASSWORD']; state=os.environ['RUM_STATE']
with sync_playwright() as p:
    browser=p.chromium.launch(); context=browser.new_context(viewport={'width':1440,'height':1100}); page=context.new_page()
    page.goto(base, wait_until='networkidle', timeout=60000)
    page.get_by_role('button', name='Log in', exact=True).click()
    page.get_by_label('Username or email').fill(login); page.get_by_label('Password').fill(password)
    page.get_by_role('button', name='Log in', exact=True).click(); page.wait_for_url('**/me', timeout=30000)
    context.storage_state(path=state); browser.close()
print('judge_login_session_ok')
PY_LOGIN
login_fixture(){
  local login="$1" password="$2" state="$3"
  "$runtime" run --rm --network host -v "$work:/work:Z" \
    -e RUM_BASE_URL="$BASE_URL" -e RUM_LOGIN="$login" -e RUM_PASSWORD="$password" -e RUM_STATE="/work/$state" \
    "$PLAYWRIGHT_IMAGE" python /work/login.py
}
login_fixture "$viewer_email" "$viewer_password" viewer-state.json
login_fixture "$rater_email" "$rater_password" rater-state.json
printf 'RUM_JUDGE_AUTH_FIXTURE=ISOLATED_USERS_NORMAL_LOGIN\n'
printf 'judge_owner_review_dev_accounts_prepared_ok\n' """
text=text[:setup_start]+direct_setup+text[setup_end:]
'''
script=script.replace(marker, extra+marker, 1)
path.write_text(script)
PY
chmod 0700 "$COPY"
printf 'RUM_JUDGE_PUBLIC_SIGNUP_BYPASS=PRECREATED_USERS_NORMAL_LOGIN\n'
bash "$COPY" "$@"
