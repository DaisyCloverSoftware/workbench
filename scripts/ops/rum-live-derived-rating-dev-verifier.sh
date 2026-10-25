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
EXPECTED_CI_NAME="CI"
DEV_NAMESPACE="rum-dev-isolated"
DEV_HOST="dev-rum.daisycloversoftware.uk"
LIVE_HOST="rateurmate.online"
BASE_URL="https://${DEV_HOST}"
PLAYWRIGHT_IMAGE="mcr.microsoft.com/playwright/python:v1.57.0-noble"
LIVE_BASELINE_CSP_HASH="sha256-A0FJyCgxFUPhG7nac5LcQPwVRK5So9ZNz7x5ubsD9kU="
LIVE_REFERENCE_SHA="8106675325eb8a516696bfb45cf817e97f03d7f5"

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
ci_successes="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/actions/runs?head_sha=${CANDIDATE_SHA}&event=pull_request&status=completed&per_page=100" --jq '[.workflow_runs[] | select(.name == "'"${EXPECTED_CI_NAME}"'" and .head_sha == "'"${CANDIDATE_SHA}"'" and .conclusion == "success")] | length')"
[[ "$ci_successes" =~ ^[0-9]+$ && "$ci_successes" -ge 1 ]] || { echo "VERIFY BLOCKED: exact-head CI success missing" >&2; exit 78; }

kctl(){ if command -v k3s >/dev/null 2>&1; then sudo k3s kubectl "$@"; else kubectl "$@"; fi; }
dev_hosts="$(kctl -n "$DEV_NAMESPACE" get ingress rum -o jsonpath='{range .spec.rules[*]}{.host}{"\n"}{end}' 2>/dev/null || true)"
[[ "$dev_hosts" == "$DEV_HOST" ]] || { echo "VERIFY BLOCKED: isolated DEV ingress mismatch" >&2; exit 78; }
if grep -Fqx "$LIVE_HOST" <<<"$dev_hosts"; then echo "VERIFY BLOCKED: DEV ingress contains LIVE host" >&2; exit 78; fi
api_pod="$(kctl -n "$DEV_NAMESPACE" get pods -l 'app.kubernetes.io/instance=rum,app.kubernetes.io/component=api' --field-selector=status.phase=Running -o jsonpath='{.items[0].metadata.name}')"
[[ -n "$api_pod" ]] || { echo "VERIFY BLOCKED: DEV API pod unavailable" >&2; exit 78; }

work="$(mktemp -d)"; trap 'rm -rf "$work"' EXIT HUP INT TERM
suffix="$(date +%s)-$(python3 -c 'import secrets; print(secrets.token_hex(2))')"
rater_username="rumrate${suffix//-/}"; rater_username="${rater_username:0:28}"
mate_username="rummate${suffix//-/}"; mate_username="${mate_username:0:28}"
rater_email="${rater_username}@example.com"
mate_email="${mate_username}@example.com"
rater_password="$(python3 -c 'import secrets; print(secrets.token_urlsafe(24))')"
mate_password="$(python3 -c 'import secrets; print(secrets.token_urlsafe(24))')"
linked_name="Quillstone${suffix//-/}Zeta"

cat >"$work/register.py" <<'PY'
import os
from playwright.sync_api import sync_playwright
base=os.environ["RUM_BASE_URL"]; email=os.environ["RUM_EMAIL"]; username=os.environ["RUM_USERNAME"]; password=os.environ["RUM_PASSWORD"]; state=os.environ["RUM_STATE"]
with sync_playwright() as p:
    browser=p.chromium.launch(); context=browser.new_context(viewport={"width":1440,"height":1000}); page=context.new_page()
    page.goto(base, wait_until="networkidle", timeout=60000)
    page.get_by_role("button", name="Create an account").click()
    page.get_by_label("Email").fill(email); page.get_by_label("Username").fill(username)
    page.get_by_label("Password", exact=True).fill(password); page.get_by_label("Confirm password").fill(password)
    page.get_by_role("checkbox", name="I accept the Terms and Community Rules.").check(); page.get_by_role("checkbox", name="I have read the Privacy Notice.").check()
    page.get_by_role("button", name="Create account").click(); page.wait_for_url("**/me", timeout=30000)
    context.storage_state(path=state); browser.close()
print(f"registration_session_ok={username}")
PY

cat >"$work/exercise.py" <<'PY'
import os,re
from playwright.sync_api import sync_playwright, TimeoutError as PlaywrightTimeoutError
base=os.environ["RUM_BASE_URL"]; mate_username=os.environ["RUM_MATE_USERNAME"]; linked_name=os.environ["RUM_LINKED_NAME"]; baseline=os.environ["RUM_BASELINE_CSP"]
console_errors=[]; page_errors=[]; request_failures=[]; api_failures=[]; identity_used=None

def wait_step(page,n,name):
    header=page.locator(".step-header").filter(has_text=f"Step {n} of 5").first
    header.wait_for(state="visible", timeout=30000); header.get_by_text(name, exact=True).wait_for(state="visible", timeout=15000)

def shortlist(page):
    choices=page.locator("button.category-choice"); choices.first.wait_for(state="visible", timeout=30000); count=choices.count()
    if count < 1 or count > 5: raise RuntimeError(f"Expected 1..5 category choices, found {count}")
    print(f"category_shortlist_count={count}"); print("category_shortlist_max_5_ok")

def rate(page,mode="first"):
    wait_step(page,2,"Value"); page.get_by_text("Loading rating options…", exact=True).wait_for(state="hidden", timeout=30000)
    page.get_by_role("radio", name="Thumbs up 5 of 5 — Exceptional").click(); page.get_by_role("button", name="Continue", exact=True).click()
    wait_step(page,3,"Category"); page.get_by_role("heading", name="Pick a category").wait_for(state="visible", timeout=15000); shortlist(page)
    search=page.get_by_role("searchbox", name="Search categories")
    if mode == "search":
        search.fill("qual"); page.get_by_text("Quality", exact=True).first.wait_for(state="visible", timeout=30000); page.locator("button.category-choice").filter(has_text="Quality").first.click(); print("category_substring_search_ok")
    elif mode == "custom":
        custom=f"Flowcraft {linked_name[-8:]}"; search.fill(custom); page.locator("button.category-choice").filter(has_text=f"Use “{custom}”").first.wait_for(state="visible", timeout=30000); print("custom_category_affordance_ok"); search.fill(""); page.locator("button.category-choice").first.wait_for(state="visible", timeout=30000); page.locator("button.category-choice").first.click()
    else: page.locator("button.category-choice").first.click()
    page.get_by_role("button", name="Continue", exact=True).click(); wait_step(page,4,"Reason")
    page.get_by_role("heading", name="Add the reason").wait_for(state="visible", timeout=15000); page.get_by_label("Why this verdict?").fill("Specific isolated DEV verification of the LIVE-derived RUM rating progression.")
    public=page.locator('input[name="audience"][value="public"]');
    if not public.is_checked(): public.check()
    page.get_by_role("button", name="Continue", exact=True).click(); wait_step(page,5,"Review"); page.get_by_role("heading", name="Check your rating").wait_for(state="visible", timeout=15000); page.get_by_role("button", name="Submit rating", exact=True).click()

with sync_playwright() as p:
    browser=p.chromium.launch(); context=browser.new_context(storage_state="/work/rater-state.json", viewport={"width":1440,"height":1200}); page=context.new_page()
    page.on("console", lambda msg: console_errors.append(msg.text) if msg.type=="error" else None); page.on("pageerror", lambda err: page_errors.append(str(err))); page.on("requestfailed", lambda req: request_failures.append(f"{req.method} {req.url}")); page.on("response", lambda res: api_failures.append(f"{res.status} {res.url}") if res.status>=400 and "/api/" in res.url else None)

    page.goto(f"{base}/rate", wait_until="networkidle", timeout=60000); wait_step(page,1,"Person"); search=page.get_by_role("searchbox", name="Search RUM members and public identities"); search.fill(mate_username)
    mate=page.locator("button.select-person").filter(has_text=mate_username).first; mate.wait_for(state="visible", timeout=30000); mate.click(); page.get_by_role("button", name="Continue", exact=True).click(); rate(page)
    page.wait_for_url(re.compile(r"/judge(?:$|[?#])"), timeout=30000)
    if page.get_by_role("button", name="Add or rate a linked thing", exact=True).count()!=0: raise RuntimeError("Mate exposed linked continuation")
    print("mate_live_derived_five_step_flow_ok"); print("mate_linked_offer_absent_ok")

    page.goto(f"{base}/rate", wait_until="networkidle", timeout=60000); wait_step(page,1,"Person"); search=page.get_by_role("searchbox", name="Search RUM members and public identities")
    for candidate in ("CJ Investigates","Colin Furze"):
        search.fill(candidate); card=page.locator(".person-card").filter(has_text=candidate).first
        try: card.wait_for(state="visible", timeout=15000)
        except PlaywrightTimeoutError: continue
        card.get_by_role("button", name="Rate", exact=True).click(); identity_used=candidate; break
    if identity_used is None: raise RuntimeError("No seeded public identity available")
    rate(page); page.get_by_role("heading", name="Your verdict is in.").wait_for(state="visible", timeout=30000); outer=page.get_by_role("button", name="Add or rate a linked thing", exact=True).first; outer.wait_for(state="visible", timeout=30000)
    print(f"public_identity={identity_used}"); print("public_live_derived_five_step_flow_ok"); print("public_linked_offer_visible_ok")

    outer.click(); chooser=page.locator("section.rating-reason-panel").filter(has_text="Add or rate a linked thing").first; chooser.wait_for(state="visible", timeout=15000); chooser.get_by_role("searchbox", name="Search linked things").wait_for(state="visible", timeout=15000); chooser.get_by_role("button", name="Add or rate a linked thing", exact=True).click()
    page.get_by_text("Search RUM before adding a missing linked thing", exact=True).wait_for(state="visible", timeout=15000); page.get_by_label("Search existing things before adding").fill(linked_name); page.get_by_role("button", name="Search RUM", exact=True).click(); page.get_by_role("button", name="None of these — add a linked thing", exact=True).wait_for(state="visible", timeout=30000); page.get_by_role("button", name="None of these — add a linked thing", exact=True).click(); page.get_by_label("Linked thing type").select_option("product"); page.get_by_label("Linked thing description").fill("Disposable linked Thing created only in isolated DEV to verify the LIVE-derived rating flow."); page.get_by_role("button", name="Check and add linked thing", exact=True).click(); rate(page,"search"); page.wait_for_url(re.compile(r"/judge(?:$|[?#])"), timeout=30000)
    if page.get_by_role("button", name="Add or rate a linked thing", exact=True).count()!=0: raise RuntimeError("Linked Thing cascaded continuation")
    print(f"linked_thing={linked_name}"); print("linked_thing_live_derived_five_step_flow_ok"); print("linked_thing_no_cascade_ok")

    page.goto(f"{base}/rate", wait_until="networkidle", timeout=60000); wait_step(page,1,"Person"); search=page.get_by_role("searchbox", name="Search RUM members and public identities"); search.fill(linked_name); thing=page.locator("button.select-person").filter(has_text=linked_name).first; thing.wait_for(state="visible", timeout=30000); thing.click(); rate(page,"custom"); page.get_by_role("heading", name="Your verdict is in.").wait_for(state="visible", timeout=30000); page.get_by_role("button", name="Add or rate a linked thing", exact=True).first.wait_for(state="visible", timeout=30000); print("thing_root_live_derived_five_step_flow_ok"); print("thing_root_linked_offer_visible_ok"); page.get_by_role("button", name="Done", exact=True).click(); page.wait_for_url(re.compile(r"/judge(?:$|[?#])"), timeout=30000)

    known=[m for m in console_errors if baseline in m and "script-src 'self'" in m]; unexpected=[m for m in console_errors if m not in known]
    print(f"known_live_baseline_csp_console_errors={len(known)}")
    if api_failures: raise RuntimeError("API responses >=400 during rating flow: "+" | ".join(api_failures[:5]))
    if request_failures: raise RuntimeError("Browser request failures: "+" | ".join(request_failures[:5]))
    if page_errors: raise RuntimeError("Browser page errors: "+" | ".join(page_errors[:5]))
    if unexpected: raise RuntimeError("Unexpected console errors: "+" | ".join(unexpected[:5]))
    browser.close()
print("rating_browser_requirements_ok"); print("unexpected_console_errors=0"); print("network_errors=0")
PY

if command -v docker >/dev/null 2>&1; then runtime=docker; elif command -v podman >/dev/null 2>&1; then runtime=podman; else echo "VERIFY BLOCKED: no container runtime" >&2; exit 78; fi
if [[ "$runtime" == podman ]]; then
  runtime_bin="$work/runtime-bin"; mkdir -p "$runtime_bin"; ln -s "$(command -v podman)" "$runtime_bin/docker"; export PATH="$runtime_bin:$PATH"
  ctx="$work/playwright-compat"; mkdir -p "$ctx"; printf 'FROM %s\nRUN python -m pip install --no-cache-dir playwright==1.57.0\n' "$PLAYWRIGHT_IMAGE" >"$ctx/Containerfile"; podman pull "$PLAYWRIGHT_IMAGE" >/dev/null; podman build --pull=missing -t localhost/rum-rating-playwright:1.57.0 "$ctx" >/dev/null; podman tag localhost/rum-rating-playwright:1.57.0 "$PLAYWRIGHT_IMAGE"
fi
printf 'RUM_RATING_CONTAINER_RUNTIME=%s\n' "$runtime"

unset TOKEN GH_TOKEN GHCR_TOKEN
register(){ local email="$1" user="$2" pass="$3" state="$4"; docker run --rm --ipc=host -v "$work:/work" -e RUM_BASE_URL="$BASE_URL" -e RUM_EMAIL="$email" -e RUM_USERNAME="$user" -e RUM_PASSWORD="$pass" -e RUM_STATE="/work/$state" "$PLAYWRIGHT_IMAGE" python /work/register.py; }
register "$rater_email" "$rater_username" "$rater_password" rater-state.json
register "$mate_email" "$mate_username" "$mate_password" mate-state.json

b64(){ printf '%s' "$1" | python3 -c 'import base64,sys; print(base64.b64encode(sys.stdin.buffer.read()).decode())'; }
rater_b64="$(b64 "$rater_email")"; mate_b64="$(b64 "$mate_email")"
PHP_BOOT='require "/var/www/html/vendor/autoload.php"; $app=require "/var/www/html/bootstrap/app.php"; $app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap(); eval($argv[1]);'
kctl -n "$DEV_NAMESPACE" exec "$api_pod" -c php-fpm -- php -r "$PHP_BOOT" "\$r=App\\Models\\User::where('email',base64_decode('${rater_b64}'))->firstOrFail(); \$m=App\\Models\\User::where('email',base64_decode('${mate_b64}'))->firstOrFail(); \$r->forceFill(['email_verified_at'=>now()])->save(); \$m->forceFill(['email_verified_at'=>now()])->save(); App\\Models\\MateRelationship::firstOrCreate(['requester_id'=>\$r->id,'addressee_id'=>\$m->id],['status'=>'accepted','accepted_at'=>now()]);" >/dev/null

docker run --rm --ipc=host -v "$work:/work" -e RUM_BASE_URL="$BASE_URL" -e RUM_MATE_USERNAME="$mate_username" -e RUM_LINKED_NAME="$linked_name" -e RUM_BASELINE_CSP="$LIVE_BASELINE_CSP_HASH" "$PLAYWRIGHT_IMAGE" python /work/exercise.py

printf 'RUM_RATING_CANDIDATE_SHA=%s\n' "$CANDIDATE_SHA"
printf 'RUM_RATING_REFERENCE_SHA=%s\n' "$LIVE_REFERENCE_SHA"
printf 'rating_rater_account=%s\n' "$rater_username"
printf 'rating_rater_email=%s\n' "$rater_email"
printf 'rating_mate_account=%s\n' "$mate_username"
printf 'rating_mate_email=%s\n' "$mate_email"
printf 'rating_linked_thing=%s\n' "$linked_name"
printf 'LIVE_RUNTIME_AFFECTED=NO\n'
printf 'RATE_ANYTHING_AFFECTED=NO\n'
printf 'RUM isolated DEV LIVE-derived rating flow visibly verified.\n'
