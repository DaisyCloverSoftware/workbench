#!/usr/bin/env bash
set -euo pipefail
umask 077

if [[ $# -ne 4 ]]; then
  echo "usage: $0 <40-char-source-sha> <canonical-cj-id> <rating-event-id-1> <rating-event-id-2>" >&2
  exit 64
fi

SOURCE_SHA="$1"
CANONICAL_ID="$2"
EVENT_ONE="$3"
EVENT_TWO="$4"

[[ "$SOURCE_SHA" =~ ^[0-9a-f]{40}$ ]] || { echo "invalid source SHA" >&2; exit 64; }
[[ "$CANONICAL_ID" =~ ^[0-9a-z]{20,40}$ ]] || { echo "invalid canonical entity id" >&2; exit 64; }
[[ "$EVENT_ONE" =~ ^[0-9a-z]{20,40}$ ]] || { echo "invalid first rating event id" >&2; exit 64; }
[[ "$EVENT_TWO" =~ ^[0-9a-z]{20,40}$ ]] || { echo "invalid second rating event id" >&2; exit 64; }

RUM_NS="rum-dev-isolated"
RUM_ORIGIN="https://dev-rum.daisycloversoftware.uk"
RAT_ORIGIN="https://dev-rum-ra.daisycloversoftware.uk"
PUBLIC_NS="rum-dev"
PUBLIC_HOST="rateurmate.online"
PLAYWRIGHT_IMAGE="mcr.microsoft.com/playwright/python:v1.57.0-noble"

for command in docker python3 curl mktemp; do
  command -v "$command" >/dev/null 2>&1 || { echo "required command unavailable: $command" >&2; exit 2; }
done

if command -v k3s >/dev/null 2>&1; then
  KUBECTL=(sudo -n k3s kubectl)
else
  KUBECTL=(kubectl)
fi
kube() { "${KUBECTL[@]}" "$@"; }

for ns in "$RUM_NS" rum-rate-anything-preview "$PUBLIC_NS"; do kube get namespace "$ns" >/dev/null; done
public_hosts="$(kube -n "$PUBLIC_NS" get ingress -o jsonpath='{range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)"
grep -Fxq "$PUBLIC_HOST" <<<"$public_hosts" || { echo "blocked: public host not found before verification" >&2; exit 78; }
public_images_before="$(kube -n "$PUBLIC_NS" get deployments -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{range .spec.template.spec.containers[*]}{.image}{";"}{end}{"\n"}{end}' | sort)"

rum_health="$(curl -fsS -H 'Cache-Control: no-cache' --max-time 20 "$RUM_ORIGIN/api/v1/health")"
python3 - "$rum_health" "$SOURCE_SHA" <<'PY'
import json,sys
obj=json.loads(sys.argv[1]); source=sys.argv[2]
assert obj.get('status') == 'ok', obj
assert obj.get('service') == 'rum-api', obj
assert source in str(obj.get('version') or ''), obj
PY
rat_version="$(curl -fsS -H 'Cache-Control: no-cache' --max-time 20 "$RAT_ORIGIN/VERSION" | tr -d '\r\n')"
grep -Fq "$SOURCE_SHA" <<<"$rat_version" || { echo "RAT frontend is not exact source head" >&2; exit 78; }

api_pod="$(kube -n "$RUM_NS" get pods -l 'app.kubernetes.io/instance=rum,app.kubernetes.io/component=api' --field-selector=status.phase=Running -o jsonpath='{.items[0].metadata.name}')"
[[ -n "$api_pod" ]] || { echo "isolated RUM API pod unavailable" >&2; exit 78; }

work="$(mktemp -d)"
cleanup() { rm -rf "$work"; }
trap cleanup EXIT HUP INT TERM
chmod 700 "$work"

suffix="$(date +%s)-$(python3 -c 'import secrets; print(secrets.token_hex(3))')"
username="rumcj${suffix//-/}"
username="${username:0:28}"
email="${username}@example.com"
password="$(python3 -c 'import secrets; print("RumCJ-" + secrets.token_urlsafe(20) + "!Aa9")')"
state="$work/rum-state.json"

cat >"$work/register.py" <<'PY'
import os
from playwright.sync_api import sync_playwright

base=os.environ['RUM_BASE_URL']
email=os.environ['RUM_TEST_EMAIL']
username=os.environ['RUM_TEST_USERNAME']
password=os.environ['RUM_TEST_PASSWORD']
state=os.environ['RUM_STATE_FILE']

with sync_playwright() as p:
    browser=p.chromium.launch()
    context=browser.new_context(viewport={"width":390,"height":844})
    page=context.new_page()
    page.goto(base, wait_until='networkidle', timeout=60000)
    page.get_by_role('button', name='Create an account').click()
    page.get_by_label('Email').fill(email)
    page.get_by_label('Username').fill(username)
    page.get_by_label('Password', exact=True).fill(password)
    page.get_by_label('Confirm password').fill(password)
    page.get_by_role('checkbox', name='I accept the Terms and Community Rules.').check()
    page.get_by_role('checkbox', name='I have read the Privacy Notice.').check()
    page.get_by_role('button', name='Create account').click()
    page.wait_for_url('**/me', timeout=30000)
    context.storage_state(path=state)
    browser.close()
print('RUM_DEV_REGISTRATION_SESSION_OK')
PY

docker run --rm --ipc=host \
  -v "$work:/work" \
  -e RUM_BASE_URL="$RUM_ORIGIN" \
  -e RUM_TEST_EMAIL="$email" \
  -e RUM_TEST_USERNAME="$username" \
  -e RUM_TEST_PASSWORD="$password" \
  -e RUM_STATE_FILE="/work/$(basename "$state")" \
  "$PLAYWRIGHT_IMAGE" python /work/register.py
[[ -s "$state" ]] || { echo "RUM browser state was not created" >&2; exit 1; }
chmod 600 "$state"

email_b64="$(printf '%s' "$email" | python3 -c 'import base64,sys; print(base64.b64encode(sys.stdin.buffer.read()).decode())')"
kube -n "$RUM_NS" exec "$api_pod" -c php-fpm -- php artisan tinker --execute="\
\$u=App\\Models\\User::where('email',base64_decode('${email_b64}'))->firstOrFail(); \
\$u->forceFill(['email_verified_at'=>now()])->save();" >/dev/null

cat >"$work/exercise.py" <<'PY'
import json, os
from urllib.parse import urlparse, parse_qs
from playwright.sync_api import sync_playwright

rum=os.environ['RUM_BASE_URL']
rat=os.environ['RAT_BASE_URL']
canonical_id=os.environ['CANONICAL_ID']
state='/work/rum-state.json'
identity_labels={'person','public person','persona','public persona','digital identity','gaming / online identity'}

def require(cond,msg):
    if not cond:
        raise AssertionError(msg)

with sync_playwright() as p:
    browser=p.chromium.launch()

    # RUM regression: authenticated mobile-first CJ search opens the existing
    # canonical RUM rating host for the exact same entity id.
    context=browser.new_context(storage_state=state, viewport={"width":390,"height":844})
    page=context.new_page()
    page_errors=[]
    page.on('pageerror', lambda err: page_errors.append(str(err)))
    page.goto(f'{rum}/rate', wait_until='networkidle', timeout=60000)
    search=page.get_by_role('searchbox', name='Search RUM members and public identities')
    search.wait_for(state='visible', timeout=30000)
    search.fill('CJ')
    card=page.locator('.person-card').filter(has_text='CJ Investigates').first
    card.wait_for(state='visible', timeout=30000)
    card.get_by_role('button', name='Rate', exact=True).click()
    panel=page.locator('.entity-bridge__public-rating')
    panel.wait_for(state='visible', timeout=30000)
    page.get_by_role('heading', name='Rate CJ Investigates').wait_for(state='visible', timeout=30000)
    page.get_by_text('1. Your RUM verdict', exact=True).wait_for(state='visible', timeout=30000)
    page.get_by_role('radio', name='Thumbs up 5 of 5 — Exceptional').wait_for(state='visible', timeout=30000)
    require(urlparse(page.url).netloc == urlparse(rum).netloc, f'RUM CJ rating left RUM host: {page.url}')
    overflow=page.evaluate('document.documentElement.scrollWidth - window.innerWidth')
    require(overflow <= 1, f'RUM mobile CJ rating overflows viewport by {overflow}px')
    current=context.request.get(f'{rum}/api/v1/rate-anything/rating?entityId={canonical_id}', max_redirects=0)
    require(current.status == 200, f'RUM canonical rating read returned {current.status}')
    payload=current.json()
    require(payload.get('meta',{}).get('thingId') == canonical_id, f'RUM canonical id mismatch: {payload.get("meta")}')
    require(payload.get('meta',{}).get('canonicalWorkflow') is True, 'RUM rating read is not canonical workflow')
    require(not page_errors, f'RUM browser runtime errors: {page_errors}')
    print('RUM_CJ_MOBILE_REGRESSION_PASS ' + json.dumps({'canonicalId':canonical_id,'overflow':overflow,'host':urlparse(page.url).netloc}))
    context.close()

    # Non-person RAT regression: enter a linked ordinary Thing from DJI, then
    # open its RAT-native Value step without relying on a CJ special case.
    context=browser.new_context(viewport={"width":1440,"height":1000})
    page=context.new_page()
    page_errors=[]
    page.on('pageerror', lambda err: page_errors.append(str(err)))
    page.goto(f'{rat}/search?q=DJI', wait_until='domcontentloaded', timeout=60000)
    page.locator('.rat-active-card h1').wait_for(state='visible', timeout=30000)
    require(page.locator('.rat-active-card h1').inner_text().strip() == 'DJI', 'DJI root did not resolve')
    cards=page.locator('.rat-linked-card')
    chosen=None
    for index in range(cards.count()):
        card=cards.nth(index)
        name=card.locator('strong').inner_text().strip()
        type_label=card.locator('small').inner_text().strip()
        if name and type_label.lower() not in identity_labels:
            chosen=(card,name,type_label)
            break
    require(chosen is not None, 'No ordinary non-person linked Thing was available from DJI')
    card,name,type_label=chosen
    card.click()
    page.locator('.rat-active-card h1').wait_for(state='visible', timeout=30000)
    require(page.locator('.rat-active-card h1').inner_text().strip() == name, f'Linked Thing did not become active: {name}')
    active_type=page.locator('.rat-active-card__type').inner_text().strip()
    require(active_type.lower() not in identity_labels, f'Chosen RAT regression target is an identity: {active_type}')
    parsed=urlparse(page.url); active=parse_qs(parsed.query).get('active',[None])[0]
    require(active, 'Non-person RAT active id is missing')
    page.get_by_role('button', name='Rate', exact=True).click()
    page.wait_for_url(lambda url: urlparse(url).netloc == urlparse(rat).netloc and urlparse(url).path == f'/rate/{active}', timeout=30000)
    page.locator('section[data-rating-step="Value"]').wait_for(state='visible', timeout=30000)
    page.locator('.rat-topbar').wait_for(state='visible', timeout=30000)
    overflow=page.evaluate('document.documentElement.scrollWidth - window.innerWidth')
    require(overflow <= 1, f'RAT non-person rating page overflows viewport by {overflow}px')
    require(not page_errors, f'RAT non-person runtime errors: {page_errors}')
    print('RAT_NON_PERSON_REGRESSION_PASS ' + json.dumps({'name':name,'type':active_type,'id':active,'overflow':overflow,'host':urlparse(page.url).netloc}))
    context.close()
    browser.close()
PY

docker run --rm --ipc=host \
  -v "$work:/work" \
  -e RUM_BASE_URL="$RUM_ORIGIN" \
  -e RAT_BASE_URL="$RAT_ORIGIN" \
  -e CANONICAL_ID="$CANONICAL_ID" \
  "$PLAYWRIGHT_IMAGE" python /work/exercise.py

# Post-deploy canonical-data proof. Inspect only the exact browser-created event
# ids and the CJ canonical-name cardinality; do not emit account identifiers.
proof="$(kube -n "$RUM_NS" exec "$api_pod" -c php-fpm -- php artisan tinker --execute="\
\$ids=['${EVENT_ONE}','${EVENT_TWO}']; \
\$events=App\\Models\\RatingEvent::with('targetEntity')->whereIn('id',\$ids)->get(); \
if (\$events->count() !== 2) { throw new RuntimeException('expected both browser rating events in canonical RUM store'); } \
foreach (\$events as \$event) { \
  if ((string)\$event->target_entity_id !== '${CANONICAL_ID}') { throw new RuntimeException('rating event target is not canonical CJ'); } \
  if ((string)optional(\$event->targetEntity)->canonical_name !== 'CJ Investigates') { throw new RuntimeException('rating event target name mismatch'); } \
  \$active=App\\Models\\RatingEvent::where('rater_account_id',\$event->rater_account_id)->where('target_entity_id','${CANONICAL_ID}')->where('status','active')->count(); \
  if (\$active !== 1) { throw new RuntimeException('duplicate active rating relationship detected'); } \
} \
\$cj=App\\Models\\Entity::where('canonical_name','CJ Investigates')->count(); \
if (\$cj !== 1) { throw new RuntimeException('duplicate CJ canonical entity detected'); } \
echo 'CANONICAL_RATING_DATA_PASS events='.\$events->count().' cj_entities='.\$cj.' target=${CANONICAL_ID}';")"
grep -Fq "CANONICAL_RATING_DATA_PASS events=2 cj_entities=1 target=$CANONICAL_ID" <<<"$proof" || { echo "canonical rating data proof failed" >&2; printf '%s\n' "$proof" >&2; exit 1; }
printf '%s\n' "$(grep -F 'CANONICAL_RATING_DATA_PASS' <<<"$proof" | tail -n1)"

[[ "$(kube -n "$PUBLIC_NS" get ingress -o jsonpath='{range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)" == "$public_hosts" ]]
[[ "$(kube -n "$PUBLIC_NS" get deployments -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{range .spec.template.spec.containers[*]}{.image}{";"}{end}{"\n"}{end}' | sort)" == "$public_images_before" ]]
printf 'PUBLIC_RUM_UNCHANGED=true\n'
printf 'RUM160_CROSS_SURFACE_REGRESSIONS_PASS=true\n'
