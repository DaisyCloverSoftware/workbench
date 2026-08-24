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

for command in gh git mktemp bash python3 sed grep; do
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

# The production API image intentionally excludes dev-only Laravel Tinker.
# Preserve the exact candidate's browser journey and database assertions while
# adapting only its three `artisan tinker --execute=...` calls in a disposable
# verifier copy to equivalent bootstrapped Laravel PHP evaluation.
tinker_calls="$(grep -c 'php artisan tinker --execute=' "$VERIFIER_PATH" || true)"
[[ "$tinker_calls" == "3" ]] || {
  echo "VERIFY BLOCKED: expected exactly three candidate Tinker probe calls; found ${tinker_calls}." >&2
  exit 78
}
compat_verifier="$tmp_root/verify-rum-dev-owner-rating-flow-compat.sh"
python3 - "$VERIFIER_PATH" "$compat_verifier" <<'PY'
from pathlib import Path
import sys
src = Path(sys.argv[1]).read_text()
bootstrap = r'''PHP_EVAL_BOOTSTRAP='require "/var/www/html/vendor/autoload.php"; $app=require "/var/www/html/bootstrap/app.php"; $app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap(); eval($argv[1]);' '''.rstrip() + "\n"
needle = 'php artisan tinker --execute='
if src.count(needle) != 3:
    raise SystemExit('unexpected tinker call count')
lines = src.splitlines(keepends=True)
insert_at = 2 if len(lines) >= 2 else 0
lines.insert(insert_at, bootstrap)
out = ''.join(lines).replace(needle, 'php -r "$PHP_EVAL_BOOTSTRAP" ')
old_checks = '''    if page_errors:\n        raise RuntimeError("Browser page errors: " + " | ".join(page_errors[:5]))\n    if console_errors:\n        raise RuntimeError("Browser console errors: " + " | ".join(console_errors[:5]))\n    if request_failures:\n        raise RuntimeError("Browser request failures: " + " | ".join(request_failures[:5]))\n    if api_failures:\n        raise RuntimeError("API responses >=400 during verified flow: " + " | ".join(api_failures[:5]))\n'''
new_checks = '''    if api_failures:\n        raise RuntimeError("API responses >=400 during verified flow: " + " | ".join(api_failures[:5]))\n    if request_failures:\n        raise RuntimeError("Browser request failures: " + " | ".join(request_failures[:5]))\n    if page_errors:\n        raise RuntimeError("Browser page errors: " + " | ".join(page_errors[:5]))\n    if console_errors:\n        raise RuntimeError("Browser console errors: " + " | ".join(console_errors[:5]))\n'''
if out.count(old_checks) != 1:
    raise SystemExit('unexpected browser failure-check block')
out = out.replace(old_checks, new_checks)
Path(sys.argv[2]).write_text(out)
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
printf 'RUM_OWNER_FLOW_TINKER_COMPAT=3_PROBES_REWRITTEN\n'
printf 'RUM_OWNER_FLOW_DIAGNOSTIC_ORDER=API_FIRST\n'

# The cluster-control host intentionally has no Docker daemon. Podman is
# Docker-CLI compatible for the candidate verifier's bounded `docker run`
# usage. The official Playwright Python image carries the browser binaries but
# does not include the Python playwright package expected by the exact
# candidate verifier, so derive a local compatibility image from that exact
# base and retag it under the verifier's unchanged image reference.
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

# The cloned verifier needs no GitHub credential. Drop token variables before
# invoking the candidate's browser/database isolation check.
unset TOKEN GH_TOKEN GHCR_TOKEN

printf 'RUM_OWNER_FLOW_CANDIDATE_SHA=%s\n' "$CANDIDATE_SHA"
printf 'RUM_OWNER_FLOW_VERIFIER=%s\n' "$VERIFIER_PATH"
printf 'RUM_OWNER_FLOW_EXECUTED_VERIFIER=%s\n' "$compat_verifier"

bash "$compat_verifier"
