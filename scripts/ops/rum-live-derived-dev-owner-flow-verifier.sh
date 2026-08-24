#!/usr/bin/env bash
set -euo pipefail
umask 077

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <exact-rum-candidate-sha>" >&2
  exit 64
fi

CANDIDATE_SHA="$1"
[[ "$CANDIDATE_SHA" =~ ^[0-9a-f]{40}$ ]] || {
  echo "candidate SHA must be a full 40-character lowercase hexadecimal commit" >&2
  exit 64
}

REPOSITORY="DaisyCloverSoftware/rum"
CANDIDATE_BRANCH="sprint-0-rum-owner-rating-flow-20260823"
CANDIDATE_PR="153"
EXPECTED_CI_NAME="CI"
VERIFIER_PATH="scripts/ops/verify-rum-dev-owner-rating-flow.sh"
PLAYWRIGHT_IMAGE="mcr.microsoft.com/playwright/python:v1.57.0-noble"
LIVE_BASELINE_CSP_HASH="sha256-A0FJyCgxFUPhG7nac5LcQPwVRK5So9ZNz7x5ubsD9kU="

for command in gh git mktemp bash python3 grep; do
  command -v "$command" >/dev/null 2>&1 || {
    echo "ERROR: required command unavailable: $command" >&2
    exit 2
  }
done

TOKEN="${GH_TOKEN:-}"
if [[ -z "$TOKEN" ]]; then
  TOKEN="$(gh auth token 2>/dev/null || true)"
fi
[[ -n "$TOKEN" ]] || {
  echo "ERROR: no GitHub token is available to verify the exact candidate" >&2
  exit 2
}

branch_sha="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/${CANDIDATE_BRANCH}" --jq '.object.sha')"
[[ "$branch_sha" == "$CANDIDATE_SHA" ]] || {
  echo "VERIFY BLOCKED: requested SHA is not the exact current RUM owner-candidate branch head." >&2
  exit 78
}

pr_state="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/pulls/${CANDIDATE_PR}" --jq '[.state, (.draft|tostring), .head.sha, (.merged_at // "")] | @tsv')"
IFS=$'\t' read -r state draft pr_head merged_at <<<"$pr_state"
[[ "$state" == "open" && "$draft" == "true" && "$pr_head" == "$CANDIDATE_SHA" && -z "$merged_at" ]] || {
  echo "VERIFY BLOCKED: RUM PR #${CANDIDATE_PR} is not the expected open, draft, unmerged exact candidate." >&2
  exit 78
}

ci_successes="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/actions/runs?head_sha=${CANDIDATE_SHA}&event=pull_request&status=completed&per_page=100" --jq '[.workflow_runs[] | select(.name == "'"${EXPECTED_CI_NAME}"'" and .head_sha == "'"${CANDIDATE_SHA}"'" and .conclusion == "success")] | length')"
[[ "$ci_successes" =~ ^[0-9]+$ && "$ci_successes" -ge 1 ]] || {
  echo "VERIFY BLOCKED: no successful exact-head ${EXPECTED_CI_NAME} workflow run exists for ${CANDIDATE_SHA}." >&2
  exit 78
}

tmp_root="$(mktemp -d)"
cleanup() {
  rm -rf "$tmp_root"
}
trap cleanup EXIT HUP INT TERM

GH_TOKEN="$TOKEN" gh repo clone "$REPOSITORY" "$tmp_root/rum" -- --no-checkout --filter=blob:none >/dev/null
cd "$tmp_root/rum"
git checkout --detach "$CANDIDATE_SHA" >/dev/null
[[ "$(git rev-parse HEAD)" == "$CANDIDATE_SHA" ]] || {
  echo "VERIFY BLOCKED: disposable checkout is not the exact candidate SHA." >&2
  exit 78
}
[[ -f "$VERIFIER_PATH" && ! -L "$VERIFIER_PATH" ]] || {
  echo "VERIFY BLOCKED: exact candidate verifier is missing or not a regular file." >&2
  exit 78
}

# Keep the candidate verifier's isolated-DEV ingress guard, disposable account
# setup, mate-edge setup, and self-declaration persistence assertion. Replace
# only its stale browser exercise in a disposable copy so verification follows
# the rating flow actually restored from LIVE commit
# 8106675325eb8a516696bfb45cf817e97f03d7f5:
# Person -> Value -> Category -> Reason/Audience -> Review.
# The production API image excludes Laravel Tinker, so rewrite the candidate's
# two Tinker probes to equivalent bootstrapped php -r evaluation as before.
tinker_calls="$(grep -c 'php artisan tinker --execute=' "$VERIFIER_PATH" || true)"
[[ "$tinker_calls" == "2" ]] || {
  echo "VERIFY BLOCKED: expected exactly two candidate Tinker probe calls; found ${tinker_calls}." >&2
  exit 78
}
compat_verifier="$tmp_root/verify-rum-dev-live-derived-flow.sh"
python3 - "$VERIFIER_PATH" "$compat_verifier" "$LIVE_BASELINE_CSP_HASH" <<'PY'
from pathlib import Path
import sys

src = Path(sys.argv[1]).read_text()
baseline = sys.argv[3]
bootstrap = r'''PHP_EVAL_BOOTSTRAP='require "/var/www/html/vendor/autoload.php"; $app=require "/var/www/html/bootstrap/app.php"; $app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap(); eval($argv[1]);' '''.rstrip() + "\n"
needle = 'php artisan tinker --execute='
if src.count(needle) != 2:
    raise SystemExit('unexpected tinker call count')
lines = src.splitlines(keepends=True)
lines.insert(2 if len(lines) >= 2 else 0, bootstrap)
out = ''.join(lines).replace(needle, 'php -r "$PHP_EVAL_BOOTSTRAP" ')

old_linked_name = 'linked_name="RUM DEV linked check ${suffix}"'
new_linked_name = 'linked_name="Quillstone${suffix//-/}Zeta"'
if out.count(old_linked_name) != 1:
    raise SystemExit('unexpected linked-name fixture assignment')
out = out.replace(old_linked_name, new_linked_name)

start_marker = 'cat >"$work/exercise.py" <<\'PY\'\n'
docker_marker = 'docker run --rm --ipc=host \\\n'
start = out.find(start_marker)
if start < 0:
    raise SystemExit('browser exercise start marker missing')
body_start = start + len(start_marker)
end_marker = '\nPY\n\n' + docker_marker
end = out.find(end_marker, body_start)
if end < 0:
    raise SystemExit('browser exercise end marker missing')

exercise = r'''import os
import re
from playwright.sync_api import sync_playwright, TimeoutError as PlaywrightTimeoutError

base = os.environ["RUM_BASE_URL"]
mate_username = os.environ["RUM_MATE_USERNAME"]
linked_name = os.environ["RUM_LINKED_NAME"]
gamertag = os.environ["RUM_GAMERTAG"]
state = "/work/rater-state.json"
console_errors = []
page_errors = []
request_failures = []
api_failures = []
identity_used = None


def wait_step(page, number, name):
    header = page.locator(".step-header").filter(has_text=f"Step {number} of 5").first
    header.wait_for(state="visible", timeout=30000)
    header.get_by_text(name, exact=True).wait_for(state="visible", timeout=15000)


def assert_initial_category_shortlist(page):
    choices = page.locator("button.category-choice")
    choices.first.wait_for(state="visible", timeout=30000)
    count = choices.count()
    if count < 1 or count > 5:
        raise RuntimeError(f"LIVE-derived category shortlist expected 1..5 choices, found {count}.")
    print(f"category_shortlist_count={count}")
    print("category_shortlist_max_5_ok")


def run_live_rating(page, category_mode="first"):
    wait_step(page, 2, "Value")
    page.get_by_text("Loading rating options…", exact=True).wait_for(state="hidden", timeout=30000)
    page.get_by_role("radio", name="Thumbs up 5 of 5 — Exceptional").click()
    page.get_by_role("button", name="Continue", exact=True).click()

    wait_step(page, 3, "Category")
    page.get_by_role("heading", name="Pick a category").wait_for(state="visible", timeout=15000)
    assert_initial_category_shortlist(page)
    category_search = page.get_by_role("searchbox", name="Search categories")
    if category_mode == "search":
        category_search.fill("qual")
        page.get_by_text("Quality", exact=True).first.wait_for(state="visible", timeout=30000)
        page.locator("button.category-choice").filter(has_text="Quality").first.click()
        print("category_substring_search_ok")
    elif category_mode == "custom-affordance":
        custom = f"Flowcraft {linked_name[-8:]}"
        category_search.fill(custom)
        use_custom = page.locator("button.category-choice").filter(has_text=f"Use “{custom}”").first
        use_custom.wait_for(state="visible", timeout=30000)
        print("custom_category_affordance_ok")
        category_search.fill("")
        page.locator("button.category-choice").first.wait_for(state="visible", timeout=30000)
        page.locator("button.category-choice").first.click()
    else:
        page.locator("button.category-choice").first.click()
    page.get_by_role("button", name="Continue", exact=True).click()

    wait_step(page, 4, "Reason")
    page.get_by_role("heading", name="Add the reason").wait_for(state="visible", timeout=15000)
    page.get_by_label("Why this verdict?").fill("Specific isolated DEV verification of the LIVE-derived RUM rating progression.")
    public_radio = page.locator('input[name="audience"][value="public"]')
    if not public_radio.is_checked():
        public_radio.check()
    page.get_by_role("button", name="Continue", exact=True).click()

    wait_step(page, 5, "Review")
    page.get_by_role("heading", name="Check your rating").wait_for(state="visible", timeout=15000)
    page.get_by_text("Specific isolated DEV verification of the LIVE-derived RUM rating progression.", exact=False).first.wait_for(state="visible", timeout=15000)
    page.get_by_role("button", name="Submit rating", exact=True).click()


with sync_playwright() as p:
    browser = p.chromium.launch()
    context = browser.new_context(storage_state=state, viewport={"width": 1440, "height": 1200})
    page = context.new_page()
    page.on("console", lambda msg: console_errors.append(msg.text) if msg.type == "error" else None)
    page.on("pageerror", lambda err: page_errors.append(str(err)))
    page.on("requestfailed", lambda req: request_failures.append(f"{req.method} {req.url}"))
    page.on("response", lambda res: api_failures.append(f"{res.status} {res.url}") if res.status >= 400 and "/api/" in res.url else None)

    # 1) MATE/PERSON: LIVE Step 1 Person selection plus the shared Value ->
    # Category -> Reason/Audience -> Review progression. Successful mate ratings
    # finish exactly like LIVE and never expose linked-Thing continuation.
    page.goto(f"{base}/rate", wait_until="networkidle", timeout=60000)
    wait_step(page, 1, "Person")
    search = page.get_by_role("searchbox", name="Search RUM members and public identities")
    search.fill(mate_username)
    mate_button = page.locator("button.select-person").filter(has_text=mate_username).first
    mate_button.wait_for(state="visible", timeout=30000)
    mate_button.click()
    if mate_button.get_attribute("aria-pressed") != "true":
        raise RuntimeError("Mate was not visibly selected before Continue.")
    page.get_by_role("button", name="Continue", exact=True).click()
    run_live_rating(page, "first")
    page.wait_for_url(re.compile(r"/judge(?:$|[?#])"), timeout=30000)
    if page.get_by_role("button", name="Add or rate a linked thing", exact=True).count() != 0:
        raise RuntimeError("Mate rating incorrectly exposed linked-Thing continuation.")
    print("mate_live_derived_five_step_flow_ok")
    print("mate_linked_offer_absent_ok")

    # 2) PUBLIC FIGURE/PERSONA: select through the normal RUM People search,
    # then use the exact same shared LIVE-derived rating flow. The only root-
    # specific addition is the exact post-completion linked-Thing action.
    page.goto(f"{base}/rate", wait_until="networkidle", timeout=60000)
    wait_step(page, 1, "Person")
    search = page.get_by_role("searchbox", name="Search RUM members and public identities")
    for candidate in ("CJ Investigates", "Colin Furze"):
        search.fill(candidate)
        card = page.locator(".person-card").filter(has_text=candidate).first
        try:
            card.wait_for(state="visible", timeout=15000)
        except PlaywrightTimeoutError:
            continue
        card.get_by_role("button", name="Rate", exact=True).click()
        identity_used = candidate
        break
    if identity_used is None:
        raise RuntimeError("Neither CJ Investigates nor Colin Furze is available through isolated DEV public-identity search.")
    run_live_rating(page, "first")
    page.get_by_role("heading", name="Your verdict is in.").wait_for(state="visible", timeout=30000)
    add_linked = page.get_by_role("button", name="Add or rate a linked thing", exact=True).first
    add_linked.wait_for(state="visible", timeout=30000)
    print(f"public_identity={identity_used}")
    print("public_live_derived_five_step_flow_ok")
    print("public_linked_offer_visible_ok")

    # 3) LINKED THING: after the permitted root continuation, create a disposable
    # isolated-DEV Thing and prove it re-enters the same Value -> Category ->
    # Reason/Audience -> Review flow. Its own completion must finish like LIVE,
    # not offer another cascading continuation.
    add_linked.click()
    page.get_by_text("Search RUM before adding a missing linked thing", exact=True).wait_for(state="visible", timeout=15000)
    page.get_by_label("Search existing things before adding").fill(linked_name)
    page.get_by_role("button", name="Search RUM", exact=True).click()
    page.get_by_role("button", name="None of these — add a linked thing", exact=True).wait_for(state="visible", timeout=30000)
    page.get_by_role("button", name="None of these — add a linked thing", exact=True).click()
    page.get_by_label("Linked thing type").select_option("product")
    page.get_by_label("Linked thing description").fill("Disposable linked Thing created only in isolated DEV to verify the LIVE-derived rating flow.")
    page.get_by_role("button", name="Check and add linked thing", exact=True).click()
    run_live_rating(page, "search")
    page.wait_for_url(re.compile(r"/judge(?:$|[?#])"), timeout=30000)
    if page.get_by_role("button", name="Add or rate a linked thing", exact=True).count() != 0:
        raise RuntimeError("Linked Thing incorrectly offered another linked-Thing continuation.")
    print(f"linked_thing={linked_name}")
    print("linked_thing_live_derived_five_step_flow_ok")
    print("linked_thing_no_cascade_ok")

    # 4) STANDALONE THING ROOT: the newly published linked Thing must now be
    # discoverable from the normal RUM Rate Step 1 Thing search and enter the
    # same rating host. A root Thing gets the one permitted linked-Thing action.
    page.goto(f"{base}/rate", wait_until="networkidle", timeout=60000)
    wait_step(page, 1, "Person")
    search = page.get_by_role("searchbox", name="Search RUM members and public identities")
    search.fill(linked_name)
    thing_button = page.locator("button.select-person").filter(has_text=linked_name).first
    thing_button.wait_for(state="visible", timeout=30000)
    thing_button.click()
    run_live_rating(page, "custom-affordance")
    page.get_by_role("heading", name="Your verdict is in.").wait_for(state="visible", timeout=30000)
    page.get_by_role("button", name="Add or rate a linked thing", exact=True).first.wait_for(state="visible", timeout=30000)
    print("thing_root_live_derived_five_step_flow_ok")
    print("thing_root_linked_offer_visible_ok")
    page.get_by_role("button", name="Done", exact=True).click()
    page.wait_for_url(re.compile(r"/judge(?:$|[?#])"), timeout=30000)

    # 5) Existing Sprint-0 self-declaration requirement: a self-added gamer /
    # online identity is immediately mine/self-declared with no verification
    # prompt, and the candidate verifier's DB persistence check follows below.
    page.goto(f"{base}/me", wait_until="networkidle", timeout=60000)
    page.get_by_role("button", name="Add or claim my identity", exact=True).wait_for(state="visible", timeout=30000)
    page.get_by_role("button", name="Add or claim my identity", exact=True).click()
    page.get_by_role("heading", name="My gamer & online identities").wait_for(state="visible", timeout=30000)
    page.get_by_label("Full gamertag").fill(gamertag)
    page.get_by_role("button", name="Check RUM", exact=True).click()
    page.get_by_role("heading", name="Not on RUM yet").wait_for(state="visible", timeout=30000)
    page.get_by_role("button", name="Add this identity as mine", exact=True).click()
    page.get_by_text("This identity is yours by self-declaration. No verification is required unless ownership is challenged later.", exact=True).wait_for(state="visible", timeout=30000)

    page.goto(f"{base}/me", wait_until="networkidle", timeout=60000)
    row = page.locator(".my-identity-row").filter(has_text=gamertag).first
    row.wait_for(state="visible", timeout=30000)
    row.get_by_text("Mine · self-declared", exact=True).wait_for(state="visible", timeout=15000)
    row.get_by_text("Ownership: self-declared. No verification is required unless ownership is challenged.", exact=True).wait_for(state="visible", timeout=15000)
    if row.get_by_role("button", name="Verify identity", exact=True).count() != 0:
        raise RuntimeError("Self-added identity incorrectly exposed Verify identity.")
    lower_text = row.inner_text().lower()
    if "verification pending" in lower_text or "not verified" in lower_text or "unverified" in lower_text:
        raise RuntimeError("Self-added identity is visibly presented as pending/unverified.")
    print(f"self_declared_identity={gamertag}")
    print("self_declared_identity_visible_ok")
    print("self_declared_verification_prompt_absent_ok")

    known_live_baseline_csp = [message for message in console_errors if "__BASELINE_HASH__" in message and "script-src 'self'" in message]
    unexpected_console_errors = [message for message in console_errors if message not in known_live_baseline_csp]
    print(f"known_live_baseline_csp_console_errors={len(known_live_baseline_csp)}")
    if api_failures:
        raise RuntimeError("API responses >=400 during verified flow: " + " | ".join(api_failures[:5]))
    if request_failures:
        raise RuntimeError("Browser request failures: " + " | ".join(request_failures[:5]))
    if page_errors:
        raise RuntimeError("Browser page errors: " + " | ".join(page_errors[:5]))
    if unexpected_console_errors:
        raise RuntimeError("Browser console errors excluding proven LIVE baseline CSP: " + " | ".join(unexpected_console_errors[:5]))

    browser.close()

print("browser_owner_requirements_ok")
print("unexpected_console_errors=0")
print("network_errors=0")
'''.replace('__BASELINE_HASH__', baseline)

out = out[:body_start] + exercise + out[end:]
Path(sys.argv[2]).write_text(out)
PY
chmod 700 "$compat_verifier"

[[ "$(grep -c 'php artisan tinker --execute=' "$compat_verifier" || true)" == "0" ]] || {
  echo "VERIFY BLOCKED: compatibility verifier still contains Tinker calls." >&2
  exit 78
}
[[ "$(grep -c 'php -r \"\$PHP_EVAL_BOOTSTRAP\"' "$compat_verifier" || true)" == "2" ]] || {
  echo "VERIFY BLOCKED: compatibility verifier did not contain exactly two Laravel bootstrap probes." >&2
  exit 78
}
[[ "$(grep -c 'mate_live_derived_five_step_flow_ok' "$compat_verifier" || true)" == "1" ]] || {
  echo "VERIFY BLOCKED: LIVE-derived mate browser exercise was not installed." >&2
  exit 78
}
[[ "$(grep -c 'thing_root_live_derived_five_step_flow_ok' "$compat_verifier" || true)" == "1" ]] || {
  echo "VERIFY BLOCKED: standalone Thing root browser exercise was not installed." >&2
  exit 78
}
[[ "$(grep -c "$LIVE_BASELINE_CSP_HASH" "$compat_verifier" || true)" == "1" ]] || {
  echo "VERIFY BLOCKED: compatibility verifier did not contain exactly one proven LIVE CSP baseline hash." >&2
  exit 78
}

printf 'RUM_OWNER_FLOW_TINKER_COMPAT=2_PROBES_REWRITTEN\n'
printf 'RUM_OWNER_FLOW_BROWSER_COMPAT=LIVE_DERIVED_FIVE_STEP\n'
printf 'RUM_OWNER_FLOW_THING_ROOT_COVERAGE=ENABLED\n'
printf 'RUM_OWNER_FLOW_LIVE_CSP_BASELINE=%s\n' "$LIVE_BASELINE_CSP_HASH"

# The cluster-control host intentionally has no Docker daemon. Podman is
# Docker-CLI compatible for the candidate verifier's bounded docker-run use.
# Derive a local Playwright Python-compatible image from the exact base if
# Podman is the available runtime.
if command -v docker >/dev/null 2>&1; then
  printf 'RUM_OWNER_FLOW_CONTAINER_RUNTIME=docker\n'
elif command -v podman >/dev/null 2>&1; then
  runtime_bin="$tmp_root/runtime-bin"
  mkdir -p "$runtime_bin"
  ln -s "$(command -v podman)" "$runtime_bin/docker"
  export PATH="$runtime_bin:$PATH"

  playwright_ctx="$tmp_root/playwright-compat"
  mkdir -p "$playwright_ctx"
  cat >"$playwright_ctx/Containerfile" <<EOF
FROM ${PLAYWRIGHT_IMAGE}
RUN python -m pip install --no-cache-dir playwright==1.57.0
EOF
  podman pull "$PLAYWRIGHT_IMAGE" >/dev/null
  podman build --pull=missing -t localhost/rum-playwright-python:1.57.0 "$playwright_ctx" >/dev/null
  podman run --rm --entrypoint python localhost/rum-playwright-python:1.57.0 -c 'import playwright; from playwright.sync_api import sync_playwright; print("playwright_python_ready")'
  podman tag localhost/rum-playwright-python:1.57.0 "$PLAYWRIGHT_IMAGE"
  printf 'RUM_OWNER_FLOW_CONTAINER_RUNTIME=podman\n'
  printf 'RUM_OWNER_FLOW_PLAYWRIGHT_PYTHON=ready\n'
else
  echo "VERIFY BLOCKED: neither Docker nor Podman is available for the isolated DEV browser verifier." >&2
  exit 78
fi

unset TOKEN GH_TOKEN GHCR_TOKEN

printf 'RUM_OWNER_FLOW_CANDIDATE_SHA=%s\n' "$CANDIDATE_SHA"
printf 'RUM_OWNER_FLOW_REFERENCE_SHA=8106675325eb8a516696bfb45cf817e97f03d7f5\n'
printf 'RUM_OWNER_FLOW_EXECUTED_VERIFIER=%s\n' "$compat_verifier"

bash "$compat_verifier"
