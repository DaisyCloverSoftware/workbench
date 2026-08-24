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

work="$(mktemp -d)"; trap 'rm -rf "$work"' EXIT HUP INT TERM
suffix="$(date +%s)-$(python3 -c 'import secrets; print(secrets.token_hex(2))')"
username="rumcj${suffix//-/}"; username="${username:0:28}"
email="${username}@example.com"
password="$(python3 -c 'import secrets; print(secrets.token_urlsafe(24))')"

cat >"$work/verify.py" <<'PY'
import json, os, re
from playwright.sync_api import sync_playwright

base=os.environ["RUM_BASE_URL"]
email=os.environ["RUM_EMAIL"]
username=os.environ["RUM_USERNAME"]
password=os.environ["RUM_PASSWORD"]
baseline=os.environ["RUM_BASELINE_CSP"]
console_errors=[]; page_errors=[]; request_failures=[]; api_failures=[]
rating_id=None

def wait_step(page,n,name):
    header=page.locator(".step-header").filter(has_text=f"Step {n} of 5").first
    header.wait_for(state="visible", timeout=30000)
    header.get_by_text(name, exact=True).wait_for(state="visible", timeout=15000)

def api_get(page,path):
    result=page.evaluate("""async (path) => {
      const response = await fetch(path, {credentials:'same-origin', headers:{Accept:'application/json'}});
      return {status: response.status, body: await response.json().catch(() => ({}))};
    }""", path)
    if result["status"] >= 400:
        raise RuntimeError(f"API GET {path} returned {result['status']}: {result['body']}")
    return result["body"]

with sync_playwright() as p:
    browser=p.chromium.launch()
    context=browser.new_context(viewport={"width":1440,"height":1200})
    page=context.new_page()
    page.on("console", lambda msg: console_errors.append(msg.text) if msg.type=="error" else None)
    page.on("pageerror", lambda err: page_errors.append(str(err)))
    page.on("requestfailed", lambda req: request_failures.append(f"{req.method} {req.url} {req.failure}"))
    page.on("response", lambda res: api_failures.append(f"{res.status} {res.url}") if res.status>=400 and "/api/" in res.url else None)

    page.goto(base, wait_until="networkidle", timeout=60000)
    page.get_by_role("button", name="Create an account").click()
    page.get_by_label("Email").fill(email)
    page.get_by_label("Username").fill(username)
    page.get_by_label("Password", exact=True).fill(password)
    page.get_by_label("Confirm password").fill(password)
    page.get_by_role("checkbox", name="I accept the Terms and Community Rules.").check()
    page.get_by_role("checkbox", name="I have read the Privacy Notice.").check()
    page.get_by_role("button", name="Create account").click()
    page.wait_for_url("**/me", timeout=30000)
    print(f"cj_surface_registration_ok={username}")

    page.goto(f"{base}/rate", wait_until="networkidle", timeout=60000)
    wait_step(page,1,"Person")
    search=page.get_by_role("searchbox", name="Search RUM members and public identities")
    search.fill("CJ Investigates")
    card=page.locator(".person-card").filter(has_text="CJ Investigates").first
    card.wait_for(state="visible", timeout=30000)
    card.get_by_role("button", name="Rate", exact=True).click()

    wait_step(page,2,"Value")
    page.get_by_text("Loading rating options…", exact=True).wait_for(state="hidden", timeout=30000)
    page.get_by_role("radio", name="Thumbs up 5 of 5 — Exceptional").click()
    page.get_by_role("button", name="Continue", exact=True).click()

    wait_step(page,3,"Category")
    page.locator("button.category-choice").first.wait_for(state="visible", timeout=30000)
    page.locator("button.category-choice").first.click()
    page.get_by_role("button", name="Continue", exact=True).click()

    wait_step(page,4,"Reason")
    page.get_by_label("Why this verdict?").fill("CJ surface regression verification in isolated DEV for Judge and Given history.")
    public=page.locator('input[name="audience"][value="public"]')
    if not public.is_checked(): public.check()
    page.get_by_role("button", name="Continue", exact=True).click()

    wait_step(page,5,"Review")
    with page.expect_response(lambda r: "/api/v1/rate-anything/rating" in r.url and r.request.method == "POST", timeout=30000) as submitted:
        page.get_by_role("button", name="Submit rating", exact=True).click()
    response=submitted.value
    if response.status not in (200,201):
        raise RuntimeError(f"Rating submission returned {response.status}")
    payload=response.json()
    rating_id=str(payload.get("data",{}).get("id", ""))
    if not rating_id:
        raise RuntimeError("Rating submission did not return canonical event id")
    page.get_by_role("heading", name="Your verdict is in.").wait_for(state="visible", timeout=30000)
    print(f"cj_rating_event_id={rating_id}")
    print("cj_rating_submission_ok")

    judge=api_get(page,"/api/v1/judge?queue=needs&scope=all")
    judge_item=next((item for item in judge.get("data",[]) if str(item.get("id")) == rating_id), None)
    if judge_item is None:
        raise RuntimeError("Canonical CJ rating missing from Judge API")
    if judge_item.get("target",{}).get("displayName") != "CJ Investigates":
        raise RuntimeError(f"Judge target mismatch: {judge_item.get('target')}")
    if judge_item.get("state") != "needs_verdict":
        raise RuntimeError(f"Judge state mismatch: {judge_item.get('state')}")
    if judge_item.get("canVote") is not False:
        raise RuntimeError("Rater must see own CJ rating as read-only in Judge")
    print("cj_judge_api_same_event_ok")

    given=api_get(page,"/api/v1/me/ratings?direction=given")
    given_item=next((item for item in given.get("data",[]) if str(item.get("id")) == rating_id), None)
    if given_item is None:
        raise RuntimeError("Canonical CJ rating missing from Given API")
    if given_item.get("person",{}).get("displayName") != "CJ Investigates":
        raise RuntimeError(f"Given target mismatch: {given_item.get('person')}")
    if given_item.get("judge",{}).get("state") != "needs_verdict":
        raise RuntimeError(f"Given Judge state mismatch: {given_item.get('judge')}")
    print("cj_given_api_same_event_ok")

    page.goto(f"{base}/judge", wait_until="networkidle", timeout=60000)
    page.get_by_text("CJ Investigates", exact=True).first.wait_for(state="visible", timeout=30000)
    page.get_by_text("Read only", exact=True).first.wait_for(state="visible", timeout=30000)
    print("cj_judge_ui_visible_ok")

    page.goto(f"{base}/me?tab=given", wait_until="networkidle", timeout=60000)
    page.get_by_role("heading", name="Ratings you gave").wait_for(state="visible", timeout=30000)
    page.get_by_text("For CJ Investigates", exact=True).first.wait_for(state="visible", timeout=30000)
    print("cj_given_ui_visible_ok")

    known=[m for m in console_errors if baseline in m and "script-src 'self'" in m]
    unexpected=[m for m in console_errors if m not in known]
    benign=[m for m in request_failures if "ERR_ABORTED" in m]
    real=[m for m in request_failures if "ERR_ABORTED" not in m]
    print(f"known_live_baseline_csp_console_errors={len(known)}")
    print(f"benign_navigation_request_aborts={len(benign)}")
    if api_failures:
        raise RuntimeError("API responses >=400: "+" | ".join(api_failures[:5]))
    if real:
        raise RuntimeError("Real browser request failures: "+" | ".join(real[:5]))
    if page_errors:
        raise RuntimeError("Browser page errors: "+" | ".join(page_errors[:5]))
    if unexpected:
        raise RuntimeError("Unexpected console errors: "+" | ".join(unexpected[:5]))
    browser.close()

print("cj_judge_given_browser_requirements_ok")
print("unexpected_console_errors=0")
print("network_errors=0")
PY

if command -v docker >/dev/null 2>&1; then runtime=docker; elif command -v podman >/dev/null 2>&1; then runtime=podman; else echo "VERIFY BLOCKED: no container runtime" >&2; exit 78; fi
if [[ "$runtime" == podman ]]; then
  runtime_bin="$work/runtime-bin"; mkdir -p "$runtime_bin"; ln -s "$(command -v podman)" "$runtime_bin/docker"; export PATH="$runtime_bin:$PATH"
  ctx="$work/playwright-compat"; mkdir -p "$ctx"; printf 'FROM %s\nRUN python -m pip install --no-cache-dir playwright==1.57.0\n' "$PLAYWRIGHT_IMAGE" >"$ctx/Containerfile"
  podman pull "$PLAYWRIGHT_IMAGE" >/dev/null
  podman build --pull=missing -t localhost/rum-cj-playwright:1.57.0 "$ctx" >/dev/null
  podman tag localhost/rum-cj-playwright:1.57.0 "$PLAYWRIGHT_IMAGE"
fi
printf 'RUM_CJ_SURFACE_CONTAINER_RUNTIME=%s\n' "$runtime"

unset TOKEN GH_TOKEN GHCR_TOKEN
"$runtime" run --rm --network host \
  -v "$work:/work:Z" \
  -e RUM_BASE_URL="$BASE_URL" \
  -e RUM_EMAIL="$email" \
  -e RUM_USERNAME="$username" \
  -e RUM_PASSWORD="$password" \
  -e RUM_BASELINE_CSP="$LIVE_BASELINE_CSP_HASH" \
  "$PLAYWRIGHT_IMAGE" python /work/verify.py

printf 'RUM_CJ_SURFACE_CANDIDATE_SHA=%s\n' "$CANDIDATE_SHA"
printf 'RUM_CJ_SURFACE_NAMESPACE=%s\n' "$DEV_NAMESPACE"
printf 'LIVE_RUNTIME_AFFECTED=NO\n'
printf 'RATE_ANYTHING_AFFECTED=NO\n'
