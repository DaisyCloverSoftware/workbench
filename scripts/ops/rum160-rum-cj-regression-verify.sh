#!/usr/bin/env bash
set -euo pipefail
umask 077

SOURCE_SHA="${1:-}"
[[ "$SOURCE_SHA" =~ ^[0-9a-f]{40}$ ]] || { echo "usage: $0 <40-char-source-sha>" >&2; exit 64; }

RUM_NS="rum-dev-isolated"
RUM_HOST="dev-rum.daisycloversoftware.uk"
RUM_BASE_URL="https://${RUM_HOST}"
PUBLIC_HOST="rateurmate.online"
PLAYWRIGHT_IMAGE="mcr.microsoft.com/playwright/python:v1.57.0-noble"

for command in docker python3 curl; do
  command -v "$command" >/dev/null 2>&1 || { echo "required command unavailable: $command" >&2; exit 3; }
done

kctl() {
  if command -v k3s >/dev/null 2>&1; then
    sudo -n k3s kubectl "$@"
  else
    kubectl "$@"
  fi
}

hosts="$(kctl -n "$RUM_NS" get ingress -o jsonpath='{range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)"
[[ "$hosts" == "$RUM_HOST" ]] || { echo "blocked: isolated RUM host mismatch" >&2; exit 78; }
if grep -Fxq "$PUBLIC_HOST" <<<"$hosts"; then
  echo "blocked: public host present in isolated RUM namespace" >&2
  exit 78
fi

health="$(curl -fsS -H 'Cache-Control: no-cache' --retry 5 --retry-delay 2 --max-time 20 "$RUM_BASE_URL/api/v1/health")"
python3 - "$health" "$SOURCE_SHA" <<'PY'
import json,sys
obj=json.loads(sys.argv[1])
assert obj.get('status') == 'ok', obj
assert obj.get('service') == 'rum-api', obj
assert sys.argv[2] in str(obj.get('version') or ''), obj
PY

api_pod="$(kctl -n "$RUM_NS" get pods \
  -l 'app.kubernetes.io/instance=rum,app.kubernetes.io/component=api' \
  --field-selector=status.phase=Running \
  -o jsonpath='{.items[0].metadata.name}')"
[[ -n "$api_pod" ]] || { echo "isolated RUM API pod unavailable" >&2; exit 78; }

work="$(mktemp -d)"
cleanup() { rm -rf "$work"; }
trap cleanup EXIT HUP INT TERM
chmod 700 "$work"

suffix="$(date +%s)-$(python3 -c 'import secrets; print(secrets.token_hex(2))')"
username="rumcj${suffix//-/}"
username="${username:0:28}"
email="${username}@example.com"
password="$(python3 -c 'import secrets; print("RUM-" + secrets.token_urlsafe(20) + "!Aa9")')"
state="$work/state.json"

cat >"$work/register.py" <<'PY'
import os
from playwright.sync_api import sync_playwright
base=os.environ['RUM_BASE_URL']
with sync_playwright() as p:
    browser=p.chromium.launch()
    context=browser.new_context(viewport={'width':390,'height':844})
    page=context.new_page()
    page.goto(base, wait_until='networkidle', timeout=60000)
    page.get_by_role('button', name='Create an account').click()
    page.get_by_label('Email').fill(os.environ['RUM_EMAIL'])
    page.get_by_label('Username').fill(os.environ['RUM_USERNAME'])
    page.get_by_label('Password', exact=True).fill(os.environ['RUM_PASSWORD'])
    page.get_by_label('Confirm password').fill(os.environ['RUM_PASSWORD'])
    page.get_by_role('checkbox', name='I accept the Terms and Community Rules.').check()
    page.get_by_role('checkbox', name='I have read the Privacy Notice.').check()
    page.get_by_role('button', name='Create account').click()
    page.wait_for_url('**/me', timeout=30000)
    context.storage_state(path='/work/state.json')
    browser.close()
PY

docker run --rm --ipc=host \
  -v "$work:/work" \
  -e RUM_BASE_URL="$RUM_BASE_URL" \
  -e RUM_EMAIL="$email" \
  -e RUM_USERNAME="$username" \
  -e RUM_PASSWORD="$password" \
  "$PLAYWRIGHT_IMAGE" python /work/register.py
[[ -s "$state" ]] || { echo "RUM registration session was not captured" >&2; exit 1; }

email_b64="$(printf '%s' "$email" | python3 -c 'import base64,sys; print(base64.b64encode(sys.stdin.buffer.read()).decode())')"
kctl -n "$RUM_NS" exec "$api_pod" -c php-fpm -- php artisan tinker --execute="\
\$u=App\\Models\\User::where('email',base64_decode('${email_b64}'))->firstOrFail(); \
\$u->forceFill(['email_verified_at'=>now()])->save();" >/dev/null

cat >"$work/verify.py" <<'PY'
import json,os
from playwright.sync_api import sync_playwright
base=os.environ['RUM_BASE_URL']
source=os.environ['SOURCE_SHA']
result={}
with sync_playwright() as p:
    browser=p.chromium.launch()
    context=browser.new_context(storage_state='/work/state.json', viewport={'width':390,'height':844})
    page=context.new_page()
    errors=[]
    page.on('pageerror', lambda error: errors.append(str(error)))
    page.goto(base + '/rate', wait_until='networkidle', timeout=60000)
    assert page.url.startswith(base + '/rate'), page.url
    search=page.get_by_role('searchbox', name='Search RUM members and public identities')
    search.wait_for(state='visible', timeout=30000)
    search.fill('CJ')
    card=page.locator('.person-card').filter(has=page.get_by_role('heading', name='CJ Investigates')).first
    card.wait_for(state='visible', timeout=30000)
    card.get_by_role('button', name='Rate', exact=True).click()
    panel=page.locator('.entity-bridge__public-rating')
    panel.wait_for(state='visible', timeout=30000)
    page.get_by_role('heading', name='Rate CJ Investigates').wait_for(state='visible', timeout=30000)
    assert page.url.startswith(base), page.url
    page.get_by_role('radio', name='Thumbs up 5 of 5 — Exceptional').click()
    category_panel=page.locator('.rating-reason-panel').filter(has_text='2. Rating category').first
    category_button=category_panel.locator('button.category-choice').first
    category_button.wait_for(state='visible', timeout=15000)
    category_button.click()
    reason_panel=page.locator('.rating-reason-panel').filter(has_text='3. Why this verdict?').first
    reason_button=reason_panel.locator('button.category-choice').first
    reason_button.wait_for(state='visible', timeout=15000)
    reason_button.click()
    page.get_by_text('4. Optional supporting evidence', exact=True).wait_for(state='visible', timeout=15000)
    page.get_by_text('5. Review and submit', exact=True).wait_for(state='visible', timeout=15000)
    page.get_by_role('button', name='Submit RUM rating', exact=True).click()
    page.get_by_text('Rating saved', exact=True).first.wait_for(state='visible', timeout=30000)
    assert page.url.startswith(base), page.url
    assert not errors, errors
    result={'result':'PASS','sourceSha':source,'host':base,'target':'CJ Investigates','viewport':{'width':390,'height':844}}
    browser.close()
print(json.dumps(result, separators=(',',':')))
PY

verification="$(docker run --rm --ipc=host \
  -v "$work:/work" \
  -e RUM_BASE_URL="$RUM_BASE_URL" \
  -e SOURCE_SHA="$SOURCE_SHA" \
  "$PLAYWRIGHT_IMAGE" python /work/verify.py)"

printf 'RUM160_RUM_CJ_BROWSER=%s\n' "$verification"
printf 'RUM160_RUM_CJ_MOBILE_FLOW_PASS=true\n'
