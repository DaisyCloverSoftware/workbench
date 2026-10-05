#!/usr/bin/env bash
set -euo pipefail
umask 077

SOURCE_SHA="${1:-}"
[[ "$SOURCE_SHA" =~ ^[0-9a-f]{40}$ ]] || { echo "usage: $0 <40-char-source-sha>" >&2; exit 64; }

RUM_NS="rum-dev-isolated"
RAT_NS="rum-rate-anything-preview"
RUM_HOST="dev-rum.daisycloversoftware.uk"
RAT_HOST="dev-rum-ra.daisycloversoftware.uk"
RAT_BASE_URL="https://${RAT_HOST}"
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

rum_api_pod="$(kctl -n "$RUM_NS" get pods \
  -l 'app.kubernetes.io/instance=rum,app.kubernetes.io/component=api' \
  --field-selector=status.phase=Running \
  -o jsonpath='{.items[0].metadata.name}')"
[[ -n "$rum_api_pod" ]] || { echo "canonical isolated RUM API pod unavailable" >&2; exit 78; }

candidate_raw="$(kctl -n "$RUM_NS" exec "$rum_api_pod" -c php-fpm -- php artisan tinker --execute="\
\$e=App\\Models\\Entity::query()->with('type')->where('visibility','public')->where('publication_state','published')->where('status','active')->where('rateability','open')->whereHas('type',fn(\$q)=>\$q->whereNotIn('key',['person','persona','digital_identity']))->orderBy('canonical_name')->first(); \
echo json_encode(['id'=>\$e?->id,'name'=>\$e?->canonical_name,'type'=>\$e?->type?->key]);")"

candidate_json="$(python3 - "$candidate_raw" <<'PY'
import json,re,sys
matches=re.findall(r'\{[^\n]*\}', sys.argv[1])
if not matches:
    raise SystemExit('no canonical non-person entity JSON found')
obj=json.loads(matches[-1])
assert obj.get('id') and obj.get('name') and obj.get('type'), obj
assert obj['type'] not in {'person','persona','digital_identity'}, obj
print(json.dumps(obj,separators=(',',':')))
PY
)"

ENTITY_ID="$(python3 -c 'import json,sys; print(json.loads(sys.argv[1])["id"])' "$candidate_json")"
ENTITY_NAME="$(python3 -c 'import json,sys; print(json.loads(sys.argv[1])["name"])' "$candidate_json")"
ENTITY_TYPE="$(python3 -c 'import json,sys; print(json.loads(sys.argv[1])["type"])' "$candidate_json")"

work="$(mktemp -d)"
cleanup() { rm -rf "$work"; }
trap cleanup EXIT HUP INT TERM
chmod 700 "$work"

cat >"$work/verify.py" <<'PY'
import json,os,time,random
from playwright.sync_api import sync_playwright
rat=os.environ['RAT_BASE_URL']
entity_id=os.environ['ENTITY_ID']
entity_name=os.environ['ENTITY_NAME']
entity_type=os.environ['ENTITY_TYPE']
source=os.environ['SOURCE_SHA']
unique=f"{int(time.time())}{random.randint(1000,9999)}"
username=("ratthing"+unique)[:28]
email=f"{username}@example.com"
password=f"RAT-{unique}-Thing!Aa9"
result={}
with sync_playwright() as p:
    browser=p.chromium.launch()
    context=browser.new_context(viewport={'width':900,'height':1000})
    page=context.new_page()
    errors=[]
    page.on('pageerror', lambda error: errors.append(str(error)))
    page.goto(rat + '/search?q=' + entity_name, wait_until='domcontentloaded', timeout=60000)
    heading=page.locator('.rat-active-card h1')
    heading.wait_for(state='visible', timeout=30000)
    assert heading.inner_text().strip() == entity_name, (entity_name, heading.inner_text())
    active=page.url.split('active=',1)[1].split('&',1)[0] if 'active=' in page.url else None
    from urllib.parse import unquote
    assert unquote(active or '') == entity_id, (entity_id, page.url)
    page.get_by_role('button', name='Rate', exact=True).click()
    page.wait_for_url(lambda url: url.startswith(rat + '/rate/' + entity_id), timeout=30000)
    page.locator('section[data-rating-step="Value"]').wait_for(state='visible', timeout=30000)
    assert page.url.startswith(rat), page.url
    page.get_by_role('radio', name='5: Exceptional. Strongest approval').click()
    page.get_by_role('button', name='Continue', exact=True).click()
    auth=page.locator('#rat-rating-auth')
    auth.wait_for(state='visible', timeout=10000)
    auth.get_by_role('button', name='Create account', exact=True).click()
    auth.get_by_label('Email').fill(email)
    auth.get_by_label('Username').fill(username)
    auth.get_by_label('Password', exact=True).fill(password)
    auth.get_by_label('Confirm password').fill(password)
    auth.get_by_role('checkbox', name='I accept the Terms and Community Rules.').check()
    auth.get_by_role('checkbox', name='I have read the Privacy Notice.').check()
    with page.expect_response(lambda r: r.url == rat + '/api/v1/rate-anything/auth/register' and r.request.method == 'POST') as reg:
        auth.get_by_role('button', name='Create RAT account and continue', exact=True).click()
    assert reg.value.status == 201, reg.value.status
    page.locator('#rat-rating-auth').wait_for(state='detached', timeout=30000)
    page.get_by_role('button', name='Continue', exact=True).click()
    page.locator('section[data-rating-step="Category"]').wait_for(state='visible', timeout=30000)
    category=page.locator('.rat-rating-choice-grid button').first
    category.wait_for(state='visible', timeout=30000)
    category.click()
    page.get_by_role('button', name='Continue', exact=True).click()
    page.locator('section[data-rating-step="Reason"]').wait_for(state='visible', timeout=30000)
    page.locator('section[data-rating-step="Reason"] textarea').fill(f'Ordinary Thing regression for {entity_name}.')
    page.get_by_role('button', name='Continue', exact=True).click()
    page.locator('section[data-rating-step="Review"]').wait_for(state='visible', timeout=30000)
    with page.expect_response(lambda r: r.url == rat + '/api/v1/rate-anything/rating' and r.request.method == 'POST') as submitted:
        page.get_by_role('button', name='Submit rating', exact=True).click()
    assert submitted.value.status in (200,201), submitted.value.status
    payload=submitted.value.json()
    assert payload['meta']['thingId'] == entity_id, payload
    assert payload['meta']['canonicalWorkflow'] is True, payload
    page.locator('section[data-rating-state="complete"]').wait_for(state='visible', timeout=30000)
    assert page.url.startswith(rat), page.url
    assert not errors, errors
    result={
      'result':'PASS','sourceSha':source,'host':rat,'entityId':entity_id,
      'entityName':entity_name,'entityType':entity_type,'ratingEventId':payload['data']['id'],
      'viewport':{'width':900,'height':1000}
    }
    browser.close()
print(json.dumps(result,separators=(',',':')))
PY

verification="$(docker run --rm --ipc=host \
  -v "$work:/work" \
  -e RAT_BASE_URL="$RAT_BASE_URL" \
  -e ENTITY_ID="$ENTITY_ID" \
  -e ENTITY_NAME="$ENTITY_NAME" \
  -e ENTITY_TYPE="$ENTITY_TYPE" \
  -e SOURCE_SHA="$SOURCE_SHA" \
  "$PLAYWRIGHT_IMAGE" python /work/verify.py)"

printf 'RUM160_RAT_NONPERSON_TARGET=%s\n' "$candidate_json"
printf 'RUM160_RAT_NONPERSON_BROWSER=%s\n' "$verification"
printf 'RUM160_RAT_NONPERSON_PASS=true\n'
