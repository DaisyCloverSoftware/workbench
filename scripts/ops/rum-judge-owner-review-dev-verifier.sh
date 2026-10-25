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
CANDIDATE_PR="153"
DEV_NAMESPACE="rum-dev-isolated"
DEV_HOST="dev-rum.daisycloversoftware.uk"
LIVE_HOST="rateurmate.online"
BASE_URL="https://${DEV_HOST}"
PLAYWRIGHT_IMAGE="mcr.microsoft.com/playwright/python:v1.57.0-noble"

for command in gh python3 mktemp; do
  command -v "$command" >/dev/null 2>&1 || { echo "required command unavailable: $command" >&2; exit 2; }
done
TOKEN="${GH_TOKEN:-}"
[[ -n "$TOKEN" ]] || TOKEN="$(gh auth token 2>/dev/null || true)"
[[ -n "$TOKEN" ]] || { echo "GitHub token unavailable" >&2; exit 2; }
branch_sha="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/${CANDIDATE_BRANCH}" --jq '.object.sha')"
[[ "$branch_sha" == "$CANDIDATE_SHA" ]] || { echo "VERIFY BLOCKED: candidate branch moved" >&2; exit 78; }
pr_state="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/pulls/${CANDIDATE_PR}" --jq '[.state, (.draft|tostring), .head.sha, (.merged_at // "")] | @tsv')"
IFS=$'\t' read -r state draft pr_head merged_at <<<"$pr_state"
[[ "$state" == "open" && "$draft" == "true" && "$pr_head" == "$CANDIDATE_SHA" && -z "$merged_at" ]] || { echo "VERIFY BLOCKED: PR not open/draft/unmerged exact head" >&2; exit 78; }
ci_successes="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/actions/runs?head_sha=${CANDIDATE_SHA}&event=pull_request&status=completed&per_page=100" --jq '[.workflow_runs[] | select(.name == "CI" and .head_sha == "'"${CANDIDATE_SHA}"'" and .conclusion == "success")] | length')"
[[ "$ci_successes" =~ ^[0-9]+$ && "$ci_successes" -ge 1 ]] || { echo "VERIFY BLOCKED: exact-head CI success missing" >&2; exit 78; }

kctl(){ if command -v k3s >/dev/null 2>&1; then sudo k3s kubectl "$@"; else kubectl "$@"; fi; }
dev_hosts="$(kctl -n "$DEV_NAMESPACE" get ingress rum -o jsonpath='{range .spec.rules[*]}{.host}{"\n"}{end}' 2>/dev/null || true)"
[[ "$dev_hosts" == "$DEV_HOST" ]] || { echo "VERIFY BLOCKED: isolated DEV ingress mismatch" >&2; exit 78; }
if grep -Fqx "$LIVE_HOST" <<<"$dev_hosts"; then echo "VERIFY BLOCKED: DEV ingress contains LIVE host" >&2; exit 78; fi
api_pod="$(kctl -n "$DEV_NAMESPACE" get pods -l 'app.kubernetes.io/instance=rum,app.kubernetes.io/component=api' --field-selector=status.phase=Running -o jsonpath='{.items[0].metadata.name}')"
[[ -n "$api_pod" ]] || { echo "VERIFY BLOCKED: DEV API pod unavailable" >&2; exit 78; }

work="$(mktemp -d)"; trap 'rm -rf "$work"' EXIT HUP INT TERM
suffix="$(date +%s)-$(python3 -c 'import secrets; print(secrets.token_hex(2))')"
viewer_username="rumjudge${suffix//-/}"; viewer_username="${viewer_username:0:28}"
rater_username="rumagain${suffix//-/}"; rater_username="${rater_username:0:28}"
viewer_email="${viewer_username}@example.com"
rater_email="${rater_username}@example.com"
viewer_password="$(python3 -c 'import secrets; print(secrets.token_urlsafe(24))')"
rater_password="$(python3 -c 'import secrets; print(secrets.token_urlsafe(24))')"

cat >"$work/register.py" <<'PY'
import os
from playwright.sync_api import sync_playwright
base=os.environ["RUM_BASE_URL"]; email=os.environ["RUM_EMAIL"]; username=os.environ["RUM_USERNAME"]; password=os.environ["RUM_PASSWORD"]; state=os.environ["RUM_STATE"]
with sync_playwright() as p:
    browser=p.chromium.launch(); context=browser.new_context(viewport={"width":1440,"height":1100}); page=context.new_page()
    page.goto(base, wait_until="networkidle", timeout=60000)
    page.get_by_role("button", name="Create an account").click()
    page.get_by_label("Email").fill(email); page.get_by_label("Username").fill(username)
    page.get_by_label("Password", exact=True).fill(password); page.get_by_label("Confirm password").fill(password)
    page.get_by_role("checkbox", name="I accept the Terms and Community Rules.").check(); page.get_by_role("checkbox", name="I have read the Privacy Notice.").check()
    page.get_by_role("button", name="Create account").click(); page.wait_for_url("**/me", timeout=30000)
    context.storage_state(path=state); browser.close()
print(f"judge_owner_review_registration_ok={username}")
PY

cat >"$work/verify.py" <<'PY'
import os,re
from playwright.sync_api import sync_playwright
base=os.environ["RUM_BASE_URL"]

def wait_step(page,n,name):
    header=page.locator(".step-header").filter(has_text=f"Step {n} of 5").first
    header.wait_for(state="visible", timeout=30000); header.get_by_text(name, exact=True).wait_for(state="visible", timeout=15000)

def selected_style(locator):
    return locator.evaluate("""(el) => {
      const s=getComputedStyle(el);
      const root=getComputedStyle(document.documentElement).getPropertyValue('--brand').trim();
      const probe=document.createElement('span'); probe.style.color=root; document.body.appendChild(probe);
      const brand=getComputedStyle(probe).color; probe.remove();
      return {outlineColor:s.outlineColor,outlineWidth:s.outlineWidth,outlineStyle:s.outlineStyle,
        borderTopWidth:s.borderTopWidth,borderRightWidth:s.borderRightWidth,borderBottomWidth:s.borderBottomWidth,borderLeftWidth:s.borderLeftWidth,
        boxShadow:s.boxShadow,brand};
    }""")

def assert_clean_selected_style(name, style):
    if style['outlineColor'] != style['brand']: raise RuntimeError(f"{name} selected outline is not RUM red: {style}")
    if style['outlineWidth'] != '2px' or style['outlineStyle'] != 'solid': raise RuntimeError(f"{name} selected outline is not one clean 2px solid accent: {style}")
    if any(style[key] != '0px' for key in ('borderTopWidth','borderRightWidth','borderBottomWidth','borderLeftWidth')): raise RuntimeError(f"{name} selected hit area has an extra border: {style}")
    if style['boxShadow'] not in ('none',''): raise RuntimeError(f"{name} selected hit area has an extra shadow/surround: {style}")
    print(f"{name.lower().replace(' ','_')}_selected_outline={style['outlineWidth']} {style['outlineStyle']} {style['outlineColor']}")

with sync_playwright() as p:
    browser=p.chromium.launch()
    viewer_context=browser.new_context(storage_state="/work/viewer-state.json", viewport={"width":1440,"height":1200})
    viewer=viewer_context.new_page()
    viewer.goto(f"{base}/judge", wait_until="networkidle", timeout=60000)
    viewer.locator('.judge-screen').wait_for(state='visible', timeout=30000)

    exhausted=0
    for _ in range(80):
        caught=viewer.get_by_role('heading', name="You’re all caught up")
        if caught.count() and caught.first.is_visible(): break
        next_button=viewer.get_by_role('button', name=re.compile(r'^(Next rating|Finish)$')).first
        next_button.wait_for(state='visible', timeout=30000)
        next_button.click(); exhausted += 1
    else: raise RuntimeError('Judge queue did not reach caught-up state within bounded traversal')
    viewer.get_by_role('button', name='Check again', exact=True).wait_for(state='visible', timeout=15000)
    print(f"judge_end_of_queue_reached_ok=steps:{exhausted}")
    caught_up_url=viewer.url

    rater_context=browser.new_context(storage_state="/work/rater-state.json", viewport={"width":1440,"height":1200})
    rater=rater_context.new_page()
    rater.goto(f"{base}/rate", wait_until="networkidle", timeout=60000); wait_step(rater,1,'Person')
    search=rater.get_by_role('searchbox', name='Search RUM members and public identities'); search.fill('CJ Investigates')
    card=rater.locator('.person-card').filter(has_text='CJ Investigates').first; card.wait_for(state='visible', timeout=30000); card.get_by_role('button', name='Rate', exact=True).click()
    wait_step(rater,2,'Value'); rater.get_by_text('Loading rating options…', exact=True).wait_for(state='hidden', timeout=30000)
    rater.get_by_role('radio', name='Thumbs up 5 of 5 — Exceptional').click(); rater.get_by_role('button', name='Continue', exact=True).click()
    wait_step(rater,3,'Category'); rater.locator('button.category-choice').first.wait_for(state='visible', timeout=30000); rater.locator('button.category-choice').first.click(); rater.get_by_role('button', name='Continue', exact=True).click()
    wait_step(rater,4,'Reason'); rater.get_by_label('Why this verdict?').fill('Isolated DEV Check again owner-review regression fixture.')
    public=rater.locator('input[name="audience"][value="public"]');
    if not public.is_checked(): public.check()
    rater.get_by_role('button', name='Continue', exact=True).click(); wait_step(rater,5,'Review')
    with rater.expect_response(lambda response: '/api/v1/rate-anything/rating' in response.url and response.request.method == 'POST', timeout=30000) as submitted:
        rater.get_by_role('button', name='Submit rating', exact=True).click()
    rating_response=submitted.value
    if rating_response.status not in (200,201): raise RuntimeError(f'fixture rating submission returned {rating_response.status}')
    rating_id=str(rating_response.json().get('data',{}).get('id',''))
    if not rating_id: raise RuntimeError('fixture rating submission returned no canonical event id')
    rater.get_by_role('heading', name='Your verdict is in.').wait_for(state='visible', timeout=30000)
    print(f"judge_check_again_new_rating_event={rating_id}")

    document_requests=[]
    viewer.on('request', lambda request: document_requests.append(request.url) if request.resource_type == 'document' else None)
    with viewer.expect_response(lambda response: '/api/v1/judge?' in response.url and response.request.method == 'GET', timeout=30000) as checked:
        viewer.get_by_role('button', name='Check again', exact=True).click()
    check_response=checked.value
    if check_response.status != 200: raise RuntimeError(f'Check again Judge fetch returned {check_response.status}')
    cache_control=check_response.headers.get('cache-control','').lower()
    if 'no-store' not in cache_control: raise RuntimeError(f'Check again Judge response was cacheable: {cache_control}')
    check_body=check_response.json(); returned_ids=[str(item.get('id')) for item in check_body.get('data',[])]
    if rating_id not in returned_ids: raise RuntimeError('Check again authoritative response did not contain the newly eligible rating')
    viewer.locator('.judge-card').first.wait_for(state='visible', timeout=30000)
    viewer.get_by_text('CJ Investigates', exact=True).first.wait_for(state='visible', timeout=30000)
    if viewer.url != caught_up_url: raise RuntimeError(f'Check again navigated away: {caught_up_url} -> {viewer.url}')
    if document_requests: raise RuntimeError('Check again caused a document navigation/reload: '+' | '.join(document_requests[:5]))
    print('check_again_alone_reloaded_new_rating_ok')
    print('check_again_no_refresh_navigation_or_reload_ok')
    print(f'check_again_cache_control={cache_control}')

    keep=viewer.get_by_role('button', name='Keep', exact=True); keep.wait_for(state='visible', timeout=15000); keep.click(); keep.wait_for(state='visible', timeout=15000)
    viewer.wait_for_function("() => document.querySelector('button[aria-label=\"Keep\"]')?.getAttribute('aria-pressed') === 'true'")
    keep_style=selected_style(keep); assert_clean_selected_style('Keep', keep_style)

    sinbin=viewer.get_by_role('button', name='Sin Bin', exact=True); sinbin.click(); viewer.wait_for_function("() => document.querySelector('button[aria-label=\"Sin Bin\"]')?.getAttribute('aria-pressed') === 'true'")
    sinbin_style=selected_style(sinbin); assert_clean_selected_style('Sin Bin', sinbin_style)
    print('judge_selected_thumb_single_rum_red_outline_ok')
    print('judge_thumb_artwork_unchanged_ok=/brand/judge-voting-thumbs.png')

    viewer_context.close(); rater_context.close(); browser.close()

print('judge_owner_review_manual_browser_verification_ok')
PY

if command -v docker >/dev/null 2>&1; then runtime=docker; elif command -v podman >/dev/null 2>&1; then runtime=podman; else echo "VERIFY BLOCKED: no container runtime" >&2; exit 78; fi
if [[ "$runtime" == podman ]]; then
  runtime_bin="$work/runtime-bin"; mkdir -p "$runtime_bin"; ln -s "$(command -v podman)" "$runtime_bin/docker"; export PATH="$runtime_bin:$PATH"
  ctx="$work/playwright-compat"; mkdir -p "$ctx"; printf 'FROM %s\nRUN python -m pip install --no-cache-dir playwright==1.57.0\n' "$PLAYWRIGHT_IMAGE" >"$ctx/Containerfile"
  podman pull "$PLAYWRIGHT_IMAGE" >/dev/null
  podman build --pull=missing -t localhost/rum-judge-owner-review-playwright:1.57.0 "$ctx" >/dev/null
  podman tag localhost/rum-judge-owner-review-playwright:1.57.0 "$PLAYWRIGHT_IMAGE"
fi
printf 'RUM_JUDGE_OWNER_REVIEW_CONTAINER_RUNTIME=%s\n' "$runtime"

unset TOKEN GH_TOKEN GHCR_TOKEN
register(){
  local email="$1" username="$2" password="$3" state="$4"
  "$runtime" run --rm --network host -v "$work:/work:Z" \
    -e RUM_BASE_URL="$BASE_URL" -e RUM_EMAIL="$email" -e RUM_USERNAME="$username" -e RUM_PASSWORD="$password" -e RUM_STATE="/work/$state" \
    "$PLAYWRIGHT_IMAGE" python /work/register.py
}
register "$viewer_email" "$viewer_username" "$viewer_password" viewer-state.json
register "$rater_email" "$rater_username" "$rater_password" rater-state.json

b64(){ printf '%s' "$1" | python3 -c 'import base64,sys; print(base64.b64encode(sys.stdin.buffer.read()).decode())'; }
viewer_b64="$(b64 "$viewer_email")"; rater_b64="$(b64 "$rater_email")"
PHP_BOOT='require "/var/www/html/vendor/autoload.php"; $app=require "/var/www/html/bootstrap/app.php"; $app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap(); eval($argv[1]);'
kctl -n "$DEV_NAMESPACE" exec "$api_pod" -c php-fpm -- php -r "$PHP_BOOT" "\$v=App\\Models\\User::where('email',base64_decode('${viewer_b64}'))->firstOrFail(); \$r=App\\Models\\User::where('email',base64_decode('${rater_b64}'))->firstOrFail(); \$v->forceFill(['email_verified_at'=>now(),'created_at'=>now()->subDays(2)])->save(); \$r->forceFill(['email_verified_at'=>now()])->save();" >/dev/null
printf 'judge_owner_review_dev_accounts_prepared_ok\n'

"$runtime" run --rm --network host -v "$work:/work:Z" -e RUM_BASE_URL="$BASE_URL" "$PLAYWRIGHT_IMAGE" python /work/verify.py

printf 'RUM_JUDGE_OWNER_REVIEW_CANDIDATE_SHA=%s\n' "$CANDIDATE_SHA"
printf 'RUM_JUDGE_OWNER_REVIEW_NAMESPACE=%s\n' "$DEV_NAMESPACE"
printf 'LIVE_RUNTIME_AFFECTED=NO\n'
printf 'RATE_ANYTHING_AFFECTED=NO\n'
