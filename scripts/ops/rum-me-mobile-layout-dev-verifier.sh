#!/usr/bin/env bash
set -euo pipefail
umask 077

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <exact-rum-candidate-sha>" >&2
  exit 64
fi

CANDIDATE_SHA="$1"
[[ "$CANDIDATE_SHA" =~ ^[0-9a-f]{40}$ ]] || { echo "full lowercase candidate SHA required" >&2; exit 64; }

REPOSITORY="DaisyCloverSoftware/rum"
BRANCH="sprint-0-rum-owner-rating-flow-20260823"
PR=153
DEV_NAMESPACE="rum-dev-isolated"
LIVE_NAMESPACE="rum-dev"
DEV_HOST="dev-rum.daisycloversoftware.uk"
LIVE_HOST="rateurmate.online"
BASE_URL="https://${DEV_HOST}"
EXPECTED_TAG="sha-${CANDIDATE_SHA:0:8}"
LIVE_BASELINE_TAG="sha-8106675"
PLAYWRIGHT_IMAGE="mcr.microsoft.com/playwright/python:v1.57.0-noble"
LIVE_BASELINE_CSP_HASH="sha256-A0FJyCgxFUPhG7nac5LcQPwVRK5So9ZNz7x5ubsD9kU="

for command in gh python3 mktemp sha256sum; do
  command -v "$command" >/dev/null 2>&1 || { echo "VERIFY BLOCKED: missing $command" >&2; exit 2; }
done
TOKEN="${GH_TOKEN:-}"
[[ -n "$TOKEN" ]] || TOKEN="$(gh auth token 2>/dev/null || true)"
[[ -n "$TOKEN" ]] || { echo "VERIFY BLOCKED: GitHub token unavailable" >&2; exit 2; }

head="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/${BRANCH}" --jq '.object.sha')"
[[ "$head" == "$CANDIDATE_SHA" ]] || { echo "VERIFY BLOCKED: candidate branch moved" >&2; exit 78; }
pr_state="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/pulls/${PR}" --jq '[.state, (.draft|tostring), .head.sha, (.merged_at // "")] | @tsv')"
IFS=$'\t' read -r state draft pr_head merged_at <<<"$pr_state"
[[ "$state" == open && "$draft" == true && "$pr_head" == "$CANDIDATE_SHA" && -z "$merged_at" ]] || { echo "VERIFY BLOCKED: PR #153 state mismatch" >&2; exit 78; }
ci_successes="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/actions/runs?head_sha=${CANDIDATE_SHA}&event=pull_request&status=completed&per_page=100" --jq '[.workflow_runs[] | select(.name == "CI" and .head_sha == "'"${CANDIDATE_SHA}"'" and .conclusion == "success")] | length')"
[[ "$ci_successes" =~ ^[0-9]+$ && "$ci_successes" -ge 1 ]] || { echo "VERIFY BLOCKED: exact-head CI success missing" >&2; exit 78; }

kctl(){ if command -v k3s >/dev/null 2>&1; then sudo k3s kubectl "$@"; else kubectl "$@"; fi; }
[[ "$(kctl -n "$DEV_NAMESPACE" get ingress rum -o jsonpath='{range .spec.rules[*]}{.host}{"\n"}{end}' 2>/dev/null || true)" == "$DEV_HOST" ]] || { echo "VERIFY BLOCKED: isolated DEV ingress mismatch" >&2; exit 78; }
[[ "$(kctl -n "$LIVE_NAMESPACE" get ingress rum -o jsonpath='{range .spec.rules[*]}{.host}{"\n"}{end}' 2>/dev/null || true)" == "$LIVE_HOST" ]] || { echo "VERIFY BLOCKED: LIVE ingress mismatch" >&2; exit 78; }
for component in web api worker; do
  image="$(kctl -n "$DEV_NAMESPACE" get deploy "rum-${component}" -o jsonpath='{.spec.template.spec.containers[0].image}')"
  [[ "$image" == *":${EXPECTED_TAG}" ]] || { echo "VERIFY BLOCKED: isolated rum-${component} is not ${EXPECTED_TAG}: ${image}" >&2; exit 78; }
  live_image="$(kctl -n "$LIVE_NAMESPACE" get deploy "rum-${component}" -o jsonpath='{.spec.template.spec.containers[0].image}')"
  [[ "$live_image" == *":${LIVE_BASELINE_TAG}" ]] || { echo "VERIFY BLOCKED: LIVE rum-${component} moved from ${LIVE_BASELINE_TAG}: ${live_image}" >&2; exit 78; }
done

api_pod="$(kctl -n "$DEV_NAMESPACE" get pods -l 'app.kubernetes.io/instance=rum,app.kubernetes.io/component=api' --field-selector=status.phase=Running -o jsonpath='{.items[0].metadata.name}')"
[[ -n "$api_pod" ]] || { echo "VERIFY BLOCKED: isolated DEV API pod missing" >&2; exit 78; }

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT HUP INT TERM
PHP_BOOT='require "/var/www/html/vendor/autoload.php"; $app=require "/var/www/html/bootstrap/app.php"; $app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap(); eval($argv[1]);'
owner_line="$(kctl -n "$DEV_NAMESPACE" exec "$api_pod" -c php-fpm -- php -r "$PHP_BOOT" '
$u=App\Models\User::query()->where("founder_number",2)->with("profile")->firstOrFail(); $p=$u->profile; echo $u->id.chr(9).base64_encode((string)($p?->display_name ?: $u->username));' 2>/dev/null | tail -n 1)"
IFS=$'\t' read -r owner_id owner_name_b64 <<<"$owner_line"
[[ "$owner_id" =~ ^[0-9a-z]{26}$ ]] || { echo "VERIFY BLOCKED: Founder #2 probe failed" >&2; exit 78; }
owner_name="$(printf '%s' "$owner_name_b64" | python3 -c 'import base64,sys; print(base64.b64decode(sys.stdin.read()).decode())')"
[[ -n "$owner_name" ]] || { echo "VERIFY BLOCKED: Founder #2 display name empty" >&2; exit 78; }

session_line="$(kctl -n "$DEV_NAMESPACE" exec "$api_pod" -c php-fpm -- php -r "$PHP_BOOT" '
$u=App\Models\User::query()->where("founder_number",2)->firstOrFail(); $s=app("session")->driver(); $s->setId(Illuminate\Support\Str::random(40)); $s->start(); $s->put(Illuminate\Support\Facades\Auth::guard("web")->getName(),$u->getAuthIdentifier()); $s->save(); $name=(string)config("session.cookie"); $prefix=Illuminate\Cookie\CookieValuePrefix::create($name,app("encrypter")->getKey()); $value=app("encrypter")->encrypt($prefix.$s->getId(),false); echo base64_encode($name).chr(9).base64_encode($value);' 2>/dev/null | tail -n 1)"
IFS=$'\t' read -r cookie_name_b64 cookie_value_b64 <<<"$session_line"
cookie_name="$(printf '%s' "$cookie_name_b64" | python3 -c 'import base64,sys; print(base64.b64decode(sys.stdin.read()).decode())')"
cookie_value="$(printf '%s' "$cookie_value_b64" | python3 -c 'import base64,sys; print(base64.b64decode(sys.stdin.read()).decode())')"
[[ -n "$cookie_name" && -n "$cookie_value" ]] || { echo "VERIFY BLOCKED: isolated owner session creation failed" >&2; exit 78; }
python3 - "$work/owner-state.json" "$cookie_name" "$cookie_value" "$DEV_HOST" <<'PY'
import json,sys
path,name,value,domain=sys.argv[1:]
json.dump({'cookies':[{'name':name,'value':value,'domain':domain,'path':'/','expires':-1,'httpOnly':True,'secure':True,'sameSite':'Lax'}],'origins':[]},open(path,'w'))
PY
chmod 600 "$work/owner-state.json"

cat >"$work/verify.py" <<'PY'
import hashlib,json,os
from playwright.sync_api import sync_playwright

base=os.environ['RUM_BASE_URL']
owner_name=os.environ['RUM_OWNER_NAME']
baseline=os.environ['RUM_BASELINE_CSP']
console_errors=[]; page_errors=[]; request_failures=[]; server_errors=[]

def watch(page):
    page.on('console', lambda msg: console_errors.append(msg.text) if msg.type == 'error' else None)
    page.on('pageerror', lambda err: page_errors.append(str(err)))
    page.on('requestfailed', lambda req: request_failures.append(f'{req.method} {req.url}'))
    page.on('response', lambda res: server_errors.append(f'{res.status} {res.url}') if res.status >= 500 else None)

def require(cond,msg):
    if not cond: raise RuntimeError(msg)

def inside(parent, child, pad=2):
    return (child['x'] >= parent['x']-pad and child['y'] >= parent['y']-pad and
            child['x']+child['width'] <= parent['x']+parent['width']+pad and
            child['y']+child['height'] <= parent['y']+parent['height']+pad)

with sync_playwright() as p:
    browser=p.chromium.launch()
    context=browser.new_context(storage_state='/work/owner-state.json', viewport={'width':390,'height':844}, device_scale_factor=1)
    page=context.new_page(); watch(page)
    page.goto(f'{base}/me', wait_until='networkidle', timeout=60000)
    page.get_by_role('heading', name=owner_name, exact=True).wait_for(state='visible', timeout=30000)

    hero=page.locator('.screen-content > .profile-hero').first; hero.wait_for(state='visible', timeout=15000)
    identity=hero.locator(':scope > .profile-hero__identity').first
    status=hero.locator(':scope > .profile-hero__status').first
    stats=hero.locator(':scope > .profile-hero__stats').first
    actions=hero.locator(':scope > .profile-hero__actions').first
    for node,name in ((identity,'identity'),(status,'status'),(stats,'stats'),(actions,'actions')):
        node.wait_for(state='visible', timeout=15000)

    hero_style=hero.evaluate("""(el) => { const s=getComputedStyle(el); return {display:s.display, gridTemplateColumns:s.gridTemplateColumns, overflowX:s.overflowX}; }""")
    require(hero_style['display'] == 'block', f'Me profile card was hijacked by dedicated profile grid styles: {hero_style}')

    hb=hero.bounding_box(); ib=identity.bounding_box(); sb=status.bounding_box(); stb=stats.bounding_box(); ab=actions.bounding_box()
    require(all((hb,ib,sb,stb,ab)), f'Me profile card bounds unavailable: hero={hb} identity={ib} status={sb} stats={stb} actions={ab}')
    for box,name in ((ib,'identity'),(sb,'status'),(stb,'stats'),(ab,'actions')):
        require(inside(hb,box), f'Me {name} content is clipped outside profile card: hero={hb} {name}={box}')

    require(sb['y'] >= ib['y'] + ib['height'] - 2, f'Me status is beside/crushed into identity row instead of below it: identity={ib} status={sb}')
    require(stb['y'] >= sb['y'] + sb['height'] - 2, f'Me stats are not below status: status={sb} stats={stb}')
    require(ab['y'] >= stb['y'] + stb['height'] - 2, f'Me actions are not below stats: stats={stb} actions={ab}')
    require(sb['width'] >= hb['width'] * 0.70, f'Me status collapsed into a narrow column: hero={hb} status={sb}')
    require(sb['height'] <= 120, f'Me status wrapped vertically like the rejected screenshot: {sb}')

    buttons=actions.locator('button')
    require(buttons.count()==3, f'Me action count changed unexpectedly: {buttons.count()}')
    for i in range(buttons.count()):
        b=buttons.nth(i); require(b.is_visible(), f'Me action {i} is hidden')
        bb=b.bounding_box(); require(bb and inside(hb,bb), f'Me action {i} is clipped outside profile card: hero={hb} button={bb}')

    overflow=page.evaluate("""() => ({inner:window.innerWidth, doc:document.documentElement.scrollWidth, body:document.body.scrollWidth})""")
    require(overflow['doc'] <= overflow['inner']+1 and overflow['body'] <= overflow['inner']+1, f'Me page has horizontal overflow: {overflow}')

    page.screenshot(path='/work/rum-me-mobile-390.png', full_page=True)
    digest=hashlib.sha256(open('/work/rum-me-mobile-390.png','rb').read()).hexdigest()
    print('me_mobile_viewport=390x844')
    print('me_profile_card_display=block')
    print('me_profile_vertical_order=identity|status|stats|actions')
    print(f'me_profile_status_width={sb["width"]:.1f}')
    print(f'me_mobile_overflow=document:{overflow["doc"]}/{overflow["inner"]} body:{overflow["body"]}/{overflow["inner"]}')
    print(f'me_mobile_screenshot_sha256={digest}')

    known=[msg for msg in console_errors if baseline in msg and "script-src 'self'" in msg]
    unexpected=[msg for msg in console_errors if msg not in known]
    print(f'me_known_live_baseline_csp_console_errors={len(known)}')
    require(not unexpected, 'unexpected Me console errors: '+' | '.join(unexpected[:5]))
    require(not page_errors, 'Me page errors: '+' | '.join(page_errors[:5]))
    require(not request_failures, 'Me request failures: '+' | '.join(request_failures[:5]))
    require(not server_errors, 'Me server errors: '+' | '.join(server_errors[:5]))
    print('me_unexpected_console_errors=0')
    print('me_network_errors=0')
    context.close(); browser.close()
print('RUM_ME_MOBILE_LAYOUT_VERIFIED=YES')
PY

if command -v docker >/dev/null 2>&1; then runtime=docker
elif command -v podman >/dev/null 2>&1; then runtime=podman
else echo "VERIFY BLOCKED: no container runtime" >&2; exit 78; fi

if [[ "$runtime" == podman ]]; then
  podman pull "$PLAYWRIGHT_IMAGE" >/dev/null
  image="localhost/rum-me-layout-playwright:1.57.0"
  ctx="$work/pw"; mkdir -p "$ctx"
  printf 'FROM %s\nRUN python -m pip install --no-cache-dir playwright==1.57.0\n' "$PLAYWRIGHT_IMAGE" >"$ctx/Containerfile"
  podman build --pull=missing -t "$image" "$ctx" >/dev/null
else
  image="$PLAYWRIGHT_IMAGE"
fi

unset TOKEN GH_TOKEN GHCR_TOKEN
"$runtime" run --rm --ipc=host -v "$work:/work" -e RUM_BASE_URL="$BASE_URL" -e RUM_OWNER_NAME="$owner_name" -e RUM_BASELINE_CSP="$LIVE_BASELINE_CSP_HASH" "$image" python /work/verify.py

for component in web api worker; do
  live_after="$(kctl -n "$LIVE_NAMESPACE" get deploy "rum-${component}" -o jsonpath='{.spec.template.spec.containers[0].image}')"
  [[ "$live_after" == *":${LIVE_BASELINE_TAG}" ]] || { echo "VERIFY FAILED: LIVE rum-${component} changed during verification: ${live_after}" >&2; exit 1; }
done
printf 'LIVE_RUNTIME_AFFECTED=NO\nRATE_ANYTHING_AFFECTED=NO\n'
