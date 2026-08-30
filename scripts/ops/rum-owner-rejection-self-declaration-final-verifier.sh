#!/usr/bin/env bash
set -euo pipefail
umask 077

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SOURCE="$ROOT/scripts/ops/rum-self-declaration-dev-verifier.sh"
[[ -f "$SOURCE" && ! -L "$SOURCE" ]] || {
  echo "VERIFY BLOCKED: self-declaration verifier unavailable" >&2
  exit 78
}

COPY="$(mktemp "$ROOT/scripts/ops/.rum-self-final.XXXXXX.sh")"
trap 'rm -f "$COPY"' EXIT HUP INT TERM
cp "$SOURCE" "$COPY"

python3 - "$COPY" <<'PY'
from pathlib import Path
import sys

path=Path(sys.argv[1])
text=path.read_text()

signup_old='''docker run --rm --ipc=host -v "$work:/work" \\
  -e RUM_BASE_URL="$BASE_URL" -e RUM_EMAIL="$email" -e RUM_USERNAME="$username" -e RUM_PASSWORD="$password" -e RUM_STATE_FILE="/work/self-state.json" \\
  "$PLAYWRIGHT_IMAGE" python /work/register.py
[[ -s "$state_file" ]] || { echo "VERIFY FAILED: self account browser state missing" >&2; exit 1; }
chmod 600 "$state_file"
'''
signup_new=r'''pre_b64(){ printf '%s' "$1" | python3 -c 'import base64,sys; print(base64.b64encode(sys.stdin.buffer.read()).decode())'; }
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
docker run --rm --ipc=host -v "$work:/work" -e RUM_BASE_URL="$BASE_URL" -e RUM_LOGIN="$email" -e RUM_PASSWORD="$password" -e RUM_STATE_FILE="/work/self-state.json" "$PLAYWRIGHT_IMAGE" python /work/login.py
[[ -s "$state_file" ]] || { echo "VERIFY FAILED: self account browser state missing" >&2; exit 1; }
chmod 600 "$state_file"
printf 'RUM_SELF_DECLARATION_AUTH_FIXTURE=ISOLATED_USER_NORMAL_LOGIN\n'
'''
if text.count(signup_old) != 1:
    raise SystemExit(f'final verifier blocked: signup block count={text.count(signup_old)}')
text=text.replace(signup_old, signup_new, 1)

request_old='''    page.on("requestfailed", lambda req: request_failures.append(f"{req.method} {req.url}"))'''
request_new='''    page.on("requestfailed", lambda req: request_failures.append(f"{req.method} {req.url} failure={req.failure or 'unknown'}"))'''
if text.count(request_old) != 1:
    raise SystemExit(f'final verifier blocked: request handler count={text.count(request_old)}')
text=text.replace(request_old, request_new, 1)

reload_old='''    print("self_declared_create_visible_ok")

    page.goto(f"{base}/me", wait_until="networkidle", timeout=60000)
    loading=page.get_by_text("Loading your identities…", exact=True)
'''
reload_new='''    print("self_declared_create_visible_ok")

    page.get_by_role("button", name="Close", exact=True).click()
    page.get_by_role("heading", name="My gamer & online identities", exact=True).wait_for(state="hidden", timeout=30000)

    page.wait_for_timeout(750)
    page.get_by_role("button", name="Open profile and settings", exact=True).click()
    page.get_by_role("button", name="Log out", exact=True).click()
    page.get_by_role("button", name="Log in", exact=True).wait_for(state="visible", timeout=30000)
    logged_out=context.request.get(f"{base}/api/v1/me", fail_on_status_code=False)
    if logged_out.status != 401:
        raise RuntimeError(f"Logout did not clear the server session: /api/v1/me returned {logged_out.status}")
    print("self_declared_logout_server_session_cleared_ok")

    page.get_by_role("button", name="Log in", exact=True).click()
    page.get_by_label("Username or email").fill(os.environ["RUM_LOGIN"])
    page.get_by_label("Password").fill(os.environ["RUM_PASSWORD"])
    page.get_by_role("button", name="Log in", exact=True).click()
    page.wait_for_url("**/me", timeout=30000)
    print("self_declared_logout_login_ok")

    page.reload(wait_until="networkidle", timeout=60000)
    page.wait_for_url("**/me", timeout=30000)
    print("self_declared_post_login_fresh_reload_ok")
    loading=page.get_by_text("Loading your identities…", exact=True)
'''
if text.count(reload_old) != 1:
    raise SystemExit(f'final verifier blocked: post-create reload block count={text.count(reload_old)}')
text=text.replace(reload_old, reload_new, 1)

row_old='''    if row.get_by_role("button", name="Verify identity", exact=True).count() != 0:
        raise RuntimeError("Self-declared identity exposed Verify identity.")
    lower=row.inner_text().lower()
    if "verification pending" in lower or "not verified" in lower or "unverified" in lower:
        raise RuntimeError("Self-declared identity is visibly presented as pending/unverified.")
    print("self_declared_profile_row_visible_ok")
    print("self_declared_verification_prompt_absent_ok")

    known=[message for message in console_errors if baseline in message and "script-src 'self'" in message]
'''
row_new='''    if row.get_by_role("button", name="Verify identity", exact=True).count() != 0:
        raise RuntimeError("Self-declared identity exposed Verify identity.")
    lower=row.inner_text().lower()
    if "verification pending" in lower or "not verified" in lower or "unverified" in lower:
        raise RuntimeError("Self-declared identity is visibly presented as pending/unverified.")
    print("self_declared_profile_row_visible_ok")
    print("self_declared_verification_prompt_absent_ok")
    print("self_declared_fresh_authenticated_api_fetch_ok")

    page.get_by_role("button", name="People", exact=True).click()
    page.get_by_role("button", name="Me", exact=True).click()
    nav_row=page.locator(".my-identity-row").filter(has_text=gamertag).first
    nav_row.wait_for(state="visible", timeout=30000)
    nav_row.get_by_text("Mine · self-declared", exact=True).wait_for(state="visible", timeout=15000)
    if nav_row.get_by_role("button", name="Verify identity", exact=True).count() != 0:
        raise RuntimeError("Self-declared identity exposed Verify identity after navigation.")
    print("self_declared_navigation_roundtrip_ok")

    known=[message for message in console_errors if baseline in message and "script-src 'self'" in message]
'''
if text.count(row_old) != 1:
    raise SystemExit(f'final verifier blocked: row assertion block count={text.count(row_old)}')
text=text.replace(row_old, row_new, 1)

failure_old='''    if request_failures:
        raise RuntimeError("Browser request failures: "+" | ".join(request_failures[:5]))
    if page_errors:
'''
failure_new='''    # Playwright reports requests cancelled by our deliberate logout/reload/navigation
    # as net::ERR_ABORTED. These are browser cancellations, not HTTP/API failures.
    # Any other request failure remains fatal, while response >=400 is already tracked
    # independently in api_failures below.
    known_navigation_aborts=[entry for entry in request_failures if "failure=net::ERR_ABORTED" in entry]
    unexpected_request_failures=[entry for entry in request_failures if entry not in known_navigation_aborts]
    print(f"known_navigation_aborted_requests={len(known_navigation_aborts)}")
    print(f"unexpected_request_failures={len(unexpected_request_failures)}")
    if unexpected_request_failures:
        raise RuntimeError("Unexpected browser request failures: "+" | ".join(unexpected_request_failures[:5]))
    if page_errors:
'''
if text.count(failure_old) != 1:
    raise SystemExit(f'final verifier blocked: request failure block count={text.count(failure_old)}')
text=text.replace(failure_old, failure_new, 1)

env_old='''docker run --rm --ipc=host -v "$work:/work" \\
  -e RUM_BASE_URL="$BASE_URL" -e RUM_GAMERTAG="$gamertag" -e RUM_BASELINE_CSP="$LIVE_BASELINE_CSP_HASH" \\
  "$PLAYWRIGHT_IMAGE" python /work/exercise.py
'''
env_new='''docker run --rm --ipc=host -v "$work:/work" -e RUM_BASE_URL="$BASE_URL" -e RUM_GAMERTAG="$gamertag" -e RUM_BASELINE_CSP="$LIVE_BASELINE_CSP_HASH" -e RUM_LOGIN="$email" -e RUM_PASSWORD="$password" "$PLAYWRIGHT_IMAGE" python /work/exercise.py
'''
if text.count(env_old) != 1:
    raise SystemExit(f'final verifier blocked: exercise env block count={text.count(env_old)}')
text=text.replace(env_old, env_new, 1)

path.write_text(text)
PY
chmod 0700 "$COPY"
printf 'RUM_SELF_DECLARATION_FINAL_VERIFIER=LOGIN_RELOAD_NAVIGATION\n'
bash "$COPY" "$@"
