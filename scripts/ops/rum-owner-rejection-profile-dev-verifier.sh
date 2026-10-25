#!/usr/bin/env bash
set -euo pipefail
umask 077

if [[ $# -ne 1 ]]; then echo "usage: $0 <exact-rum-candidate-sha>" >&2; exit 64; fi
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
PLAYWRIGHT_IMAGE="mcr.microsoft.com/playwright/python:v1.57.0-noble"

for command in gh python3 mktemp sha256sum; do command -v "$command" >/dev/null 2>&1 || { echo "VERIFY BLOCKED: missing $command" >&2; exit 2; }; done
TOKEN="${GH_TOKEN:-}"; [[ -n "$TOKEN" ]] || TOKEN="$(gh auth token 2>/dev/null || true)"; [[ -n "$TOKEN" ]] || { echo "VERIFY BLOCKED: GitHub token unavailable" >&2; exit 2; }
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
  [[ "$image" == *":${EXPECTED_TAG}" ]] || { echo "VERIFY BLOCKED: rum-${component} is not ${EXPECTED_TAG}: ${image}" >&2; exit 78; }
  printf 'ISOLATED_%s_IMAGE=%s\n' "${component^^}" "$image"
done
live_web_before="$(kctl -n "$LIVE_NAMESPACE" get deploy rum-web -o jsonpath='{.spec.template.spec.containers[0].image}')"
live_api_before="$(kctl -n "$LIVE_NAMESPACE" get deploy rum-api -o jsonpath='{.spec.template.spec.containers[0].image}')"
live_worker_before="$(kctl -n "$LIVE_NAMESPACE" get deploy rum-worker -o jsonpath='{.spec.template.spec.containers[0].image}')"
api_pod="$(kctl -n "$DEV_NAMESPACE" get pods -l 'app.kubernetes.io/instance=rum,app.kubernetes.io/component=api' --field-selector=status.phase=Running -o jsonpath='{.items[0].metadata.name}')"
[[ -n "$api_pod" ]] || { echo "VERIFY BLOCKED: isolated DEV API pod missing" >&2; exit 78; }

work="$(mktemp -d)"; trap 'rm -rf "$work"' EXIT HUP INT TERM
PHP_BOOT='require "/var/www/html/vendor/autoload.php"; $app=require "/var/www/html/bootstrap/app.php"; $app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap(); eval($argv[1]);'
owner_line="$(kctl -n "$DEV_NAMESPACE" exec "$api_pod" -c php-fpm -- php -r "$PHP_BOOT" '
$u=App\Models\User::query()->where("founder_number",2)->with("profile")->firstOrFail();
$p=$u->profile; if(!$p || !$p->avatar_media_id){fwrite(STDERR,"Founder #2 has no stored avatar media\n"); exit(3);} $m=App\Models\MediaAsset::query()->findOrFail($p->avatar_media_id);
$d=Storage::disk($m->storage_disk); if(!$m->thumbnail_key || !$d->exists($m->thumbnail_key)){fwrite(STDERR,"Founder #2 stored avatar thumbnail unavailable in isolated DEV\n"); exit(4);} $b=$d->get($m->thumbnail_key); if(strlen($b)<100){fwrite(STDERR,"Founder #2 avatar thumbnail is unexpectedly small\n"); exit(5);} echo $u->id.chr(9).base64_encode($p->display_name).chr(9).$p->avatar_media_id.chr(9).strlen($b).chr(9).hash("sha256",$b);' 2>/dev/null | tail -n 1)"
IFS=$'\t' read -r owner_id owner_name_b64 avatar_media_id avatar_bytes avatar_hash <<<"$owner_line"
[[ "$owner_id" =~ ^[0-9a-z]{26}$ && "$avatar_media_id" =~ ^[0-9a-z]{26}$ && "$avatar_bytes" =~ ^[0-9]+$ && "$avatar_hash" =~ ^[0-9a-f]{64}$ ]] || { echo "VERIFY BLOCKED: Founder #2 media probe failed" >&2; exit 78; }
owner_name="$(printf '%s' "$owner_name_b64" | python3 -c 'import base64,sys; print(base64.b64decode(sys.stdin.read()).decode())')"
[[ -n "$owner_name" ]] || { echo "VERIFY BLOCKED: Founder #2 display name empty" >&2; exit 78; }
printf 'OWNER_PROFILE_USER_ID=%s\nOWNER_PROFILE_AVATAR_MEDIA_ID=%s\nOWNER_PROFILE_AVATAR_BYTES=%s\nOWNER_PROFILE_AVATAR_SHA256=%s\n' "$owner_id" "$avatar_media_id" "$avatar_bytes" "$avatar_hash"

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
import os,hashlib,json
from playwright.sync_api import sync_playwright
base=os.environ['RUM_BASE_URL']; owner_name=os.environ['RUM_OWNER_NAME']; avatar_media_id=os.environ['RUM_AVATAR_MEDIA_ID']
console_errors=[]; page_errors=[]; request_failures=[]; server_errors=[]

def watch(page):
    page.on('console', lambda msg: console_errors.append(msg.text) if msg.type == 'error' else None)
    page.on('pageerror', lambda err: page_errors.append(str(err)))
    page.on('requestfailed', lambda req: request_failures.append(f'{req.method} {req.url}'))
    page.on('response', lambda res: server_errors.append(f'{res.status} {res.url}') if res.status >= 500 else None)

def require(cond,msg):
    if not cond: raise RuntimeError(msg)

def profile_assertions(page, mobile=False):
    page.get_by_role('heading', name=owner_name, exact=True).wait_for(state='visible', timeout=30000)
    hero=page.locator('.profile-hero').first; hero.wait_for(state='visible', timeout=15000)
    avatar=page.locator('.profile-hero__avatar .avatar').first
    image=avatar.locator('img').first; image.wait_for(state='visible', timeout=15000)
    image_state=image.evaluate("""(img) => ({src:img.src, complete:img.complete, naturalWidth:img.naturalWidth, naturalHeight:img.naturalHeight, objectFit:getComputedStyle(img).objectFit, width:img.getBoundingClientRect().width, height:img.getBoundingClientRect().height, parentRadius:getComputedStyle(img.parentElement).borderRadius, parentOverflow:getComputedStyle(img.parentElement).overflow})""")
    require(f'/api/v1/media/{avatar_media_id}/content?variant=thumbnail' in image_state['src'], f'profile image src is not stored avatar: {image_state}')
    require(image_state['complete'] and image_state['naturalWidth'] > 0 and image_state['naturalHeight'] > 0, f'profile image failed to load: {image_state}')
    require(image_state['objectFit'] == 'cover', f'profile image is not object-fit cover: {image_state}')
    require(image_state['parentRadius'] in ('50%', '9999px') or image_state['parentRadius'].startswith('50%'), f'avatar is not circular: {image_state}')
    require(image_state['parentOverflow'] == 'hidden', f'avatar does not clip image: {image_state}')

    mini=page.locator('.profile-hero__founder').first; mini.wait_for(state='visible', timeout=15000)
    canonical=mini.locator('.founder-alpha-approved').first; canonical.wait_for(state='visible', timeout=15000)
    require(canonical.locator('text=FOUNDER').count() >= 1 and canonical.locator('text=ALPHA').count() >= 1, 'mini Founder is not canonical Founder Alpha artwork')
    a=avatar.bounding_box(); b=mini.bounding_box(); require(a and b, 'avatar/founder bounds unavailable')
    ar=a['x']+a['width']; ab=a['y']+a['height']; br=b['x']+b['width']; bb=b['y']+b['height']
    overlap_x=max(0,min(ar,br)-max(a['x'],b['x'])); overlap_y=max(0,min(ab,bb)-max(a['y'],b['y']))
    overlap_ratio=(overlap_x*overlap_y)/(b['width']*b['height'])
    ratio=b['width']/a['width']
    require(b['x']+b['width']/2 > a['x']+a['width']/2 and b['y']+b['height']/2 > a['y']+a['height']/2, f'Founder badge is not in lower-right quadrant: avatar={a} badge={b}')
    require(b['x'] < ar and b['y'] < ab and br > ar and bb > ab, f'Founder badge does not overlap/project from lower-right edge: avatar={a} badge={b}')
    require(.35 <= overlap_ratio <= .65, f'Founder overlap footprint not approximately half: {overlap_ratio:.3f}')
    require(.42 <= ratio <= .68, f'Founder/avatar scale relationship unexpected: {ratio:.3f}')
    print(f"{'mobile' if mobile else 'desktop'}_avatar_image=loaded stored_media object_fit_cover circular_clip")
    print(f"{'mobile' if mobile else 'desktop'}_founder_geometry=lower_right overlap_ratio:{overlap_ratio:.3f} size_ratio:{ratio:.3f} avatar:{json.dumps(a,separators=(',',':'))} badge:{json.dumps(b,separators=(',',':'))}")

    rating=page.locator('.profile-rating-overview').first
    require(page.locator('.profile-rating-overview').count()==1, 'Rating Snapshot is not one outer feature card')
    rating.wait_for(state='visible', timeout=15000)
    require(rating.locator('.profile-section-heading').count()==1, 'Rating Snapshot current-view heading missing')
    selector=page.locator('.profile-rating-overview > .segmented').first
    metrics=rating.locator('.profile-metrics-row').first; filtered=rating.locator('.profile-filtered-rating').first; category=rating.locator('.profile-category-filter').first
    for item,name in ((selector,'selector'),(metrics,'metrics'),(filtered,'filtered result'),(category,'category controls')):
        item.wait_for(state='visible', timeout=15000)
    require(rating.locator('.profile-summary-card').count()==0, 'Rejected nested summary cards remain inside Rating Snapshot')
    metric_cells=metrics.locator('.profile-snapshot-metric')
    require(metric_cells.count()==3, f'expected three principal rating metrics, got {metric_cells.count()}')
    labels=[metric_cells.nth(i).locator(':scope > span').inner_text().strip() for i in range(3)]
    require(labels==['Overall Public Rating','Rate My Rating','Mates Only Rating'], f'unexpected principal metric labels: {labels}')
    boxes=[metric_cells.nth(i).bounding_box() for i in range(3)]; require(all(boxes), 'metric bounds unavailable')
    tops=[round(x['y'],1) for x in boxes]; require(max(tops)-min(tops) <= 2.0, f'three metrics are not in one row: {tops}')
    selector_buttons=selector.locator('button'); require(selector_buttons.count()>=1, 'profile view selector has no choices')
    for i in range(selector_buttons.count()):
        require(selector_buttons.nth(i).is_visible(), f'profile view selector choice {i} is hidden')
    print(f"{'mobile' if mobile else 'desktop'}_rating_snapshot=one_outer_card metrics_same_row labels:{'|'.join(labels)} selector_choices:{selector_buttons.count()}")

    overflow=page.evaluate("""() => {
      const doc=document.documentElement;
      const rating=document.querySelector('.profile-rating-overview');
      const selector=document.querySelector('.profile-rating-overview > .segmented');
      const offenders=[...rating.querySelectorAll('*')].filter(el => {
        const s=getComputedStyle(el); return (s.overflowX==='auto'||s.overflowX==='scroll') && el.scrollWidth > el.clientWidth + 1;
      }).map(el => ({tag:el.tagName, cls:el.className, scrollWidth:el.scrollWidth, clientWidth:el.clientWidth, overflowX:getComputedStyle(el).overflowX}));
      return {docScroll:doc.scrollWidth, docClient:doc.clientWidth, ratingScroll:rating.scrollWidth, ratingClient:rating.clientWidth, selectorScroll:selector.scrollWidth, selectorClient:selector.clientWidth, offenders};
    }""")
    require(overflow['docScroll'] <= overflow['docClient']+1, f'page horizontal overflow: {overflow}')
    require(overflow['ratingScroll'] <= overflow['ratingClient']+1, f'Rating Snapshot horizontal overflow: {overflow}')
    require(overflow['selectorScroll'] <= overflow['selectorClient']+1, f'profile selector horizontal overflow: {overflow}')
    require(not overflow['offenders'], f'nested horizontal scrollbar exists: {overflow["offenders"]}')
    print(f"{'mobile' if mobile else 'desktop'}_overflow=document:{overflow['docScroll']}/{overflow['docClient']} rating:{overflow['ratingScroll']}/{overflow['ratingClient']} selector:{overflow['selectorScroll']}/{overflow['selectorClient']} nested_scrollbars:0")

with sync_playwright() as p:
    browser=p.chromium.launch()
    desktop_context=browser.new_context(storage_state='/work/owner-state.json', viewport={'width':1280,'height':1000})
    desktop=desktop_context.new_page(); watch(desktop); desktop.goto(f'{base}/profile', wait_until='networkidle', timeout=60000)
    profile_assertions(desktop, False)
    desktop.get_by_role('button', name='Badges').click()
    full=desktop.locator('.profile-badge-mark--founder-canonical .founder-alpha-approved').first; full.wait_for(state='visible', timeout=15000)
    require(full.locator('text=FOUNDER').count()>=1 and full.locator('text=ALPHA').count()>=1, 'full Founder badge is not canonical Founder Alpha artwork')
    require(desktop.locator('.profile-badge-mark--founder').count()==0, 'rejected generic Founder badge mark remains')
    print('desktop_full_founder_badge=canonical_founder_alpha')
    desktop.get_by_role('button', name='Close').click()
    desktop.screenshot(path='/work/owner-profile-desktop.png', full_page=True)
    desktop_context.close()

    mobile_context=browser.new_context(storage_state='/work/owner-state.json', viewport={'width':390,'height':844})
    mobile=mobile_context.new_page(); watch(mobile); mobile.goto(f'{base}/profile', wait_until='networkidle', timeout=60000)
    profile_assertions(mobile, True)
    mobile.screenshot(path='/work/owner-profile-mobile-390.png', full_page=True)
    mobile_context.close(); browser.close()

if page_errors: raise RuntimeError('page errors: '+' | '.join(page_errors[:5]))
if request_failures: raise RuntimeError('request failures: '+' | '.join(request_failures[:5]))
if server_errors: raise RuntimeError('HTTP 5xx errors: '+' | '.join(server_errors[:5]))
# A known production-baseline CSP report may log in some browsers; no new console errors are accepted here.
if console_errors: raise RuntimeError('console errors: '+' | '.join(console_errors[:5]))
print('profile_unexpected_console_errors=0')
print('profile_network_errors=0')
PY

if command -v docker >/dev/null 2>&1; then runtime=docker
elif command -v podman >/dev/null 2>&1; then runtime=podman
else echo "VERIFY BLOCKED: no container runtime" >&2; exit 78; fi
if [[ "$runtime" == podman ]]; then
  podman pull "$PLAYWRIGHT_IMAGE" >/dev/null
  image="localhost/rum-owner-profile-playwright:1.57.0"
  ctx="$work/pw"; mkdir -p "$ctx"; printf 'FROM %s\nRUN python -m pip install --no-cache-dir playwright==1.57.0\n' "$PLAYWRIGHT_IMAGE" >"$ctx/Containerfile"
  podman build --pull=missing -t "$image" "$ctx" >/dev/null
else
  image="$PLAYWRIGHT_IMAGE"
fi
unset TOKEN GH_TOKEN GHCR_TOKEN
"$runtime" run --rm --ipc=host -v "$work:/work" -e RUM_BASE_URL="$BASE_URL" -e RUM_OWNER_NAME="$owner_name" -e RUM_AVATAR_MEDIA_ID="$avatar_media_id" "$image" python /work/verify.py

for f in owner-profile-desktop.png owner-profile-mobile-390.png; do
  [[ -s "$work/$f" ]] || { echo "VERIFY FAILED: screenshot missing: $f" >&2; exit 1; }
  printf 'SCREENSHOT_%s_SHA256=%s bytes=%s\n' "$(printf '%s' "$f" | tr '[:lower:].-' '[:upper:]__')" "$(sha256sum "$work/$f" | awk '{print $1}')" "$(wc -c <"$work/$f" | tr -d ' ')"
done

live_web_after="$(kctl -n "$LIVE_NAMESPACE" get deploy rum-web -o jsonpath='{.spec.template.spec.containers[0].image}')"
live_api_after="$(kctl -n "$LIVE_NAMESPACE" get deploy rum-api -o jsonpath='{.spec.template.spec.containers[0].image}')"
live_worker_after="$(kctl -n "$LIVE_NAMESPACE" get deploy rum-worker -o jsonpath='{.spec.template.spec.containers[0].image}')"
[[ "$live_web_before" == "$live_web_after" && "$live_api_before" == "$live_api_after" && "$live_worker_before" == "$live_worker_after" ]] || { echo "VERIFY FAILED: LIVE deployment changed during verification" >&2; exit 78; }
printf 'RUM_OWNER_REJECTION_PROFILE_CANDIDATE_SHA=%s\n' "$CANDIDATE_SHA"
printf 'RUM_OWNER_REJECTION_PROFILE_URL=%s/profile\n' "$BASE_URL"
printf 'LIVE_WEB_IMAGE=%s\nLIVE_API_IMAGE=%s\nLIVE_WORKER_IMAGE=%s\n' "$live_web_after" "$live_api_after" "$live_worker_after"
printf 'LIVE_RUNTIME_UNCHANGED=YES\nLIVE_MUTATION=NO\nRATE_ANYTHING_AFFECTED=NO\n'
printf 'OWNER_REJECTION_PROFILE_BROWSER_VERIFIED=YES\n'
