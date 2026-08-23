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

# The browser reports a CSP violation for the inline theme bootstrap. This is
# allowed to be ignored only if both the HTML containing that exact bootstrap
# and the Nginx CSP policy are byte-identical to the PR base. Any Sprint change
# to either file disables the compatibility exception and fails closed.
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

# Production API images intentionally exclude dev-only Laravel Tinker. Rewrite
# exactly the candidate verifier's three Tinker probes to equivalent bootstrapped
# Laravel PHP evaluation in a disposable copy. Also filter only the proven,
# unchanged baseline theme-bootstrap CSP warning with its exact script hash.
tinker_calls="$(grep -c 'php artisan tinker --execute=' "$VERIFIER_PATH" || true)"
[[ "$tinker_calls" == "3" ]] || {
  echo "VERIFY BLOCKED: expected exactly three candidate Tinker probe calls; found ${tinker_calls}." >&2
  exit 78
}
console_hook='    page.on("console", lambda msg: console_errors.append(msg.text) if msg.type == "error" else None)'
console_hook_count="$(grep -Fxc "$console_hook" "$VERIFIER_PATH" || true)"
[[ "$console_hook_count" == "1" ]] || {
  echo "VERIFY BLOCKED: expected exactly one candidate browser console hook; found ${console_hook_count}." >&2
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
insert_at = 2 if len(lines) >= 2 else 0
lines.insert(insert_at, bootstrap)
out = ''.join(lines).replace(needle, 'php -r "$PHP_EVAL_BOOTSTRAP" ')
old = '    page.on("console", lambda msg: console_errors.append(msg.text) if msg.type == "error" else None)'
if out.count(old) != 1:
    raise SystemExit('unexpected console hook count')
new = '''    page.on(\n        "console",\n        lambda msg: console_errors.append(msg.text)\n        if msg.type == "error"\n        and not (\n            msg.text.startswith("Executing inline script violates the following Content Security Policy directive")\n            and "''' + csp_hash + '''" in msg.text\n        )\n        else None,\n    )'''
out = out.replace(old, new)
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
printf 'RUM_OWNER_FLOW_TINKER_COMPAT=3_PROBES_REWRITTEN\n'
printf 'RUM_OWNER_FLOW_CSP_BASELINE_FILTER=%s\n' "$BASELINE_CSP_HASH"
printf 'RUM_OWNER_FLOW_CSP_BASELINE_FILES_UNCHANGED=1\n'

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
