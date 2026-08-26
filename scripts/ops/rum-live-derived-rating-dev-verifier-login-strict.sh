#!/usr/bin/env bash
set -euo pipefail
umask 077

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SOURCE="$ROOT/scripts/ops/rum-live-derived-rating-dev-verifier-strict.sh"
[[ -f "$SOURCE" && ! -L "$SOURCE" ]] || {
  echo "VERIFY BLOCKED: strict rating verifier unavailable" >&2
  exit 78
}

COPY="$(mktemp "$ROOT/scripts/ops/.rum-rating-login-strict.XXXXXX.sh")"
trap 'rm -f "$COPY"' EXIT HUP INT TERM
cp "$SOURCE" "$COPY"

# Preserve the strict rating-flow assertions and replace only the public signup
# transport. Disposable users are created inside isolated DEV, then sign in via
# the deployed RUM login form. No browser session is synthesized. The owner-
# rejected repeated linked-Thing search is also asserted here: the query typed
# into the linked chooser must be carried into the global RUM duplicate check
# automatically, with no second required search field or button action.
python3 - "$COPY" <<'PY'
from pathlib import Path
import sys

path=Path(sys.argv[1])
script=path.read_text()
marker='Path(sys.argv[2]).write_text(out)\nPY\n'
if script.count(marker) != 1:
    raise SystemExit(f'rating login wrapper blocked: strict write marker count={script.count(marker)}')
extra=r'''
login_old='register(){ local email="$1" user="$2" pass="$3" state="$4"; docker run --rm --ipc=host -v "$work:/work" -e RUM_BASE_URL="$BASE_URL" -e RUM_EMAIL="$email" -e RUM_USERNAME="$user" -e RUM_PASSWORD="$pass" -e RUM_STATE="/work/$state" "$PLAYWRIGHT_IMAGE" python /work/register.py; }'
login_new=r"""enc64(){ printf '%s' "$1" | python3 -c 'import base64,sys; print(base64.b64encode(sys.stdin.buffer.read()).decode())'; }
rater_email_pre="$(enc64 "$rater_email")"; mate_email_pre="$(enc64 "$mate_email")"
rater_user_pre="$(enc64 "$rater_username")"; mate_user_pre="$(enc64 "$mate_username")"
rater_pass_pre="$(enc64 "$rater_password")"; mate_pass_pre="$(enc64 "$mate_password")"
PHP_BOOT_PRE='require "/var/www/html/vendor/autoload.php"; $app=require "/var/www/html/bootstrap/app.php"; $app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap(); eval($argv[1]);'
kctl -n "$DEV_NAMESPACE" exec "$api_pod" -c php-fpm -- php -r "$PHP_BOOT_PRE" "
\$make=function(\$email64,\$username64,\$password64){
  \$email=base64_decode(\$email64); \$username=base64_decode(\$username64); \$password=base64_decode(\$password64);
  \$u=App\\Models\\User::query()->create(['email'=>\$email,'username'=>\$username,'password'=>Illuminate\\Support\\Facades\\Hash::make(\$password),'status'=>'active','age_status'=>'pending','terms_accepted_at'=>now(),'privacy_accepted_at'=>now(),'last_seen_at'=>now()]);
  \$u->forceFill(['email_verified_at'=>now(),'created_at'=>now()->subDay()])->save();
  App\\Models\\Profile::query()->create(['user_id'=>\$u->id,'display_name'=>\$username,'status_text'=>'Rating isolated browser fixture','presence_visibility'=>'mates','profile_visibility'=>'members','rating_permission'=>'mates','message_permission'=>'mates','proximity_opt_in'=>false,'avatar_tone'=>1]);
};
\$make('${rater_email_pre}','${rater_user_pre}','${rater_pass_pre}'); \$make('${mate_email_pre}','${mate_user_pre}','${mate_pass_pre}');" >/dev/null
cat >"$work/login.py" <<'PY_LOGIN'
import os
from playwright.sync_api import sync_playwright
base=os.environ['RUM_BASE_URL']; login=os.environ['RUM_LOGIN']; password=os.environ['RUM_PASSWORD']; state=os.environ['RUM_STATE']
with sync_playwright() as p:
    browser=p.chromium.launch(); context=browser.new_context(viewport={'width':1440,'height':1000}); page=context.new_page()
    page.goto(base, wait_until='networkidle', timeout=60000)
    page.get_by_role('button', name='Log in', exact=True).click()
    page.get_by_label('Username or email').fill(login); page.get_by_label('Password').fill(password)
    page.get_by_role('button', name='Log in', exact=True).click(); page.wait_for_url('**/me', timeout=30000)
    context.storage_state(path=state); browser.close()
print('rating_login_session_ok')
PY_LOGIN
register(){ local email="$1" user="$2" pass="$3" state="$4"; "$runtime" run --rm --network host -v "$work:/work:Z" -e RUM_BASE_URL="$BASE_URL" -e RUM_LOGIN="$email" -e RUM_PASSWORD="$pass" -e RUM_STATE="/work/$state" "$PLAYWRIGHT_IMAGE" python /work/login.py; }
"""
if out.count(login_old) != 1:
    raise SystemExit(f'rating login wrapper blocked: registration function count={out.count(login_old)}')
out=out.replace(login_old,login_new,1)

linked_old=r'''    outer.click(); chooser=page.locator("section.rating-reason-panel").filter(has_text="Add or rate a linked thing").first; chooser.wait_for(state="visible", timeout=15000); chooser.get_by_role("searchbox", name="Search linked things").wait_for(state="visible", timeout=15000); chooser.get_by_role("button", name="Add or rate a linked thing", exact=True).click()
    page.get_by_text("Search RUM before adding a missing linked thing", exact=True).wait_for(state="visible", timeout=15000); page.get_by_label("Search existing things before adding").fill(linked_name); page.get_by_role("button", name="Search RUM", exact=True).click(); page.get_by_role("button", name="None of these — add a linked thing", exact=True).wait_for(state="visible", timeout=30000); page.get_by_role("button", name="None of these — add a linked thing", exact=True).click(); page.get_by_label("Linked thing type").select_option("product"); page.get_by_label("Linked thing description").fill("Disposable linked Thing created only in isolated DEV to verify the LIVE-derived rating flow."); page.get_by_role("button", name="Check and add linked thing", exact=True).click(); rate(page,"search"); page.wait_for_url(re.compile(r"/judge(?:$|[?#])"), timeout=30000)'''
linked_new=r'''    outer.click(); chooser=page.locator("section.rating-reason-panel").filter(has_text="Add or rate a linked thing").first; chooser.wait_for(state="visible", timeout=15000); linked_search=chooser.get_by_role("searchbox", name="Search linked things"); linked_search.wait_for(state="visible", timeout=15000); linked_search.fill(linked_name); chooser.get_by_role("button", name="Add or rate a linked thing", exact=True).click()
    page.get_by_text(f"Checked RUM for “{linked_name}”", exact=True).wait_for(state="visible", timeout=30000)
    if page.get_by_label("Search existing things before adding").count()!=0: raise RuntimeError("Linked Thing flow asked for the same search a second time")
    if page.get_by_role("button", name="Search RUM", exact=True).count()!=0: raise RuntimeError("Linked Thing flow exposed a second mandatory Search RUM action")
    add_missing=page.get_by_role("button", name=f"None of these — add “{linked_name}”", exact=True); add_missing.wait_for(state="visible", timeout=30000); add_missing.click(); print("linked_query_single_entry_handoff_ok")
    page.get_by_label("Linked thing type").select_option("product"); page.get_by_label("Linked thing description").fill("Disposable linked Thing created only in isolated DEV to verify the LIVE-derived rating flow."); page.get_by_role("button", name="Check and add linked thing", exact=True).click(); rate(page,"search"); page.wait_for_url(re.compile(r"/judge(?:$|[?#])"), timeout=30000)'''
if out.count(linked_old) != 1:
    raise SystemExit(f'rating login wrapper blocked: linked-query flow count={out.count(linked_old)}')
out=out.replace(linked_old,linked_new,1)
'''
script=script.replace(marker, extra+marker, 1)
path.write_text(script)
PY
chmod 0700 "$COPY"
printf 'RUM_RATING_PUBLIC_SIGNUP_BYPASS=PRECREATED_USERS_NORMAL_LOGIN\n'
printf 'RUM_RATING_LINKED_QUERY_HANDOFF=SINGLE_ENTRY_REQUIRED\n'
bash "$COPY" "$@"
