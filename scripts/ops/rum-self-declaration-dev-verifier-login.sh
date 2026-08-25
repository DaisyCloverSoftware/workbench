#!/usr/bin/env bash
set -euo pipefail
umask 077

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SOURCE="$ROOT/scripts/ops/rum-self-declaration-dev-verifier.sh"
[[ -f "$SOURCE" && ! -L "$SOURCE" ]] || {
  echo "VERIFY BLOCKED: self-declaration verifier unavailable" >&2
  exit 78
}

COPY="$(mktemp "$ROOT/scripts/ops/.rum-self-login.XXXXXX.sh")"
trap 'rm -f "$COPY"' EXIT HUP INT TERM
cp "$SOURCE" "$COPY"

# Preserve every self-declaration browser/persistence assertion. Replace only
# public signup with an isolated disposable account that authenticates through
# the deployed RUM login UI. No browser session is synthesized.
python3 - "$COPY" <<'PY'
from pathlib import Path
import sys

path=Path(sys.argv[1])
text=path.read_text()
old='''docker run --rm --ipc=host -v "$work:/work" \\
  -e RUM_BASE_URL="$BASE_URL" -e RUM_EMAIL="$email" -e RUM_USERNAME="$username" -e RUM_PASSWORD="$password" -e RUM_STATE_FILE="/work/self-state.json" \\
  "$PLAYWRIGHT_IMAGE" python /work/register.py
[[ -s "$state_file" ]] || { echo "VERIFY FAILED: self account browser state missing" >&2; exit 1; }
chmod 600 "$state_file"
'''
new=r'''pre_b64(){ printf '%s' "$1" | python3 -c 'import base64,sys; print(base64.b64encode(sys.stdin.buffer.read()).decode())'; }
email_pre="$(pre_b64 "$email")"; user_pre="$(pre_b64 "$username")"; password_pre="$(pre_b64 "$password")"
PHP_BOOT_PRE='require "/var/www/html/vendor/autoload.php"; $app=require "/var/www/html/bootstrap/app.php"; $app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap(); eval($argv[1]);'
kctl -n "$DEV_NAMESPACE" exec "$api_pod" -c php-fpm -- php -r "$PHP_BOOT_PRE" "
\$email=base64_decode('${email_pre}'); \$username=base64_decode('${user_pre}'); \$password=base64_decode('${password_pre}');
\$u=App\\Models\\User::query()->create(['email'=>\$email,'username'=>\$username,'password'=>Illuminate\\Support\\Facades\\Hash::make(\$password),'status'=>'active','age_status'=>'pending','terms_accepted_at'=>now(),'privacy_accepted_at'=>now(),'last_seen_at'=>now()]);
\$u->forceFill(['email_verified_at'=>now(),'created_at'=>now()->subDay()])->save();
App\\Models\\Profile::query()->create(['user_id'=>\$u->id,'display_name'=>\$username,'status_text'=>'Self declaration isolated browser fixture','presence_visibility'=>'mates','profile_visibility'=>'members','rating_permission'=>'mates','message_permission'=>'mates','proximity_opt_in'=>false,'avatar_tone'=>1]);" >/dev/null
cat >"$work/login.py" <<'PY_LOGIN'
import os
from playwright.sync_api import sync_playwright
base=os.environ['RUM_BASE_URL']; login=os.environ['RUM_LOGIN']; password=os.environ['RUM_PASSWORD']; state=os.environ['RUM_STATE_FILE']
with sync_playwright() as p:
    browser=p.chromium.launch(); context=browser.new_context(viewport={'width':1440,'height':1000}); page=context.new_page()
    page.goto(base, wait_until='networkidle', timeout=60000)
    page.get_by_role('button', name='Log in', exact=True).click()
    page.get_by_label('Username or email').fill(login); page.get_by_label('Password').fill(password)
    page.get_by_role('button', name='Log in', exact=True).click(); page.wait_for_url('**/me', timeout=30000)
    context.storage_state(path=state); browser.close()
print('self_login_session_ok')
PY_LOGIN
docker run --rm --ipc=host -v "$work:/work" \
  -e RUM_BASE_URL="$BASE_URL" -e RUM_LOGIN="$email" -e RUM_PASSWORD="$password" -e RUM_STATE_FILE="/work/self-state.json" \
  "$PLAYWRIGHT_IMAGE" python /work/login.py
[[ -s "$state_file" ]] || { echo "VERIFY FAILED: self account browser state missing" >&2; exit 1; }
chmod 600 "$state_file"
printf 'RUM_SELF_DECLARATION_AUTH_FIXTURE=ISOLATED_USER_NORMAL_LOGIN\n'
'''
if text.count(old) != 1:
    raise SystemExit(f'self login wrapper blocked: signup block count={text.count(old)}')
path.write_text(text.replace(old,new,1))
PY
chmod 0700 "$COPY"
printf 'RUM_SELF_DECLARATION_PUBLIC_SIGNUP_BYPASS=PRECREATED_USER_NORMAL_LOGIN\n'
bash "$COPY" "$@"
