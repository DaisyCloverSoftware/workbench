#!/usr/bin/env bash
set -euo pipefail

NS="rum-rate-anything-preview"
HOST="dev-rum-ra.daisycloversoftware.uk"
URL="https://${HOST}"
RELEASE="rum-rate-anything"
EXPECTED_SHA="9f946210bc7053d043289107709010e8f88ee788"
EXPECTED_FRONTEND="ghcr.io/daisycloversoftware/rum-rate-anything@sha256:d18050db375eddb1adcc627721ddf31e0ebf8f2761cb423c7f933e3fa4013f73"
EXPECTED_API="ghcr.io/daisycloversoftware/rum-api@sha256:b3ee83307651abae8e9b93e63920336659f29932474788e7c0ac9022bc7bee44"
STALE_JOB="rum-migrate-20"
EVIDENCE_DIR="${HOME}/.cache/workbench-evidence/rat-s1-20260828"
PROFILE_DIR="${EVIDENCE_DIR}/chromium-profile"
DOM_FILE="${EVIDENCE_DIR}/rate-anything.dom.html"
VERSION_DOM_FILE="${EVIDENCE_DIR}/version.dom.html"
PAGE_SCREENSHOT="${EVIDENCE_DIR}/rate-anything.png"
VERSION_SCREENSHOT="${EVIDENCE_DIR}/version.png"

if sudo -n kubectl version --client >/dev/null 2>&1; then
  KUBECTL=(sudo -n kubectl)
elif sudo -n k3s kubectl version --client >/dev/null 2>&1; then
  KUBECTL=(sudo -n k3s kubectl)
else
  echo "ERROR: no sanctioned non-interactive Kubernetes client is available" >&2
  exit 11
fi

if ! "${KUBECTL[@]}" get namespace "${NS}" -o name >/dev/null 2>&1; then
  echo "ERROR: isolated RAT namespace is not readable: ${NS}" >&2
  exit 12
fi

mkdir -p "${EVIDENCE_DIR}"
rm -rf "${PROFILE_DIR}"
mkdir -p "${PROFILE_DIR}"

printf 'scope.namespace=%s\n' "${NS}"
printf 'scope.host=%s\n' "${HOST}"
printf 'expected.sha=%s\n' "${EXPECTED_SHA}"

# Neutralize only the stale failed Helm rev-20 migration hook. Suspending is
# deliberately reversible and prevents the old image from executing if it ever
# becomes pullable. No production namespace or LIVE hostname is touched.
if "${KUBECTL[@]}" get job "${STALE_JOB}" -n "${NS}" >/dev/null 2>&1; then
  "${KUBECTL[@]}" patch job "${STALE_JOB}" -n "${NS}" --type=merge -p '{"spec":{"suspend":true}}' >/dev/null
  suspended="$("${KUBECTL[@]}" get job "${STALE_JOB}" -n "${NS}" -o jsonpath='{.spec.suspend}')"
  if [[ "${suspended}" != "true" ]]; then
    echo "ERROR: stale migration job did not become suspended" >&2
    exit 21
  fi

  # Suspension should terminate any active pod. Give the controller a bounded
  # interval, then fail closed if any pod for the stale job is still Pending or Running.
  for _ in $(seq 1 30); do
    phases="$("${KUBECTL[@]}" get pods -n "${NS}" -l "job-name=${STALE_JOB}" -o jsonpath='{range .items[*]}{.status.phase}{"\n"}{end}' 2>/dev/null || true)"
    if ! grep -Eq '^(Pending|Running)$' <<<"${phases}"; then
      break
    fi
    sleep 2
  done
  phases="$("${KUBECTL[@]}" get pods -n "${NS}" -l "job-name=${STALE_JOB}" -o jsonpath='{range .items[*]}{.status.phase}{"\n"}{end}' 2>/dev/null || true)"
  if grep -Eq '^(Pending|Running)$' <<<"${phases}"; then
    echo "ERROR: stale migration job still has an active pod after suspension" >&2
    printf '%s\n' "${phases}" >&2
    exit 22
  fi
  echo "stale_migration.suspended=true"
  echo "stale_migration.active_pods=false"
else
  echo "stale_migration.absent=true"
fi

api_image="$("${KUBECTL[@]}" get deployment rum-api -n "${NS}" -o jsonpath='{.spec.template.spec.containers[0].image}')"
frontend_image="$("${KUBECTL[@]}" get deployment rum-rate-anything -n "${NS}" -o jsonpath='{.spec.template.spec.containers[0].image}')"
api_ready="$("${KUBECTL[@]}" get deployment rum-api -n "${NS}" -o jsonpath='{.status.readyReplicas}')"
frontend_ready="$("${KUBECTL[@]}" get deployment rum-rate-anything -n "${NS}" -o jsonpath='{.status.readyReplicas}')"

[[ "${api_image}" == "${EXPECTED_API}" ]] || { echo "ERROR: API digest mismatch: ${api_image}" >&2; exit 31; }
[[ "${frontend_image}" == "${EXPECTED_FRONTEND}" ]] || { echo "ERROR: frontend digest mismatch: ${frontend_image}" >&2; exit 32; }
[[ "${api_ready:-0}" -ge 1 ]] || { echo "ERROR: API deployment is not ready" >&2; exit 33; }
[[ "${frontend_ready:-0}" -ge 1 ]] || { echo "ERROR: frontend deployment is not ready" >&2; exit 34; }

printf 'deployed.api=%s\n' "${api_image}"
printf 'deployed.frontend=%s\n' "${frontend_image}"
printf 'deployment.api_ready=%s\n' "${api_ready}"
printf 'deployment.frontend_ready=%s\n' "${frontend_ready}"

# Read the deployed Helm revision from Helm's release-secret labels using the
# same sanctioned Kubernetes client, avoiding any ambient kubeconfig dependency.
helm_revisions="$("${KUBECTL[@]}" get secrets -n "${NS}" -l "owner=helm,name=${RELEASE},status=deployed" -o jsonpath='{range .items[*]}{.metadata.labels.version}{"\n"}{end}')"
helm_revision="$(printf '%s\n' "${helm_revisions}" | grep -E '^[0-9]+$' | sort -n | tail -n 1)"
[[ -n "${helm_revision}" ]] || { echo "ERROR: no deployed Helm release revision found" >&2; exit 35; }
printf 'helm.revision=%s\n' "${helm_revision}"
printf 'helm.status=deployed\n'

BROWSER=""
for candidate in chromium chromium-browser google-chrome google-chrome-stable; do
  if command -v "${candidate}" >/dev/null 2>&1; then
    BROWSER="$(command -v "${candidate}")"
    break
  fi
done
[[ -n "${BROWSER}" ]] || { echo "ERROR: no Chromium-family browser is installed on the Workbench runner" >&2; exit 41; }
printf 'browser.binary=%s\n' "${BROWSER}"

browser_common=(
  --headless
  --no-sandbox
  --disable-gpu
  --disable-dev-shm-usage
  --incognito
  --user-data-dir="${PROFILE_DIR}"
  --window-size=1440,1800
  --virtual-time-budget=8000
)

"${BROWSER}" "${browser_common[@]}" --dump-dom "${URL}/" >"${DOM_FILE}" 2>"${EVIDENCE_DIR}/dom.stderr.log"
"${BROWSER}" "${browser_common[@]}" --screenshot="${PAGE_SCREENSHOT}" "${URL}/" >"${EVIDENCE_DIR}/screenshot.stdout.log" 2>"${EVIDENCE_DIR}/screenshot.stderr.log"
"${BROWSER}" "${browser_common[@]}" --dump-dom "${URL}/VERSION" >"${VERSION_DOM_FILE}" 2>"${EVIDENCE_DIR}/version-dom.stderr.log"
"${BROWSER}" "${browser_common[@]}" --screenshot="${VERSION_SCREENSHOT}" "${URL}/VERSION" >"${EVIDENCE_DIR}/version-screenshot.stdout.log" 2>"${EVIDENCE_DIR}/version-screenshot.stderr.log"

for marker in \
  'RUM — Rate Anything' \
  'Search anything. Find the right thing first.' \
  'Find it. Understand it. Rate it.' \
  'One down. Five up. No stars.' \
  'Explore RUM' \
  '/brand/rum-swipe-card-r-mark-transparent.png' \
  '/brand/rum-six-position-rating-device.png'; do
  grep -Fq "${marker}" "${DOM_FILE}" || { echo "ERROR: browser DOM missing marker: ${marker}" >&2; exit 51; }
done

if grep -Fq 'RateUrMate' "${DOM_FILE}"; then
  echo "ERROR: browser DOM still contains forbidden legacy RateUrMate marker" >&2
  exit 52
fi
if grep -Fq 'rum-primary-horizontal-logo-on-cream.png' "${DOM_FILE}"; then
  echo "ERROR: browser DOM still contains forbidden old wordmark asset" >&2
  exit 53
fi

grep -Fq "${EXPECTED_SHA}" "${VERSION_DOM_FILE}" || {
  echo "ERROR: browser /VERSION did not expose expected SHA" >&2
  sed -n '1,40p' "${VERSION_DOM_FILE}" >&2
  exit 54
}

[[ -s "${PAGE_SCREENSHOT}" ]] || { echo "ERROR: page screenshot was not created" >&2; exit 55; }
[[ -s "${VERSION_SCREENSHOT}" ]] || { echo "ERROR: VERSION screenshot was not created" >&2; exit 56; }

page_sha256="$(sha256sum "${PAGE_SCREENSHOT}" | awk '{print $1}')"
version_sha256="$(sha256sum "${VERSION_SCREENSHOT}" | awk '{print $1}')"
page_bytes="$(wc -c <"${PAGE_SCREENSHOT}" | tr -d ' ')"
version_bytes="$(wc -c <"${VERSION_SCREENSHOT}" | tr -d ' ')"

printf 'version=%s\n' "${EXPECTED_SHA}"
printf 'browser.page_verified=true\n'
printf 'browser.old_page_replaced=true\n'
printf 'browser.version_verified=true\n'
printf 'evidence.page.path=%s\n' "${PAGE_SCREENSHOT}"
printf 'evidence.page.sha256=%s\n' "${page_sha256}"
printf 'evidence.page.bytes=%s\n' "${page_bytes}"
printf 'evidence.version.path=%s\n' "${VERSION_SCREENSHOT}"
printf 'evidence.version.sha256=%s\n' "${version_sha256}"
printf 'evidence.version.bytes=%s\n' "${version_bytes}"

echo "OWNER_REVIEW_PROOF_OK"
