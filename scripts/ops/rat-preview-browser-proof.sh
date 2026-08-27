#!/usr/bin/env bash
set -euo pipefail
umask 077

NAMESPACE="rum-rate-anything-preview"
HOST="dev-rum-ra.daisycloversoftware.uk"
URL="https://${HOST}"
EXPECTED_VERSION="9f946210bc7053d043289107709010e8f88ee788"
POD="rat-browser-proof-$(date +%s)"
ARTIFACT="/tmp/rat-browser-evidence-20260828.jpg"
IMAGE="mcr.microsoft.com/playwright:v1.62.0-noble"

for command in sudo k3s grep sha256sum base64 mktemp; do
  command -v "$command" >/dev/null 2>&1 || {
    echo "ERROR: required command unavailable: $command" >&2
    exit 3
  }
done

kc() {
  sudo -n k3s kubectl "$@"
}

cleanup() {
  kc -n "$NAMESPACE" delete pod "$POD" --ignore-not-found --wait=false >/dev/null 2>&1 || true
}

node_script="$(mktemp)"
trap 'rm -f "$node_script"; cleanup' EXIT HUP INT TERM
cat > "$node_script" <<'NODE'
const fs = require('fs');
const crypto = require('crypto');
const { chromium } = require('playwright');

(async () => {
  const url = process.env.RAT_URL;
  const expectedVersion = process.env.RAT_EXPECTED_VERSION;
  const screenshot = '/evidence/rat-sprint.jpg';
  const markers = [
    'Search anything. Find the right thing first.',
    'Find it. Understand it. Rate it.',
    'One down. Five up. No stars.',
    'Explore RUM',
  ];
  const consoleErrors = [];
  const pageErrors = [];

  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({
    ignoreHTTPSErrors: false,
    viewport: { width: 1440, height: 1000 },
    serviceWorkers: 'block',
  });
  const page = await context.newPage();
  page.on('console', (msg) => { if (msg.type() === 'error') consoleErrors.push(msg.text()); });
  page.on('pageerror', (err) => pageErrors.push(String(err)));
  await page.setExtraHTTPHeaders({ 'Cache-Control': 'no-cache', 'Pragma': 'no-cache' });
  const response = await page.goto(url + '/', { waitUntil: 'networkidle', timeout: 60000 });
  await page.waitForTimeout(2500);
  const title = await page.title();
  const visible = await page.locator('body').innerText();
  await page.screenshot({ path: screenshot, fullPage: true, type: 'jpeg', quality: 78 });

  const versionPage = await context.newPage();
  await versionPage.setExtraHTTPHeaders({ 'Cache-Control': 'no-cache', 'Pragma': 'no-cache' });
  const versionResponse = await versionPage.goto(url + '/VERSION', { waitUntil: 'domcontentloaded', timeout: 30000 });
  const versionText = (await versionPage.locator('body').innerText()).trim();

  const bytes = fs.readFileSync(screenshot);
  const digest = crypto.createHash('sha256').update(bytes).digest('hex');
  const compactVisible = visible.replace(/\s+/g, ' ').trim().slice(0, 2200);
  console.log('HTTP_STATUS=' + (response ? response.status() : 'none'));
  console.log('BROWSER_TITLE=' + title.replace(/\s+/g, ' ').trim());
  console.log('VISIBLE_TEXT=' + compactVisible);
  console.log('CONSOLE_ERRORS=' + consoleErrors.map(x => x.replace(/\s+/g, ' ')).join(' || ').slice(0, 1800));
  console.log('PAGE_ERRORS=' + pageErrors.map(x => x.replace(/\s+/g, ' ')).join(' || ').slice(0, 1800));
  console.log('VERSION_HTTP_STATUS=' + (versionResponse ? versionResponse.status() : 'none'));
  console.log('VERSION=' + versionText.replace(/\s+/g, ' ').trim());
  console.log('SCREENSHOT_SHA256=' + digest);
  console.log('__RAT_BROWSER_CAPTURED__');

  const missing = markers.filter(marker => !visible.includes(marker));
  if (missing.length) throw new Error('missing visible markers: ' + missing.join(' || '));
  if (visible.includes('RateUrMate')) throw new Error('obsolete RateUrMate placeholder text is still visibly present');
  if (!versionText.includes(expectedVersion)) throw new Error('VERSION mismatch: ' + JSON.stringify(versionText));

  console.log('VISIBLE_MARKER_SEARCH=true');
  console.log('VISIBLE_MARKER_FIND_RATE=true');
  console.log('VISIBLE_MARKER_SCALE=true');
  console.log('VISIBLE_MARKER_EXPLORE=true');
  console.log('OLD_PLACEHOLDER_VISIBLE=false');
  console.log('__RAT_BROWSER_PROOF_DONE__');
  await browser.close();
})().catch((error) => {
  console.error('BROWSER_PROOF_ERROR=' + (error && error.stack ? error.stack : String(error)));
  process.exit(1);
});
NODE

PROOF_JS_B64="$(base64 -w 0 "$node_script")"

kc -n "$NAMESPACE" run "$POD" \
  --image="$IMAGE" \
  --restart=Never \
  --labels=app.kubernetes.io/component=rat-browser-proof \
  --env="RAT_URL=$URL" \
  --env="RAT_EXPECTED_VERSION=$EXPECTED_VERSION" \
  --env="PROOF_JS_B64=$PROOF_JS_B64" \
  --command -- bash -lc '
    set -u
    mkdir -p /proof /evidence
    printf "%s" "$PROOF_JS_B64" | base64 -d > /proof/proof.js
    cd /proof
    npm init -y >/dev/null 2>&1
    npm install --no-save playwright@1.62.0
    node /proof/proof.js
    status=$?
    echo BROWSER_NODE_EXIT=$status
    sleep 600
  ' >/dev/null

echo "BROWSER_PROOF_POD=$NAMESPACE/$POD"

logs=""
for _ in $(seq 1 120); do
  logs="$(kc -n "$NAMESPACE" logs "$POD" 2>&1 || true)"
  if grep -Eq '__RAT_BROWSER_(CAPTURED|PROOF_DONE)__' <<<"$logs"; then
    if grep -Fq 'BROWSER_NODE_EXIT=' <<<"$logs"; then
      break
    fi
  fi
  phase="$(kc -n "$NAMESPACE" get pod "$POD" -o jsonpath='{.status.phase}' 2>/dev/null || true)"
  if [[ "$phase" == "Failed" || "$phase" == "Succeeded" ]]; then
    break
  fi
  sleep 5
done

printf '%s\n' "$logs"

if grep -Fq '__RAT_BROWSER_CAPTURED__' <<<"$logs"; then
  kc -n "$NAMESPACE" cp "$POD:/evidence/rat-sprint.jpg" "$ARTIFACT" >/dev/null || true
fi
if [[ -s "$ARTIFACT" ]]; then
  printf 'BROWSER_EVIDENCE_CAPTURED=true\n'
  printf 'BROWSER_EVIDENCE_PATH=%s\n' "$ARTIFACT"
  printf 'BROWSER_EVIDENCE_SHA256=%s\n' "$(sha256sum "$ARTIFACT" | awk '{print $1}')"
  printf 'BROWSER_EVIDENCE_BYTES=%s\n' "$(wc -c < "$ARTIFACT" | tr -d ' ')"
else
  printf 'BROWSER_EVIDENCE_CAPTURED=false\n'
fi

grep -Fq '__RAT_BROWSER_PROOF_DONE__' <<<"$logs" || {
  echo "ERROR: browser proof assertions did not pass" >&2
  exit 5
}
