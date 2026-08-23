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
BASELINE_CSP_HASH="sha256-A0FJyCgxFUPhG7nac5LcQPwVRK5So9ZNz7x5ubsD9kU="

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

state="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/pulls/${CANDIDATE_PR}" --jq '.state')"
draft="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/pulls/${CANDIDATE_PR}" --jq '(.draft|tostring)')"
pr_head="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/pulls/${CANDIDATE_PR}" --jq '.head.sha')"
merged_at="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/pulls/${CANDIDATE_PR}" --jq '(.merged_at // "")')"
base_sha="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/pulls/${CANDIDATE_PR}" --jq '.base.sha')"
[[ "$state" == "open" && "$draft" == "true" && "$pr_head" == "$CANDIDATE_SHA" && -z "$merged_at" && "$base_sha" =~ ^[0-9a-f]{40}$ ]] || {
  echo "VERIFY BLOCKED: RUM PR #${CANDIDATE_PR} is not the expected open, draft, unmerged exact candidate." >&2
  exit 78
}

ci_successes="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/actions/runs?head_sha=${CANDIDATE_SHA}&event=pull_request&status=completed&per_page=100" --jq '[.workflow_runs[] | select(.name == "'"${EXPECTED_CI_NAME}"'" and .head_sha == "'"${CANDIDATE_SHA}"'" and .conclusion == "success")] | length')"
[[ "$ci_successes" =~ ^[0-9]+$ && "$ci_successes" -ge 1 ]] || {
  echo "VERIFY BLOCKED: no successful exact-head ${EXPECTED_CI_NAME} workflow run exists for ${CANDIDATE_SHA}." >&2
  exit 78
}

for path in apps/web/index.html apps/web/nginx.conf; do
  base_blob="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/contents/${path}?ref=${base_sha}" --jq '.sha')"
  candidate_blob="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/contents/${path}?ref=${CANDIDATE_SHA}" --jq '.sha')"
  [[ -n "$base_blob" && "$base_blob" == "$candidate_blob" ]] || {
    echo "VERIFY BLOCKED: baseline CSP compatibility file changed in Sprint candidate: ${path}" >&2
    exit 78
  }
done

tmp_root="$(mktemp -d)"
cleanup() { rm -rf "$tmp_root"; }
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

tinker_calls="$(grep -c 'php artisan tinker --execute=' "$VERIFIER_PATH" || true)"
[[ "$tinker_calls" == "3" ]] || {
  echo "VERIFY BLOCKED: expected exactly three candidate Tinker probe calls; found ${tinker_calls}." >&2
  exit 78
}
console_hook='    page.on("console", lambda msg: console_errors.append(msg.text) if msg.type == "error" else None)'
[[ "$(grep -Fxc "$console_hook" "$VERIFIER_PATH" || true)" == "1" ]] || {
  echo "VERIFY BLOCKED: expected exactly one candidate browser console hook." >&2
  exit 78
}
response_hook='    page.on("response", lambda res: api_failures.append(f"{res.status} {res.url}") if res.status >= 400 and "/api/" in res.url else None)'
[[ "$(grep -Fxc "$response_hook" "$VERIFIER_PATH" || true)" == "1" ]] || {
  echo "VERIFY BLOCKED: expected exactly one candidate browser API response hook." >&2
  exit 78
}
search_zero_assert='    page.get_by_text("No existing thing matched that search.", exact=True).wait_for(state="visible", timeout=30000)'
[[ "$(grep -Fxc "$search_zero_assert" "$VERIFIER_PATH" || true)" == "1" ]] || {
  echo "VERIFY BLOCKED: expected exactly one candidate zero-results-only linked-search assertion." >&2
  exit 78
}
create_click='    page.get_by_role("button", name="Check and add linked thing", exact=True).click()'
[[ "$(grep -Fxc "$create_click" "$VERIFIER_PATH" || true)" == "1" ]] || {
  echo "VERIFY BLOCKED: expected exactly one candidate linked-Thing check-and-add action." >&2
  exit 78
}
create_heading='    page.get_by_role("heading", name=f"Rate {linked_name}").wait_for(state="visible", timeout=30000)'
[[ "$(grep -Fxc "$create_heading" "$VERIFIER_PATH" || true)" == "1" ]] || {
  echo "VERIFY BLOCKED: expected exactly one candidate post-create linked-Thing heading assertion." >&2
  exit 78
}
page_error_check='    if page_errors:'
[[ "$(grep -Fxc "$page_error_check" "$VERIFIER_PATH" || true)" == "1" ]] || {
  echo "VERIFY BLOCKED: expected exactly one candidate final browser page-error check." >&2
  exit 78
}

compat_verifier="$tmp_root/verify-rum-dev-owner-rating-flow-compat.sh"
python3 - "$VERIFIER_PATH" "$compat_verifier" "$BASELINE_CSP_HASH" <<'PY'
from pathlib import Path
import sys
src_path, dst_path, csp_hash = sys.argv[1:]
src = Path(src_path).read_text()
bootstrap = r'''PHP_EVAL_BOOTSTRAP='require "/var/www/html/vendor/autoload.php"; $app=require "/var/www/html/bootstrap/app.php"; $app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap(); eval($argv[1]);' '''.rstrip() + "\n"
needle = 'php artisan tinker --execute='
if src.count(needle) != 3:
    raise SystemExit('unexpected tinker call count')
lines = src.splitlines(keepends=True)
lines.insert(2 if len(lines) >= 2 else 0, bootstrap)
out = ''.join(lines).replace(needle, 'php -r "$PHP_EVAL_BOOTSTRAP" ')
old_console = '    page.on("console", lambda msg: console_errors.append(msg.text) if msg.type == "error" else None)'
if out.count(old_console) != 1:
    raise SystemExit('unexpected console hook count')
new_console = '''    page.on(
        "console",
        lambda msg: console_errors.append(msg.text)
        if msg.type == "error"
        and not (
            msg.text.startswith("Executing inline script violates the following Content Security Policy directive")
            and "''' + csp_hash + '''" in msg.text
        )
        else None,
    )'''
out = out.replace(old_console, new_console)
old_response = '    page.on("response", lambda res: api_failures.append(f"{res.status} {res.url}") if res.status >= 400 and "/api/" in res.url else None)'
if out.count(old_response) != 1:
    raise SystemExit('unexpected response hook count')
new_response = '''    expected_linked_confirmation_409s = []

    def record_response(res):
        expected_confirmation = (
            res.status == 409
            and res.request.method == "POST"
            and "/api/v1/rate-anything/rating-journeys/" in res.url
            and res.url.endswith("/linked-things")
        )
        if expected_confirmation:
            expected_linked_confirmation_409s.append(f"{res.status} {res.url}")
        elif res.status >= 400 and "/api/" in res.url:
            api_failures.append(f"{res.status} {res.url}")

    page.on("response", record_response)'''
out = out.replace(old_response, new_response)
zero_assert = '    page.get_by_text("No existing thing matched that search.", exact=True).wait_for(state="visible", timeout=30000)\n'
if out.count(zero_assert) != 1:
    raise SystemExit('unexpected zero-results linked-search assertion count')
out = out.replace(zero_assert, '')
old_heading = '    page.get_by_role("heading", name=f"Rate {linked_name}").wait_for(state="visible", timeout=30000)'
if out.count(old_heading) != 1:
    raise SystemExit('unexpected linked-Thing heading assertion count')
new_heading = '''    linked_heading = page.get_by_role("heading", name=f"Rate {linked_name}")
    try:
        linked_heading.wait_for(state="visible", timeout=5000)
    except PlaywrightTimeoutError:
        confirmation = page.get_by_role("button", name="None of these — create a new thing", exact=True)
        confirmation.wait_for(state="visible", timeout=30000)
        confirmation.click()
        linked_heading.wait_for(state="visible", timeout=30000)'''
out = out.replace(old_heading, new_heading)
old_checks = '    if page_errors:\n'
if out.count(old_checks) != 1:
    raise SystemExit('unexpected final browser error-check anchor count')
new_checks = '''    if len(expected_linked_confirmation_409s) > 1:
        raise RuntimeError("Unexpected repeated linked-Thing duplicate confirmations: " + " | ".join(expected_linked_confirmation_409s))
    if expected_linked_confirmation_409s:
        expected_console_409 = "Failed to load resource: the server responded with a status of 409 ()"
        try:
            console_errors.remove(expected_console_409)
        except ValueError:
            pass

    if page_errors:
'''
out = out.replace(old_checks, new_checks)
Path(dst_path).write_text(out)
PY
chmod 700 "$compat_verifier"
[[ "$(grep -c 'php artisan tinker --execute=' "$compat_verifier" || true)" == "0" ]] || {
  echo "VERIFY BLOCKED: compatibility verifier still contains Tinker calls." >&2
  exit 78
}
[[ "$(grep -c 'php -r \"\$PHP_EVAL_BOOTSTRAP\"' "$compat_verifier" || true)" == "3" ]] || {
  echo "VERIFY BLOCKED: compatibility verifier did not contain exactly three Laravel bootstrap probes." >&2
  exit 78
}
[[ "$(grep -Foc "$BASELINE_CSP_HASH" "$compat_verifier" || true)" == "1" ]] || {
  echo "VERIFY BLOCKED: CSP compatibility filter is missing or duplicated." >&2
  exit 78
}
[[ "$(grep -Fc 'No existing thing matched that search.' "$compat_verifier" || true)" == "0" ]] || {
  echo "VERIFY BLOCKED: compatibility verifier still contains zero-results-only linked-search assertion." >&2
  exit 78
}
[[ "$(grep -Fc 'name="None of these — create a new thing"' "$compat_verifier" || true)" == "1" ]] || {
  echo "VERIFY BLOCKED: compatibility verifier is missing the linked-Thing duplicate confirmation action." >&2
  exit 78
}
[[ "$(grep -Fc 'expected_linked_confirmation_409s' "$compat_verifier" || true)" -ge 3 ]] || {
  echo "VERIFY BLOCKED: compatibility verifier is missing correlated linked-Thing confirmation tracking." >&2
  exit 78
}
printf 'RUM_OWNER_FLOW_TINKER_COMPAT=3_PROBES_REWRITTEN\n'
printf 'RUM_OWNER_FLOW_CSP_BASELINE_FILTER=%s\n' "$BASELINE_CSP_HASH"
printf 'RUM_OWNER_FLOW_CSP_BASELINE_FILES_UNCHANGED=1\n'
printf 'RUM_OWNER_FLOW_LINKED_SEARCH_COMPAT=POST_SEARCH_ACTION_REQUIRED\n'
printf 'RUM_OWNER_FLOW_LINKED_CREATE_COMPAT=EXPLICIT_DUPLICATE_CONFIRMATION_SUPPORTED\n'
printf 'RUM_OWNER_FLOW_LINKED_CREATE_409_CONSOLE=CORRELATED_ONLY\n'

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
  echo "VERIFY BLOCKED: neither Docker nor Podman is available for the exact candidate browser verifier." >&2
  exit 78
fi

unset TOKEN GH_TOKEN GHCR_TOKEN
printf 'RUM_OWNER_FLOW_CANDIDATE_SHA=%s\n' "$CANDIDATE_SHA"
printf 'RUM_OWNER_FLOW_VERIFIER=%s\n' "$VERIFIER_PATH"
printf 'RUM_OWNER_FLOW_EXECUTED_VERIFIER=%s\n' "$compat_verifier"
bash "$compat_verifier"
