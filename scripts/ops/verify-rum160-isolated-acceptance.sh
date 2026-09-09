#!/usr/bin/env bash
set -euo pipefail
umask 077

if [[ $# -ne 0 ]]; then
  echo "usage: $0" >&2
  exit 64
fi

SOURCE_SHA="226f898fa0508b9f5b41b8d0651bab315b7dd204"
API_IMAGE="ghcr.io/daisycloversoftware/rum-api@sha256:8daee721ada11f1ffcaf89a1c874319ed307ce51884153b8e7a2298b4a66936e"
WEB_IMAGE="ghcr.io/daisycloversoftware/rum-web@sha256:77f83cf631918cc72179cc8e57a7b12b1ff8884ef6efdde58306d7e3cabf401e"
RAT_IMAGE="ghcr.io/daisycloversoftware/rum-rate-anything@sha256:619130a9843cbd2c38d22e591019b3196c5960eea3583cdff01dd3524086dcbb"
RUM_NS="rum-dev-isolated"
RAT_NS="rum-rate-anything-preview"
PUBLIC_NS="rum-dev"
RUM_ORIGIN="https://dev-rum.daisycloversoftware.uk"
RAT_ORIGIN="https://dev-rum-ra.daisycloversoftware.uk"
PUBLIC_HOST="rateurmate.online"
SENTINEL_NAME="RUM160 Private Acceptance Sentinel ${SOURCE_SHA:0:8}"
SENTINEL_SLUG="rum160-private-acceptance-${SOURCE_SHA:0:12}"
workdir=""

kctl() {
  if command -v k3s >/dev/null 2>&1; then
    sudo -n k3s kubectl "$@"
  else
    kubectl "$@"
  fi
}

deployment_image() {
  local ns="$1" deployment="$2" container="$3"
  kctl -n "$ns" get deployment "$deployment" -o "jsonpath={.spec.template.spec.containers[?(@.name==\"$container\")].image}"
}

cleanup() {
  set +e
  kctl -n "$RUM_NS" exec deployment/rum-api -c php-fpm -- \
    env RUM160_SENTINEL_SLUG="$SENTINEL_SLUG" php -r '
require "vendor/autoload.php";
$app = require "bootstrap/app.php";
$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
App\Models\Entity::query()->where("canonical_slug", getenv("RUM160_SENTINEL_SLUG"))->delete();
' >/dev/null 2>&1
  [[ -n "$workdir" ]] && rm -rf "$workdir"
}
trap cleanup EXIT HUP INT TERM

# Refuse acceptance until all exact isolated candidate identities are pinned.
[[ "$(deployment_image "$RUM_NS" rum-api php-fpm)" == "$API_IMAGE" ]]
[[ "$(deployment_image "$RUM_NS" rum-web web)" == "$WEB_IMAGE" ]]
[[ "$(deployment_image "$RAT_NS" rum-api php-fpm)" == "$API_IMAGE" ]]
[[ "$(deployment_image "$RAT_NS" rum-rate-anything rate-anything)" == "$RAT_IMAGE" ]]

# Never contact LIVE during acceptance. Only assert its namespace still exists and leave it untouched.
kctl get namespace "$PUBLIC_NS" >/dev/null

# Prove the deployed RAT frontend itself carries the exact source revision.
rat_version="$(kctl -n "$RAT_NS" exec deployment/rum-rate-anything -c rate-anything -- cat /usr/share/nginx/html/VERSION | tr -d '\r\n')"
grep -Fq "$SOURCE_SHA" <<<"$rat_version"

# The current isolated dataset has no private active person. Create one deterministic
# synthetic sentinel in isolated RUM DEV only, exercise the real RAT catalogue privacy
# boundary against it, then delete it in the EXIT trap on both pass and failure.
kctl -n "$RUM_NS" exec deployment/rum-api -c php-fpm -- \
  env RUM160_SENTINEL_NAME="$SENTINEL_NAME" RUM160_SENTINEL_SLUG="$SENTINEL_SLUG" php -r '
require "vendor/autoload.php";
$app = require "bootstrap/app.php";
$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$name = getenv("RUM160_SENTINEL_NAME");
$slug = getenv("RUM160_SENTINEL_SLUG");
$type = App\Models\EntityType::query()->where("key", "person")->firstOrFail();
App\Models\Entity::query()->where("canonical_slug", $slug)->delete();
App\Models\Entity::query()->create([
  "entity_type_id" => $type->id,
  "canonical_name" => $name,
  "normalised_name" => app(App\Services\EntityNameNormalizer::class)->normalise($name),
  "canonical_slug" => $slug,
  "description" => "Temporary isolated RUM160 acceptance sentinel.",
  "status" => "active",
  "visibility" => "private",
  "rateability" => "closed",
  "reviewability" => "closed",
  "sensitivity" => "normal",
  "verification_state" => "unverified",
  "publication_state" => "published",
]);
' >/dev/null

workdir="$(mktemp -d)"
cd "$workdir"
npm init -y >/dev/null 2>&1
npm install --ignore-scripts --no-audit --no-fund @playwright/test@1.57.0 >/dev/null

browser_path=""
for candidate in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser; do
  if [[ -x "$candidate" ]]; then
    browser_path="$candidate"
    break
  fi
done
if [[ -z "$browser_path" ]]; then
  npx playwright install chromium >/dev/null
fi

SOURCE_SHA="$SOURCE_SHA" \
API_IMAGE="$API_IMAGE" \
WEB_IMAGE="$WEB_IMAGE" \
RAT_IMAGE="$RAT_IMAGE" \
RUM_ORIGIN="$RUM_ORIGIN" \
RAT_ORIGIN="$RAT_ORIGIN" \
PRIVATE_PERSON_NAME="$SENTINEL_NAME" \
CHROMIUM_PATH="$browser_path" \
node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import crypto from 'node:crypto'
import { chromium, expect } from '@playwright/test'

const sourceSha = process.env.SOURCE_SHA
const rum = process.env.RUM_ORIGIN
const rat = process.env.RAT_ORIGIN
const privatePersonName = process.env.PRIVATE_PERSON_NAME
const executablePath = process.env.CHROMIUM_PATH || undefined
const allowedOrigins = new Set([rum, rat])
const viewports = [
  { name: 'desktop', width: 1440, height: 1000 },
  { name: 'mobile', width: 390, height: 844 },
]

const browser = await chromium.launch({
  headless: true,
  executablePath,
  args: ['--no-sandbox', '--disable-dev-shm-usage'],
})

const evidence = {
  sourceSha,
  images: { api: process.env.API_IMAGE, web: process.env.WEB_IMAGE, rat: process.env.RAT_IMAGE },
  hosts: [rum, rat],
  privacy: null,
  viewports: [],
}

try {
  const api = await browser.newContext()
  const canonicalResponse = await api.request.get(`${rum}/api/v1/public/entities/rat-catalogue/search?q=CJ`, { maxRedirects: 0 })
  assert.equal(canonicalResponse.status(), 200, 'canonical CJ HTTP status')
  const canonical = await canonicalResponse.json()
  assert.equal(canonical?.meta?.source, 'rum-dev-canonical')
  assert.equal(canonical?.data?.active?.name, 'CJ Investigates')
  const canonicalId = canonical?.data?.active?.id
  assert.ok(canonicalId, 'CJ canonical id missing')

  // Synthetic private-person sentinel must never be surfaced by the real deployed catalogue.
  const privateResponse = await api.request.get(`${rum}/api/v1/public/entities/rat-catalogue/search?${new URLSearchParams({ q: privatePersonName })}`, { maxRedirects: 0 })
  assert.equal(privateResponse.status(), 200, 'private-person canonical query HTTP status')
  const privateJson = await privateResponse.json()
  const activePrivateCandidate = privateJson?.data?.active
  assert.notEqual(activePrivateCandidate?.name, privatePersonName, 'private RUM person leaked into RAT canonical catalogue')
  evidence.privacy = {
    check: 'synthetic private RUM person excluded from deployed RAT canonical catalogue',
    result: 'PASS',
    foundAlternativePublicResult: Boolean(activePrivateCandidate),
  }
  await api.close()

  for (const viewport of viewports) {
    const context = await browser.newContext({ viewport: { width: viewport.width, height: viewport.height } })
    const page = await context.newPage()
    const errors = []
    const blocked = []
    const apiRequests = []
    const heading = page.locator('.rat-active-card h1')
    const routeState = () => new URL(page.url()).searchParams
    const search = async query => {
      await page.locator('#rat-search-results').fill(query)
      await page.getByRole('button', { name: 'Search', exact: true }).click()
    }
    const snapHash = async () => {
      const bytes = await page.screenshot({ fullPage: true })
      return crypto.createHash('sha256').update(bytes).digest('hex')
    }

    page.on('pageerror', error => errors.push(`page:${error.message}`))
    page.on('console', message => {
      if (message.type() === 'error') errors.push(`console:${message.text()}`)
    })
    page.on('response', response => {
      const url = new URL(response.url())
      if (allowedOrigins.has(url.origin) && url.pathname.startsWith('/api/')) {
        apiRequests.push({ path: url.pathname + url.search, status: response.status() })
        if (response.status() >= 500) errors.push(`HTTP ${response.status()} ${url.pathname}`)
      }
    })
    await context.route('**/*', async route => {
      const request = route.request()
      const url = new URL(request.url())
      if (!allowedOrigins.has(url.origin) || !['GET', 'HEAD', 'OPTIONS'].includes(request.method())) {
        blocked.push({ method: request.method(), origin: url.origin, path: url.pathname })
        await route.abort('blockedbyclient')
        return
      }
      await route.continue()
    })

    try {
      await page.goto(`${rat}/search?q=DJI`, { waitUntil: 'domcontentloaded', timeout: 30_000 })
      await expect(heading).toHaveText('DJI', { timeout: 30_000 })
      const djiActive = routeState().get('active')
      assert.ok(djiActive, 'DJI result did not establish active state')
      const beforeHash = await snapHash()

      // Delay CJ: explicit search must clear stale active=DJI before the new result arrives.
      let releaseSearch
      const gate = new Promise(resolve => { releaseSearch = resolve })
      const cjRoute = `${rum}/api/v1/public/entities/rat-catalogue/search?q=CJ`
      await page.route(cjRoute, async route => { await gate; await route.continue() })
      try {
        await search('CJ')
        assert.equal(routeState().get('q'), 'CJ')
        assert.equal(routeState().get('active'), null, 'explicit CJ search retained stale DJI active state')
      } finally {
        releaseSearch()
      }

      await expect(heading).toHaveText('CJ Investigates', { timeout: 30_000 })
      await page.unroute(cjRoute)
      assert.equal(routeState().get('active'), canonicalId, 'CJ search did not activate canonical CJ identity')
      assert.notEqual(routeState().get('active'), djiActive, 'CJ reused DJI active identity')
      assert.equal(await page.locator('.rat-active-card h1', { hasText: 'DJI' }).count(), 0, 'DJI remained the active result for CJ')
      const cjHash = await snapHash()

      // Approved UI contract: existing search/active/exploration surfaces remain visible,
      // with no mobile/desktop horizontal overflow. PR #160 contains no RAT style/component changes.
      await expect(page.locator('#rat-search-results')).toBeVisible()
      await expect(page.getByRole('button', { name: 'Search', exact: true })).toBeVisible()
      await expect(page.locator('.rat-active-card')).toBeVisible()
      const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)
      assert.ok(overflow <= 1, `horizontal overflow detected: ${overflow}px`)

      await search('zzzz-no-such-rat-entity-zzzz')
      await expect(page.locator('.rat-message')).toBeVisible({ timeout: 30_000 })
      await expect(heading).toHaveCount(0)
      assert.equal(routeState().get('active'), null, 'no-match retained active URL state')
      const noMatchHash = await snapHash()

      await page.goto(`${rat}/search?q=DJI`, { waitUntil: 'domcontentloaded', timeout: 30_000 })
      await expect(heading).toHaveText('DJI', { timeout: 30_000 })
      const linked = page.locator('.rat-linked-card').first()
      await expect(linked).toBeVisible({ timeout: 30_000 })
      const linkedName = (await linked.locator('strong').innerText()).trim()
      await linked.click()
      await expect(heading).toHaveText(linkedName, { timeout: 30_000 })
      const linkedActive = routeState().get('active')
      assert.ok(linkedActive)
      assert.notEqual(linkedActive, djiActive)
      assert.equal(routeState().get('q'), 'DJI', 'linked Thing lost search context')
      await page.reload({ waitUntil: 'domcontentloaded' })
      await expect(heading).toHaveText(linkedName, { timeout: 30_000 })
      assert.equal(routeState().get('active'), linkedActive)
      await search('CJ')
      await expect(heading).toHaveText('CJ Investigates', { timeout: 30_000 })
      assert.equal(routeState().get('active'), canonicalId)
      const linkedHash = await snapHash()

      assert.deepEqual(errors, [], `${viewport.name} browser/runtime errors`)
      assert.deepEqual(blocked, [], `${viewport.name} attempted write/LIVE/external request`)
      evidence.viewports.push({
        name: viewport.name,
        viewport: { width: viewport.width, height: viewport.height },
        djiActive,
        cjActive: canonicalId,
        linkedActive,
        linkedName,
        horizontalOverflowPx: overflow,
        screenshots: { dji: beforeHash, cj: cjHash, noMatch: noMatchHash, linkedThenCj: linkedHash },
        apiRequests,
        errors: [],
        blockedRequests: [],
        result: 'PASS',
      })
    } finally {
      await context.close()
    }
  }

  evidence.result = 'PASS'
  console.log('RUM160_ACCEPTANCE ' + JSON.stringify(evidence))
  console.log('RUM160_PRIVATE_PERSON_EXCLUSION_VERIFIED')
  console.log('RUM160_DESKTOP_MOBILE_BROWSER_VERIFIED')
  console.log('RUM160_LINKED_THING_VERIFIED')
  console.log('RUM160_NO_MATCH_VERIFIED')
  console.log('RUM160_BROWSER_ERROR_CHECKS_VERIFIED')
} finally {
  await browser.close()
}
NODE

printf 'RUM160_ACCEPTANCE_COMPLETE source_sha=%s public_host_not_contacted=%s\n' "$SOURCE_SHA" "$PUBLIC_HOST"
