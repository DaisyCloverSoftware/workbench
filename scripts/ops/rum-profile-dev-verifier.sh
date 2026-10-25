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
CANDIDATE_BRANCH="sprint-0-rum-owner-rating-flow-20260823"
CANDIDATE_PR=153
DEV_NAMESPACE="rum-dev-isolated"
DEV_HOST="dev-rum.daisycloversoftware.uk"
LIVE_HOST="rateurmate.online"
BASE_URL="https://${DEV_HOST}"
EXPECTED_TAG="sha-${CANDIDATE_SHA:0:8}"
PLAYWRIGHT_IMAGE="mcr.microsoft.com/playwright/python:v1.57.0-noble"

for command in gh python3 mktemp; do
  command -v "$command" >/dev/null 2>&1 || { echo "VERIFY BLOCKED: required command unavailable: $command" >&2; exit 2; }
done
TOKEN="${GH_TOKEN:-}"
[[ -n "$TOKEN" ]] || TOKEN="$(gh auth token 2>/dev/null || true)"
[[ -n "$TOKEN" ]] || { echo "VERIFY BLOCKED: GitHub token unavailable" >&2; exit 2; }
branch_sha="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/${CANDIDATE_BRANCH}" --jq '.object.sha')"
[[ "$branch_sha" == "$CANDIDATE_SHA" ]] || { echo "VERIFY BLOCKED: candidate branch moved" >&2; exit 78; }
pr_state="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/pulls/${CANDIDATE_PR}" --jq '[.state, (.draft|tostring), .head.sha, (.merged_at // "")] | @tsv')"
IFS=$'\t' read -r state draft pr_head merged_at <<<"$pr_state"
[[ "$state" == "open" && "$draft" == "true" && "$pr_head" == "$CANDIDATE_SHA" && -z "$merged_at" ]] || { echo "VERIFY BLOCKED: PR is not open/draft/unmerged exact head" >&2; exit 78; }
ci_successes="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/actions/runs?head_sha=${CANDIDATE_SHA}&event=pull_request&status=completed&per_page=100" --jq '[.workflow_runs[] | select(.name == "CI" and .head_sha == "'"${CANDIDATE_SHA}"'" and .conclusion == "success")] | length')"
[[ "$ci_successes" =~ ^[0-9]+$ && "$ci_successes" -ge 1 ]] || { echo "VERIFY BLOCKED: exact-head CI success missing" >&2; exit 78; }

kctl(){ if command -v k3s >/dev/null 2>&1; then sudo k3s kubectl "$@"; else kubectl "$@"; fi; }
dev_hosts="$(kctl -n "$DEV_NAMESPACE" get ingress rum -o jsonpath='{range .spec.rules[*]}{.host}{"\n"}{end}' 2>/dev/null || true)"
[[ "$dev_hosts" == "$DEV_HOST" ]] || { echo "VERIFY BLOCKED: isolated DEV ingress mismatch: $dev_hosts" >&2; exit 78; }
if grep -Fqx "$LIVE_HOST" <<<"$dev_hosts"; then echo "VERIFY BLOCKED: isolated DEV ingress contains LIVE host" >&2; exit 78; fi
for component in web api worker; do
  image="$(kctl -n "$DEV_NAMESPACE" get deploy "rum-${component}" -o jsonpath='{.spec.template.spec.containers[0].image}')"
  [[ "$image" == *":${EXPECTED_TAG}" ]] || { echo "VERIFY BLOCKED: rum-${component} is not exact candidate ${EXPECTED_TAG}: ${image}" >&2; exit 78; }
done
api_pod="$(kctl -n "$DEV_NAMESPACE" get pods -l 'app.kubernetes.io/instance=rum,app.kubernetes.io/component=api' --field-selector=status.phase=Running -o jsonpath='{.items[0].metadata.name}')"
[[ -n "$api_pod" ]] || { echo "VERIFY BLOCKED: isolated DEV API pod unavailable" >&2; exit 78; }

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT HUP INT TERM
suffix="$(date +%s)-$(python3 -c 'import secrets; print(secrets.token_hex(2))')"
owner_username="rumprofile${suffix//-/}"; owner_username="${owner_username:0:28}"
rater_username="rumprater${suffix//-/}"; rater_username="${rater_username:0:28}"
viewer_username="rumpview${suffix//-/}"; viewer_username="${viewer_username:0:28}"
owner_email="${owner_username}@example.com"; rater_email="${rater_username}@example.com"; viewer_email="${viewer_username}@example.com"
owner_password="$(python3 -c 'import secrets; print(secrets.token_urlsafe(24))')"
rater_password="$(python3 -c 'import secrets; print(secrets.token_urlsafe(24))')"
viewer_password="$(python3 -c 'import secrets; print(secrets.token_urlsafe(24))')"

cat >"$work/register.py" <<'PY'
import os
from playwright.sync_api import sync_playwright
base=os.environ['RUM_BASE_URL']; email=os.environ['RUM_EMAIL']; username=os.environ['RUM_USERNAME']; password=os.environ['RUM_PASSWORD']; state=os.environ['RUM_STATE']
with sync_playwright() as p:
    browser=p.chromium.launch(); context=browser.new_context(viewport={'width':1280,'height':1000}); page=context.new_page()
    page.goto(base, wait_until='networkidle', timeout=60000)
    page.get_by_role('button', name='Create an account').click()
    page.get_by_label('Email').fill(email); page.get_by_label('Username').fill(username)
    page.get_by_label('Password', exact=True).fill(password); page.get_by_label('Confirm password').fill(password)
    page.get_by_role('checkbox', name='I accept the Terms and Community Rules.').check(); page.get_by_role('checkbox', name='I have read the Privacy Notice.').check()
    page.get_by_role('button', name='Create account').click(); page.wait_for_url('**/me', timeout=30000)
    context.storage_state(path=state); browser.close()
print(f'profile_registration_ok={username}')
PY

cat >"$work/rate.py" <<'PY'
import os
from playwright.sync_api import sync_playwright
base=os.environ['RUM_BASE_URL']; target=os.environ['RUM_TARGET']; state=os.environ['RUM_STATE']; out=os.environ['RUM_RATING_ID_FILE']

def wait_step(page,n,name):
    header=page.locator('.step-header').filter(has_text=f'Step {n} of 5').first
    header.wait_for(state='visible', timeout=30000); header.get_by_text(name, exact=True).wait_for(state='visible', timeout=15000)

with sync_playwright() as p:
    browser=p.chromium.launch(); context=browser.new_context(storage_state=state, viewport={'width':1280,'height':1100}); page=context.new_page()
    page.goto(f'{base}/rate', wait_until='networkidle', timeout=60000); wait_step(page,1,'Person')
    search=page.get_by_role('searchbox', name='Search RUM members and public identities'); search.fill(target)
    member=page.locator('button.select-person').filter(has_text=target).first; member.wait_for(state='visible', timeout=30000); member.click()
    page.get_by_role('button', name='Continue', exact=True).click(); wait_step(page,2,'Value')
    page.get_by_text('Loading rating options…', exact=True).wait_for(state='hidden', timeout=30000)
    page.get_by_role('radio', name='Thumbs up 5 of 5 — Exceptional').click(); page.get_by_role('button', name='Continue', exact=True).click(); wait_step(page,3,'Category')
    page.locator('button.category-choice').first.wait_for(state='visible', timeout=30000); page.locator('button.category-choice').first.click(); page.get_by_role('button', name='Continue', exact=True).click(); wait_step(page,4,'Reason')
    page.get_by_label('Why this verdict?').fill('Profile browser integration rating.')
    public=page.locator('input[name="audience"][value="public"]')
    if not public.is_checked(): public.check()
    page.get_by_role('button', name='Continue', exact=True).click(); wait_step(page,5,'Review')
    with page.expect_response(lambda response: response.request.method == 'POST' and '/api/v1/rate-anything/rating' in response.url, timeout=30000) as submitted:
        page.get_by_role('button', name='Submit rating', exact=True).click()
    response=submitted.value
    if response.status not in (200,201): raise RuntimeError(f'profile fixture rating returned {response.status}')
    rating_id=str(response.json().get('data',{}).get('id',''))
    if not rating_id: raise RuntimeError('profile fixture rating returned no canonical rating id')
    page.get_by_role('heading', name='Your verdict is in.').wait_for(state='visible', timeout=30000)
    open(out,'w').write(rating_id)
    context.close(); browser.close()
print(f'profile_fixture_rating_ok={rating_id}')
PY

cat >"$work/verify.py" <<'PY'
import os,base64
from playwright.sync_api import sync_playwright
base=os.environ['RUM_BASE_URL']; owner_id=os.environ['RUM_OWNER_ID']; owner_name=os.environ['RUM_OWNER_NAME']; rating_id=os.environ['RUM_RATING_ID']
errors=[]

def watch(page):
    page.on('pageerror', lambda err: errors.append(f'pageerror:{err}'))
    page.on('console', lambda msg: errors.append(f'console:{msg.text}') if msg.type == 'error' else None)

def csrf_patch_visibility(page, value):
    result=page.evaluate("""async (value) => {
      await fetch('/sanctum/csrf-cookie', {credentials:'same-origin'});
      const part=document.cookie.split('; ').find(v => v.startsWith('XSRF-TOKEN='));
      const token=part ? decodeURIComponent(part.slice('XSRF-TOKEN='.length)) : '';
      const response=await fetch('/api/v1/me/profile', {method:'PATCH', credentials:'same-origin', headers:{Accept:'application/json','Content-Type':'application/json','X-XSRF-TOKEN':token}, body:JSON.stringify({profileVisibility:value})});
      return {status:response.status, body:await response.json().catch(()=>({}))};
    }""", value)
    if result['status'] != 200: raise RuntimeError(f'profile visibility patch {value} returned {result}')

def judge_item(page):
    return page.evaluate("""async (ratingId) => {
      const response=await fetch('/api/v1/judge?queue=all&scope=all&limit=50', {credentials:'same-origin', headers:{Accept:'application/json'}, cache:'no-store'});
      if (!response.ok) return {status:response.status};
      const body=await response.json();
      return {status:response.status, item:(body.data||[]).find(item => String(item.id) === String(ratingId)) || null};
    }""", rating_id)

with sync_playwright() as p:
    browser=p.chromium.launch()
    owner_context=browser.new_context(storage_state='/work/owner-state.json', viewport={'width':1280,'height':1100})
    owner=owner_context.new_page(); watch(owner)
    owner.goto(f'{base}/profile', wait_until='networkidle', timeout=60000)
    owner.get_by_role('heading', name=owner_name, exact=True).wait_for(state='visible', timeout=30000)
    owner.get_by_text('F #27', exact=True).wait_for(state='visible', timeout=15000)
    for text in ('Overall Public Rating','Rate My Rating','Filtered Rating','Ratings Received','Ratings Given'):
        owner.get_by_text(text, exact=True).first.wait_for(state='visible', timeout=15000)
    owner.get_by_role('searchbox', name='Filter this view by category').wait_for(state='visible', timeout=15000)
    card=owner.locator('.profile-rating-card').filter(has_text='Profile browser integration rating.').first; card.wait_for(state='visible', timeout=30000)
    card.get_by_role('button', name='Help Me Out', exact=True).click()
    owner.get_by_text('Help Me Out active', exact=True).wait_for(state='visible', timeout=30000)
    card=owner.locator('.profile-rating-card').filter(has_text='Profile browser integration rating.').first
    card.locator('.profile-attention--help').get_by_text('Help Me Out', exact=True).wait_for(state='visible', timeout=15000)
    owner.get_by_role('button', name='Badges').click()
    owner.get_by_text('Founder #27 · Platinum', exact=True).wait_for(state='visible', timeout=15000)
    owner.get_by_role('heading', name='First Rating Received', exact=True).wait_for(state='visible', timeout=15000)
    owner.get_by_role('button', name='Close').click()
    print('profile_owner_header_summary_history_help_badges_ok')

    viewer_context=browser.new_context(storage_state='/work/viewer-state.json', viewport={'width':1280,'height':1100})
    viewer=viewer_context.new_page(); watch(viewer)
    viewer.goto(f'{base}/people/{owner_id}/profile', wait_until='networkidle', timeout=60000)
    viewer.get_by_role('heading', name=owner_name, exact=True).wait_for(state='visible', timeout=30000)
    viewer_card=viewer.locator('.profile-rating-card').filter(has_text='Profile browser integration rating.').first; viewer_card.wait_for(state='visible', timeout=30000)
    keep=viewer_card.locator('button[aria-label="Keep"]'); keep.wait_for(state='visible', timeout=15000); keep.click()
    viewer.wait_for_function("() => document.querySelector('.profile-rating-card button[aria-label=\"Keep\"]')?.getAttribute('aria-pressed') === 'true'")
    current=judge_item(viewer)
    if current.get('status') != 200 or not current.get('item') or current['item'].get('viewerVote') != 1:
        raise RuntimeError(f'profile Keep vote did not reconcile to canonical Judge item: {current}')
    sinbin=viewer_card.locator('button[aria-label="Sin Bin"]'); sinbin.click()
    viewer.wait_for_function("() => document.querySelector('.profile-rating-card button[aria-label=\"Sin Bin\"]')?.getAttribute('aria-pressed') === 'true'")
    current=judge_item(viewer)
    if current.get('status') != 200 or not current.get('item') or current['item'].get('viewerVote') != -1:
        raise RuntimeError(f'profile Sin Bin vote did not replace canonical Judge vote: {current}')
    viewer_card.get_by_role('button', name='Remove my vote', exact=True).click()
    viewer.get_by_text('Judge vote removed', exact=True).wait_for(state='visible', timeout=30000)
    current=judge_item(viewer)
    if current.get('status') != 200 or not current.get('item') or current['item'].get('viewerVote') is not None:
        raise RuntimeError(f'profile vote removal did not reconcile to canonical Judge item: {current}')
    print('profile_judge_shared_changeable_vote_ok')

    csrf_patch_visibility(owner, 'private')
    viewer.goto(f'{base}/people/{owner_id}/profile', wait_until='networkidle', timeout=60000)
    viewer.get_by_role('heading', name='Profile unavailable', exact=True).wait_for(state='visible', timeout=30000)
    owner.goto(f'{base}/profile', wait_until='networkidle', timeout=60000)
    owner.get_by_role('heading', name=owner_name, exact=True).wait_for(state='visible', timeout=30000)
    csrf_patch_visibility(owner, 'members')
    print('profile_privacy_fail_closed_owner_access_ok')

    mobile_context=browser.new_context(storage_state='/work/owner-state.json', viewport={'width':390,'height':844})
    mobile=mobile_context.new_page(); watch(mobile); mobile.goto(f'{base}/profile', wait_until='networkidle', timeout=60000)
    mobile.get_by_role('heading', name=owner_name, exact=True).wait_for(state='visible', timeout=30000)
    overflow=mobile.evaluate('() => ({scroll:document.documentElement.scrollWidth, inner:window.innerWidth})')
    if overflow['scroll'] > overflow['inner'] + 1: raise RuntimeError(f'profile mobile horizontal overflow: {overflow}')
    mobile.screenshot(path='/work/profile-owner-mobile.png', full_page=True)
    with open('/work/profile-owner-mobile.png','rb') as fh:
        print('profile_owner_mobile_png_base64='+base64.b64encode(fh.read()).decode())
    print(f"profile_mobile_no_horizontal_overflow={overflow['scroll']}<={overflow['inner']+1}")

    if errors:
        raise RuntimeError('unexpected profile browser errors: '+' | '.join(errors[:10]))
    mobile_context.close(); viewer_context.close(); owner_context.close(); browser.close()
print('RUM_PROFILE_BROWSER_VERIFICATION_OK')
PY

if command -v docker >/dev/null 2>&1; then runtime=docker; elif command -v podman >/dev/null 2>&1; then runtime=podman; else echo "VERIFY BLOCKED: no container runtime" >&2; exit 78; fi
if [[ "$runtime" == podman ]]; then
  runtime_bin="$work/runtime-bin"; mkdir -p "$runtime_bin"; ln -s "$(command -v podman)" "$runtime_bin/docker"; export PATH="$runtime_bin:$PATH"
  ctx="$work/playwright-compat"; mkdir -p "$ctx"; printf 'FROM %s\nRUN python -m pip install --no-cache-dir playwright==1.57.0\n' "$PLAYWRIGHT_IMAGE" >"$ctx/Containerfile"
  podman pull "$PLAYWRIGHT_IMAGE" >/dev/null
  podman build --pull=missing -t localhost/rum-profile-playwright:1.57.0 "$ctx" >/dev/null
  podman tag localhost/rum-profile-playwright:1.57.0 "$PLAYWRIGHT_IMAGE"
fi
printf 'RUM_PROFILE_CONTAINER_RUNTIME=%s\n' "$runtime"

unset TOKEN GH_TOKEN GHCR_TOKEN
register(){
  local email="$1" username="$2" password="$3" state="$4"
  "$runtime" run --rm --network host -v "$work:/work:Z" \
    -e RUM_BASE_URL="$BASE_URL" -e RUM_EMAIL="$email" -e RUM_USERNAME="$username" -e RUM_PASSWORD="$password" -e RUM_STATE="/work/$state" \
    "$PLAYWRIGHT_IMAGE" python /work/register.py
}
register "$owner_email" "$owner_username" "$owner_password" owner-state.json
register "$rater_email" "$rater_username" "$rater_password" rater-state.json
register "$viewer_email" "$viewer_username" "$viewer_password" viewer-state.json

b64(){ printf '%s' "$1" | python3 -c 'import base64,sys; print(base64.b64encode(sys.stdin.buffer.read()).decode())'; }
owner_b64="$(b64 "$owner_email")"; rater_b64="$(b64 "$rater_email")"; viewer_b64="$(b64 "$viewer_email")"
PHP_BOOT='require "/var/www/html/vendor/autoload.php"; $app=require "/var/www/html/bootstrap/app.php"; $app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap(); eval($argv[1]);'
identity_line="$(kctl -n "$DEV_NAMESPACE" exec "$api_pod" -c php-fpm -- php -r "$PHP_BOOT" "\$o=App\\Models\\User::where('email',base64_decode('${owner_b64}'))->firstOrFail(); \$r=App\\Models\\User::where('email',base64_decode('${rater_b64}'))->firstOrFail(); \$v=App\\Models\\User::where('email',base64_decode('${viewer_b64}'))->firstOrFail(); foreach([\$o,\$r,\$v] as \$u){\$u->forceFill(['email_verified_at'=>now(),'created_at'=>now()->subDays(30)])->save();} \$o->forceFill(['founder_number'=>27])->save(); App\\Models\\MateRelationship::query()->updateOrCreate(['requester_id'=>\$o->id,'addressee_id'=>\$r->id],['status'=>'accepted','accepted_at'=>now()]); echo \$o->id.'|'.\$o->profile->display_name;")"
owner_id="${identity_line%%|*}"; owner_name="${identity_line#*|}"
[[ "$owner_id" =~ ^[0-9a-z]{26}$ && -n "$owner_name" ]] || { echo "VERIFY BLOCKED: fixture identity setup failed: $identity_line" >&2; exit 70; }

"$runtime" run --rm --network host -v "$work:/work:Z" \
  -e RUM_BASE_URL="$BASE_URL" -e RUM_TARGET="$owner_username" -e RUM_STATE=/work/rater-state.json -e RUM_RATING_ID_FILE=/work/rating-id \
  "$PLAYWRIGHT_IMAGE" python /work/rate.py

rating_id="$(cat "$work/rating-id")"
[[ "$rating_id" =~ ^[0-9a-z]{26}$ ]] || { echo "VERIFY BLOCKED: invalid profile fixture rating id" >&2; exit 70; }

"$runtime" run --rm --network host -v "$work:/work:Z" \
  -e RUM_BASE_URL="$BASE_URL" -e RUM_OWNER_ID="$owner_id" -e RUM_OWNER_NAME="$owner_name" -e RUM_RATING_ID="$rating_id" \
  "$PLAYWRIGHT_IMAGE" python /work/verify.py

printf 'RUM_PROFILE_CANDIDATE_SHA=%s\n' "$CANDIDATE_SHA"
printf 'RUM_PROFILE_DEPLOY_TAG=%s\n' "$EXPECTED_TAG"
printf 'RUM_PROFILE_DEV_HOST=%s\n' "$DEV_HOST"
printf 'RUM_PROFILE_LIVE_MUTATION=NO\n'
printf 'RATE_ANYTHING_AFFECTED=NO\n'
