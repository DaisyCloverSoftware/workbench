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
[[ "$state" == "open" && "$draft" == "true" && "$pr_head" == "$CANDIDATE_SHA" && -z "$merged_at" ]] || {
  echo "VERIFY BLOCKED: PR #${CANDIDATE_PR} is not open/draft/unmerged at exact head" >&2
  exit 78
}
ci_successes="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/actions/runs?head_sha=${CANDIDATE_SHA}&event=pull_request&status=completed&per_page=100" --jq '[.workflow_runs[] | select(.name == "'"${EXPECTED_CI_NAME}"'" and .head_sha == "'"${CANDIDATE_SHA}"'" and .conclusion == "success")] | length')"
[[ "$ci_successes" =~ ^[0-9]+$ && "$ci_successes" -ge 1 ]] || { echo "VERIFY BLOCKED: exact-head CI success missing" >&2; exit 78; }

kctl() {
  if command -v k3s >/dev/null 2>&1; then
    sudo k3s kubectl "$@"
  else
    kubectl "$@"
  fi
}

dev_hosts="$(kctl -n "$DEV_NAMESPACE" get ingress rum -o jsonpath='{range .spec.rules[*]}{.host}{"\n"}{end}' 2>/dev/null || true)"
[[ "$dev_hosts" == "$DEV_HOST" ]] || { echo "VERIFY BLOCKED: isolated DEV ingress is not exactly ${DEV_HOST}" >&2; exit 78; }
if grep -Fqx "$LIVE_HOST" <<<"$dev_hosts"; then
  echo "VERIFY BLOCKED: isolated DEV ingress contains LIVE host" >&2
  exit 78
fi
api_pod="$(kctl -n "$DEV_NAMESPACE" get pods -l 'app.kubernetes.io/instance=rum,app.kubernetes.io/component=api' --field-selector=status.phase=Running -o jsonpath='{.items[0].metadata.name}')"
[[ -n "$api_pod" ]] || { echo "VERIFY BLOCKED: isolated DEV API pod unavailable" >&2; exit 78; }

work="$(mktemp -d)"
cleanup(){ rm -rf "$work"; }
trap cleanup EXIT HUP INT TERM
suffix="$(date +%s)-$(python3 -c 'import secrets; print(secrets.token_hex(2))')"
username="rumself${suffix//-/}"
username="${username:0:28}"
email="${username}@example.com"
password="$(python3 -c 'import secrets; print(secrets.token_urlsafe(24))')"
gamertag="RUMSelfOnly${suffix//-/}"
gamertag="${gamertag:0:30}"
state_file="$work/self-state.json"

cat >"$work/register.py" <<'PY'
import os
from playwright.sync_api import sync_playwright
base=os.environ["RUM_BASE_URL"]
email=os.environ["RUM_EMAIL"]
username=os.environ["RUM_USERNAME"]
password=os.environ["RUM_PASSWORD"]
state=os.environ["RUM_STATE_FILE"]
with sync_playwright() as p:
    browser=p.chromium.launch()
    context=browser.new_context(viewport={"width":1440,"height":1000})
    page=context.new_page()
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
    context.storage_state(path=state)
    browser.close()
print(f"self_registration_session_ok={username}")
PY

cat >"$work/exercise.py" <<'PY'
import os
from playwright.sync_api import sync_playwright
base=os.environ["RUM_BASE_URL"]
gamertag=os.environ["RUM_GAMERTAG"]
state="/work/self-state.json"
baseline=os.environ["RUM_BASELINE_CSP"]
console_errors=[]
page_errors=[]
request_failures=[]
api_failures=[]
with sync_playwright() as p:
    browser=p.chromium.launch()
    context=browser.new_context(storage_state=state, viewport={"width":1440,"height":1200})
    page=context.new_page()
    page.on("console", lambda msg: console_errors.append(msg.text) if msg.type == "error" else None)
    page.on("pageerror", lambda err: page_errors.append(str(err)))
    page.on("requestfailed", lambda req: request_failures.append(f"{req.method} {req.url}"))
    page.on("response", lambda res: api_failures.append(f"{res.status} {res.url}") if res.status >= 400 and "/api/" in res.url else None)

    page.goto(f"{base}/me", wait_until="networkidle", timeout=60000)
    page.get_by_role("button", name="Add or claim my identity", exact=True).wait_for(state="visible", timeout=30000)
    page.get_by_role("button", name="Add or claim my identity", exact=True).click()
    page.get_by_role("heading", name="My gamer & online identities").wait_for(state="visible", timeout=30000)
    page.get_by_label("Full gamertag").fill(gamertag)
    page.get_by_role("button", name="Check RUM", exact=True).click()
    page.get_by_role("heading", name="Not on RUM yet").wait_for(state="visible", timeout=30000)
    page.get_by_role("button", name="Add this identity as mine", exact=True).click()
    page.get_by_text("This identity is yours by self-declaration. No verification is required unless ownership is challenged later.", exact=True).wait_for(state="visible", timeout=30000)
    print("self_declared_create_visible_ok")

    page.goto(f"{base}/me", wait_until="networkidle", timeout=60000)
    loading=page.get_by_text("Loading your identities…", exact=True)
    if loading.count():
        loading.first.wait_for(state="hidden", timeout=30000)
    row=page.locator(".my-identity-row").filter(has_text=gamertag).first
    row.wait_for(state="visible", timeout=30000)
    row.get_by_text("Mine · self-declared", exact=True).wait_for(state="visible", timeout=15000)
    row.get_by_text("Ownership: self-declared. No verification is required unless ownership is challenged.", exact=True).wait_for(state="visible", timeout=15000)
    if row.get_by_role("button", name="Verify identity", exact=True).count() != 0:
        raise RuntimeError("Self-declared identity exposed Verify identity.")
    lower=row.inner_text().lower()
    if "verification pending" in lower or "not verified" in lower or "unverified" in lower:
        raise RuntimeError("Self-declared identity is visibly presented as pending/unverified.")
    print("self_declared_profile_row_visible_ok")
    print("self_declared_verification_prompt_absent_ok")

    known=[message for message in console_errors if baseline in message and "script-src 'self'" in message]
    unexpected=[message for message in console_errors if message not in known]
    print(f"known_live_baseline_csp_console_errors={len(known)}")
    if api_failures:
        raise RuntimeError("API responses >=400 during self-declaration flow: "+" | ".join(api_failures[:5]))
    if request_failures:
        raise RuntimeError("Browser request failures: "+" | ".join(request_failures[:5]))
    if page_errors:
        raise RuntimeError("Browser page errors: "+" | ".join(page_errors[:5]))
    if unexpected:
        raise RuntimeError("Unexpected console errors: "+" | ".join(unexpected[:5]))
    browser.close()
print("self_declared_browser_requirements_ok")
print("unexpected_console_errors=0")
print("network_errors=0")
PY

if command -v docker >/dev/null 2>&1; then
  printf 'RUM_SELF_DECLARATION_CONTAINER_RUNTIME=docker\n'
elif command -v podman >/dev/null 2>&1; then
  runtime_bin="$work/runtime-bin"
  mkdir -p "$runtime_bin"
  ln -s "$(command -v podman)" "$runtime_bin/docker"
  export PATH="$runtime_bin:$PATH"
  ctx="$work/playwright-compat"
  mkdir -p "$ctx"
  cat >"$ctx/Containerfile" <<EOF
FROM ${PLAYWRIGHT_IMAGE}
RUN python -m pip install --no-cache-dir playwright==1.57.0
EOF
  podman pull "$PLAYWRIGHT_IMAGE" >/dev/null
  podman build --pull=missing -t localhost/rum-self-playwright:1.57.0 "$ctx" >/dev/null
  podman tag localhost/rum-self-playwright:1.57.0 "$PLAYWRIGHT_IMAGE"
  printf 'RUM_SELF_DECLARATION_CONTAINER_RUNTIME=podman\n'
else
  echo "VERIFY BLOCKED: neither Docker nor Podman available" >&2
  exit 78
fi

unset TOKEN GH_TOKEN GHCR_TOKEN

docker run --rm --ipc=host -v "$work:/work" \
  -e RUM_BASE_URL="$BASE_URL" -e RUM_EMAIL="$email" -e RUM_USERNAME="$username" -e RUM_PASSWORD="$password" -e RUM_STATE_FILE="/work/self-state.json" \
  "$PLAYWRIGHT_IMAGE" python /work/register.py
[[ -s "$state_file" ]] || { echo "VERIFY FAILED: self account browser state missing" >&2; exit 1; }
chmod 600 "$state_file"

email_b64="$(printf '%s' "$email" | python3 -c 'import base64,sys; print(base64.b64encode(sys.stdin.buffer.read()).decode())')"
PHP_BOOT='require "/var/www/html/vendor/autoload.php"; $app=require "/var/www/html/bootstrap/app.php"; $app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap(); eval($argv[1]);'
kctl -n "$DEV_NAMESPACE" exec "$api_pod" -c php-fpm -- php -r "$PHP_BOOT" "\$u=App\\Models\\User::where('email',base64_decode('${email_b64}'))->firstOrFail(); \$u->forceFill(['email_verified_at'=>now()])->save();" >/dev/null

docker run --rm --ipc=host -v "$work:/work" \
  -e RUM_BASE_URL="$BASE_URL" -e RUM_GAMERTAG="$gamertag" -e RUM_BASELINE_CSP="$LIVE_BASELINE_CSP_HASH" \
  "$PLAYWRIGHT_IMAGE" python /work/exercise.py

gamertag_b64="$(printf '%s' "$gamertag" | python3 -c 'import base64,sys; print(base64.b64encode(sys.stdin.buffer.read()).decode())')"
probe="$(kctl -n "$DEV_NAMESPACE" exec "$api_pod" -c php-fpm -- php -r "$PHP_BOOT" "\$u=App\\Models\\User::where('email',base64_decode('${email_b64}'))->firstOrFail(); \$e=App\\Models\\Entity::where('canonical_name',base64_decode('${gamertag_b64}'))->firstOrFail(); \$c=App\\Models\\EntityClaim::where('entity_id',\$e->id)->where('claimant_user_id',\$u->id)->firstOrFail(); echo \$c->status.' '.\$c->verification_state.' '.App\\Models\\EntityClaimVerification::where('entity_claim_id',\$c->id)->count();" 2>/dev/null | tail -n 1)"
read -r claim_status verification_state verification_count <<<"$probe"
[[ "$claim_status" == "approved" && "$verification_state" == "self_declared" && "$verification_count" == "0" ]] || {
  echo "VERIFY FAILED: self-declaration persistence mismatch" >&2
  exit 1
}

printf 'RUM_SELF_DECLARATION_CANDIDATE_SHA=%s\n' "$CANDIDATE_SHA"
printf 'self_declared_account=%s\n' "$username"
printf 'self_declared_email=%s\n' "$email"
printf 'self_declared_identity=%s\n' "$gamertag"
printf 'self_declared_persistence_ok\n'
printf 'LIVE_RUNTIME_AFFECTED=NO\n'
printf 'RATE_ANYTHING_AFFECTED=NO\n'
printf 'RUM isolated DEV self-declaration visibly verified.\n'
