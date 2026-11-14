#!/usr/bin/env bash
set -euo pipefail
# PR #169 isolated acceptance. Only explicitly approved DEV test account,
# type, Thing and initial rating are created via the normal UI. No build,
# rollout, migration, seed/reset, direct SQL writes, LIVE or PR merge.
[[ $# -ge 1 ]] || { echo 'usage: run <sha> <id> | image <id> <stage> | db <entity> <classification-key> <journey>'; exit 64; }
mode="$1"; shift
root='/tmp/rum-rat169-youtube-browser-20261114'
case "$mode" in
image)
  [[ $# -eq 2 ]] || exit 64
  run_id="$1"; stage="$2"
  [[ "$run_id" =~ ^[a-z0-9-]{8,42}$ && "$stage" =~ ^[0-9]{2}-[a-z-]+$ ]] || exit 64
  file="$root/$run_id/$stage.jpg"
  [[ -f "$file" && ! -L "$file" ]] || { echo "EVIDENCE_NOT_FOUND=$stage"; exit 66; }
  echo "RAT_SCREENSHOT_STAGE=$stage"
  echo "RAT_SCREENSHOT_BYTES=$(wc -c < "$file")"
  printf 'RAT_SCREENSHOT_JPEG_BASE64='
  base64 -w0 "$file"
  printf '\n'
  exit 0
  ;;
db)
  [[ $# -eq 3 ]] || exit 64
  entity="$1"; classification="$2"; journey="$3"
  [[ "$entity" =~ ^[a-zA-Z0-9-]{1,64}$ && "$classification" =~ ^[a-z0-9_-]{2,100}$ && "$journey" =~ ^[a-zA-Z0-9]{1,40}$ ]] || exit 64
  kctl() { if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
  host="$(kctl -n rum-dev-isolated get ingress -o jsonpath='{.items[0].spec.rules[0].host}')"
  [[ "$host" == 'dev-rum.daisycloversoftware.uk' ]] || exit 78
  probe='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";
$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$eid=$argv[1]; $key=$argv[2]; $jid=$argv[3];
$entity=Illuminate\Support\Facades\DB::table("entities")->where("id",$eid)->first();
$type=Illuminate\Support\Facades\DB::table("entity_types")->where("key","digital_resource")->first();
$cls=Illuminate\Support\Facades\DB::table("classifications")->where("key",$key)->first();
$relation=$cls ? Illuminate\Support\Facades\DB::table("entity_classifications")->where("entity_id",$eid)->where("classification_id",$cls->id)->first() : null;
$journey=Illuminate\Support\Facades\DB::table("rating_journeys")->where("id",$jid)->where("root_entity_id",$eid)->first();
$ratings=Illuminate\Support\Facades\DB::table("rating_events")->where("target_entity_id",$eid)->count();
$result=[
 "entity_present" => (bool)$entity,
 "entity_publication_state"=>$entity ? $entity->publication_state : null,
 "entity_visibility"=>$entity ? $entity->visibility : null,
 "structural_kind_correct" => (bool)($entity && $type && (string)$entity->entity_type_id === (string)$type->id),
 "classification_present"=>(bool)$cls,
 "classification_structural_kind_correct" => (bool)($cls && $type && (string)$cls->entity_type_id === (string)$type->id),
 "canonical_assignment_present" => (bool)$relation,
 "canonical_assignment_primary"=>$relation ? (bool)$relation->is_primary : false,
 "canonical_assignment_state"=>$relation ? $relation->verification_state : null,
 "journey_present"=>(bool)$journey,
 "journey_state"=>$journey ? $journey->state : null,
 "rating_event_count"=>$ratings,
];
echo "RAT169_PERSISTED_DB=".json_encode($result, JSON_THROW_ON_ERROR)."\n";
if(!$entity || !$type || !$cls || !$relation || !$journey || $ratings<1 || $entity->publication_state!=="published" || !$result["structural_kind_correct"] || !$result["classification_structural_kind_correct"] || !$result["canonical_assignment_primary"]) exit(2);
'
  kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$probe" -- "$entity" "$classification" "$journey"
  exit 0
  ;;
run)
  [[ $# -eq 2 ]] || exit 64
  source_sha="$1"; run_id="$2"
  [[ "$source_sha" =~ ^[0-9a-f]{40}$ && "$run_id" =~ ^[a-z0-9-]{8,42}$ ]] || exit 64
  mkdir -p "$root"; chmod 700 "$root"
  dir="$root/$run_id"
  if [[ -e "$dir" ]]; then echo "EVIDENCE_EXISTS_REUSE_IMAGE_OR_DB=$dir"; exit 65; fi
  mkdir -m 700 "$dir"
  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' EXIT
  git -C "$tmp" init -q
  git -C "$tmp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
  git -C "$tmp" fetch -q --depth=1 origin "$source_sha"
  git -C "$tmp" checkout -q --detach FETCH_HEAD
  [[ "$(git -C "$tmp" rev-parse HEAD)" == "$source_sha" ]] || exit 78
  cd "$tmp/apps/rate-anything"
  npm ci --ignore-scripts --no-audit --no-fund >/dev/null
  browser_path=""
  for candidate in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser; do
    if [[ -x "$candidate" ]]; then browser_path="$candidate"; break; fi
  done
  if [[ -z "$browser_path" ]]; then npx playwright install chromium >/dev/null; fi
  RAT_ACCEPT_SHA="$source_sha" RAT_ACCEPT_ID="$run_id" RAT_ACCEPT_DIR="$dir" CHROMIUM_PATH="$browser_path" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import { randomBytes } from 'node:crypto'
import { writeFile } from 'node:fs/promises'
import { chromium, expect } from '@playwright/test'

const RAT = 'https://dev-rum-ra.daisycloversoftware.uk'
const DEV = 'https://dev-rum.daisycloversoftware.uk'
const runId = process.env.RAT_ACCEPT_ID
const output = process.env.RAT_ACCEPT_DIR
const sourceSha = process.env.RAT_ACCEPT_SHA
const thingName = 'Saffron Telescope Beacon ' + runId
const freshName = 'Obsidian Kestrel Clock ' + runId
const typeLabel = 'RAT169 DEV digital catalogue type ' + runId
const description = 'Harmless isolated DEV acceptance item. No person, identity or relationship.'
const report = { sourceSha, runId, thingName, typeLabel, checks: [], screenshots: [], errors: [], requests: [] }
const browser = await chromium.launch({ headless: true, executablePath: process.env.CHROMIUM_PATH || undefined, args: ['--no-sandbox','--disable-dev-shm-usage'] })
const context = await browser.newContext({ viewport: { width: 1100, height: 800 }, ignoreHTTPSErrors: false })
const page = await context.newPage()
let lastStage = 'preflight'
const note = (name, detail = '') => { report.checks.push({ name, detail }); console.log('RAT169_CHECK_PASS=' + name + (detail ? ' ' + detail : '')) }
const shot = async (name, locator = null) => {
  try {
    const target = locator || page
    await target.screenshot({ path: output + '/' + name + '.jpg', type: 'jpeg', quality: 35, animations: 'disabled', timeout: 15000 })
    report.screenshots.push(name)
    console.log('RAT169_SCREENSHOT_SAVED=' + name)
  } catch (err) { report.errors.push('screenshot ' + name + ': ' + String(err)) }
}
page.on('pageerror', err => report.errors.push('pageerror: ' + err.message))
page.on('response', response => {
  const url = new URL(response.url())
  if ([RAT,DEV].includes(url.origin) && url.pathname.startsWith('/api/')) {
    const req = response.request()
    if (req.method() !== 'GET' || response.status() >= 400) report.requests.push({ method: req.method(), path: url.pathname, status: response.status() })
  }
})
await context.route('**/*', route => {
  const request = route.request()
  const method = request.method()
  const url = new URL(request.url())
  if (url.hostname === 'rateurmate.online' || (!['GET','HEAD','OPTIONS'].includes(method) && ![RAT,DEV].includes(url.origin))) return route.abort('blockedbyclient')
  return route.continue()
})
const fullResponse = async (resp) => ({ status: resp.status(), body: await resp.json() })
const typeGet = async (term) => {
  const resp = await context.request.get(RAT + '/api/v1/rate-anything/thing-types?q=' + encodeURIComponent(term), { headers: {'Cache-Control':'no-cache'} })
  assert.equal(resp.status(), 200)
  return await resp.json()
}
const tryFail = async (urlGlob, method, message, fn) => {
  const handler = async route => {
    if (route.request().method() !== method) return route.continue()
    return route.fulfill({ status: 503, contentType: 'application/json', body: JSON.stringify({ message }) })
  }
  await page.route(urlGlob, handler)
  try { await fn() } finally { await page.unroute(urlGlob, handler) }
}
try {
  lastStage = 'preflight'
  const versionResp = await context.request.get(RAT + '/VERSION', { headers: { 'Cache-Control': 'no-cache' } })
  assert.equal(versionResp.status(), 200)
  const servedVersion = (await versionResp.text()).trim()
  assert.ok(servedVersion.includes(sourceSha), 'served frontend mismatch: ' + servedVersion)
  report.servedVersion = servedVersion
  note('served-frontend-version', servedVersion)
  const healthResp = await context.request.get(DEV + '/api/v1/health', { headers: { 'Cache-Control': 'no-cache' } })
  assert.equal(healthResp.status(), 200)
  const health = await healthResp.json()
  assert.equal(health.service, 'rum-api')
  assert.ok(String(health.version).includes(sourceSha), 'served canonical API mismatch: ' + JSON.stringify(health))
  report.servedApiVersion = health.version
  note('served-api-version', health.version)
  for (const term of ['DJI', 'CJ']) {
    const resp = await context.request.get(DEV + '/api/v1/public/entities/rat-catalogue/search?q=' + encodeURIComponent(term))
    assert.equal(resp.status(), 200)
    const data = await resp.json()
    assert.equal(data.meta.privacy, 'private-rum-people-excluded')
    assert.equal(data.meta.source, 'rum-dev-canonical')
    assert.ok(data.data?.active?.id)
    if (term === 'DJI') assert.equal(data.data.active.name, 'DJI')
    if (term === 'CJ') assert.equal(data.data.active.name, 'CJ Investigates')
  }
  note('DJI-CJ-canonical-discovery-private-person-policy')
  lastStage = 'no-match'
  await page.goto(RAT + '/search?q=' + encodeURIComponent(thingName), { waitUntil: 'domcontentloaded' })
  await expect(page.getByText(/No RAT-visible Thing matched/)).toBeVisible({ timeout: 30000 })
  const addAction = page.getByRole('button', { name: 'Add a Thing', exact: true })
  await expect(addAction).toBeVisible()
  const style = await addAction.evaluate(el => ({ background: getComputedStyle(el).backgroundColor, color: getComputedStyle(el).color }))
  assert.notEqual(style.background, 'rgba(0, 0, 0, 0)')
  await shot('01-no-match')
  note('genuine-no-match-add-action', JSON.stringify(style))
  await addAction.click()
  const dialog = page.getByRole('dialog', { name: 'Add a Thing', exact: true })
  await expect(dialog).toBeVisible()
  await expect(dialog.getByText('Adding: ' + thingName, { exact: false })).toBeVisible({ timeout: 10000 })
  lastStage = 'register-preview-account'
  const username = 'rat169qa' + runId.replace(/[^a-z0-9]/g, '').slice(-12)
  const password = randomBytes(20).toString('base64url') + 'Aa1!'
  await dialog.getByRole('button', { name: 'Create account', exact: true }).click()
  await dialog.getByLabel('Email', { exact: true }).fill(username + '@example.com')
  await dialog.getByLabel('Username', { exact: true }).fill(username)
  await dialog.getByLabel('Password', { exact: true }).fill(password)
  await dialog.getByLabel('Confirm password').fill(password)
  await dialog.getByRole('checkbox', { name: /Terms and Community Rules/ }).check()
  await dialog.getByRole('checkbox', { name: /Privacy Notice/ }).check()
  await dialog.getByRole('button', { name: 'Create RAT account and continue' }).click()
  await expect(dialog.locator('.contribution-member')).toBeVisible({ timeout: 20000 })
  report.testUsername = username
  note('preview-account-authenticated', username)
  const picker = dialog.getByRole('combobox', { name: 'What kind of thing is it?' })
  await expect(dialog.getByLabel('Thing name', { exact: true })).toHaveValue(thingName)
  await dialog.getByLabel('Thing description').fill(description)
  note('searched-name-prefilled')
  lastStage = 'synonym-lookup'
  await picker.fill('Creator group')
  await expect(dialog.getByRole('button', { name: /^Creator collective/ })).toBeVisible({ timeout: 15000 })
  await expect(dialog.getByRole('button', { name: /^Add “Creator group”/ })).toHaveCount(0)
  await picker.press('ArrowDown')
  await picker.press('ArrowUp')
  await picker.press('Enter')
  await expect(dialog.getByText('Selected: Creator collective', { exact: true })).toBeVisible()
  await expect(dialog.getByLabel('Thing name', { exact: true })).toHaveValue(thingName)
  await expect(dialog.getByLabel('Thing description')).toHaveValue(description)
  await shot('02-synonym-preserved', dialog)
  note('synonym-reused-keyboard-form-preserved')
  lastStage = 'lookup-error'
  await tryFail('**/api/v1/rate-anything/thing-types?*', 'GET', 'Injected DEV test catalogue lookup failure', async () => {
    await picker.fill(typeLabel)
    await expect(dialog.getByText(/Injected DEV test catalogue lookup failure/)).toBeVisible({ timeout: 15000 })
    await expect(dialog.getByRole('button', { name: /^Add “RAT169 DEV digital catalogue type/ })).toHaveCount(0)
    await expect(dialog.getByLabel('Thing name', { exact: true })).toHaveValue(thingName)
    await expect(dialog.getByLabel('Thing description')).toHaveValue(description)
    await shot('03-lookup-failure-preserved', dialog)
  })
  await dialog.getByRole('button', { name: 'Retry type lookup' }).click()
  const addType = dialog.getByRole('button', { name: /^Add “RAT169 DEV digital catalogue type/ })
  await expect(addType).toBeVisible({ timeout: 15000 })
  note('lookup-failure-retry-preserves-form')
  lastStage = 'type-creation-error'
  await addType.click()
  const kindChoice = dialog.getByRole('radio', { name: /An online page or channel/ })
  await expect(kindChoice).toBeFocused()
  await expect(dialog.getByRole('button', { name: 'Save new type' })).toBeDisabled()
  await kindChoice.check()
  await tryFail('**/api/v1/rate-anything/thing-types', 'POST', 'Injected DEV test type-creation failure', async () => {
    await dialog.getByRole('button', { name: 'Save new type' }).click()
    await expect(dialog.getByText(/Injected DEV test type-creation failure/)).toBeVisible({ timeout: 15000 })
    await expect(kindChoice).toBeChecked()
    await expect(dialog.getByLabel('Thing name', { exact: true })).toHaveValue(thingName)
    await expect(dialog.getByLabel('Thing description')).toHaveValue(description)
    await shot('04-creation-failure-preserved', dialog)
  })
  note('creation-failure-preserves-form')
  lastStage = 'create-new-type'
  const typeResponsePromise = page.waitForResponse(resp => {
    const u = new URL(resp.url())
    return u.pathname === '/api/v1/rate-anything/thing-types' && resp.request().method() === 'POST'
  })
  await dialog.getByRole('button', { name: 'Save new type' }).click()
  const typeResponse = await fullResponse(await typeResponsePromise)
  assert.ok([200,201].includes(typeResponse.status), 'type POST: ' + JSON.stringify(typeResponse))
  assert.equal(typeResponse.body.data.label, typeLabel)
  assert.equal(typeResponse.body.data.entityType, 'digital_resource')
  assert.ok(typeResponse.body.data.classificationKey)
  report.classificationKey = typeResponse.body.data.classificationKey
  report.classificationId = typeResponse.body.data.id
  const freshLookup = await typeGet(typeLabel)
  assert.ok(freshLookup.data.some(o => o.classificationKey === report.classificationKey && o.label === typeLabel), 'type missing in server response')
  await expect(dialog.getByText('Selected: ' + typeLabel)).toBeVisible()
  await expect(dialog.getByLabel('Thing name', { exact: true })).toHaveValue(thingName)
  await expect(dialog.getByLabel('Thing description')).toHaveValue(description)
  await shot('05-created-type-preserved', dialog)
  note('persisted-server-backed-new-type', report.classificationKey)
  lastStage = 'create-thing'
  const thingPromise = page.waitForResponse(resp => new URL(resp.url()).pathname === '/api/v1/rate-anything/standalone-things' && resp.request().method() === 'POST', { timeout: 30000 })
  await dialog.getByRole('button', { name: 'Check and add', exact: true }).click()
  const thingResponse = await fullResponse(await thingPromise)
  assert.equal(thingResponse.status, 201, 'Thing create POST ' + JSON.stringify(thingResponse))
  const root = thingResponse.body
  assert.equal(root.data.name, thingName)
  assert.equal(root.data.publicationState, 'pending')
  assert.equal(root.meta.publiclyOpenable, false)
  assert.equal(root.meta.canonicalWorkflow, true)
  assert.equal(root.meta.journey.state, 'active')
  assert.ok(root.data.classifications.some(c => c.key === report.classificationKey))
  const postBody = thingResponse.body
  report.entityId = root.data.id
  report.journeyId = root.meta.journey.id
  report.pendingClassificationVerified = true
  await expect(page.locator('.rat-rating-context .eyebrow')).toHaveText(typeLabel, { timeout: 20000 })
  await expect(page.getByRole('heading', { name: 'What’s your verdict on ' + thingName + '?' })).toBeVisible({ timeout: 30000 })
  await shot('06-initial-rating-pending')
  note('thing-pending-authorised-and-associated', report.entityId)
  lastStage = 'refresh-readback'
  await page.reload({ waitUntil:'domcontentloaded' })
  await expect(page.locator('.rat-rating-context .eyebrow')).toHaveText(typeLabel, { timeout: 20000 })
  await expect(page.getByRole('heading', { name: 'What’s your verdict on ' + thingName + '?' })).toBeVisible({ timeout: 25000 })
  const resumedResp = await context.request.get(RAT + '/api/v1/rate-anything/standalone-things/' + report.journeyId)
  assert.equal(resumedResp.status(),200)
  const resumed = await resumedResp.json()
  assert.equal(resumed.data.id, report.entityId)
  assert.equal(resumed.data.publicationState,'pending')
  assert.ok(resumed.data.classifications.some(c=>c.key===report.classificationKey))
  note('refresh-persisted-canonical-readback')
  lastStage = 'fresh-form-server-reuse'
  const otherPage = await context.newPage()
  await otherPage.goto(RAT + '/search?q=' + encodeURIComponent(freshName), { waitUntil: 'domcontentloaded' })
  await expect(otherPage.getByRole('button',{name:'Add a Thing',exact:true})).toBeVisible({timeout:20000})
  await otherPage.getByRole('button',{name:'Add a Thing',exact:true}).click()
  const otherDialog = otherPage.getByRole('dialog',{name:'Add a Thing',exact:true})
  await expect(otherDialog.locator('.contribution-member')).toBeVisible({timeout:15000})
  const otherPicker = otherDialog.getByRole('combobox',{name:'What kind of thing is it?'})
  const catalogResponsePromise = otherPage.waitForResponse(resp => new URL(resp.url()).pathname === '/api/v1/rate-anything/thing-types' && resp.request().method() === 'GET' && new URL(resp.url()).searchParams.get('q') === typeLabel)
  await otherPicker.fill(typeLabel)
  const catalogResponse = await catalogResponsePromise
  assert.equal(catalogResponse.status(),200)
  const catalogue = await catalogResponse.json()
  assert.ok(catalogue.data.some(o => o.classificationKey === report.classificationKey && o.label === typeLabel))
  await expect(otherDialog.getByRole('button',{name: typeLabel + ' Existing type',exact:true})).toBeVisible()
  await expect(otherDialog.getByRole('button',{name: new RegExp('^Add “RAT169 DEV digital catalogue type')})).toHaveCount(0)
  await otherPicker.press('Enter')
  await expect(otherDialog.getByText('Selected: ' + typeLabel)).toBeVisible()
  await expect(otherDialog.getByLabel('Thing name',{exact:true})).toHaveValue(freshName)
  await shot('08-fresh-form-reuse', otherDialog)
  await otherDialog.getByRole('button',{name:'Close',exact:true}).click()
  await otherPage.close()
  note('fresh-form-reuse-from-http-not-local-state')
  lastStage = 'initial-rating'
  await expect(page.locator('[data-rating-step="Value"]')).toBeVisible({timeout:20000})
  await page.getByRole('radio',{name:'5: Exceptional. Strongest approval'}).click()
  await page.getByRole('button',{name:'Continue',exact:true}).click()
  await expect(page.locator('[data-rating-step="Category"]')).toBeVisible({timeout:15000})
  await page.getByLabel('Search categories').fill('quality')
  await expect(page.getByRole('button',{name:/^Quality/i}).first()).toBeVisible({timeout:15000})
  await page.getByRole('button',{name:/^Quality/i}).first().click()
  await page.getByRole('button',{name:'Continue',exact:true}).click()
  await expect(page.locator('[data-rating-step="Reason"]')).toBeVisible()
  await page.getByLabel('Why this verdict?').fill('Harmless DEV acceptance rating for verified persisted digital test item.')
  await page.getByRole('button',{name:'Continue',exact:true}).click()
  await expect(page.locator('[data-rating-step="Review"]')).toBeVisible()
  const ratingPromise = page.waitForResponse(resp => new URL(resp.url()).pathname === '/api/v1/rate-anything/rating' && resp.request().method() === 'POST', {timeout:30000})
  await page.getByRole('button',{name:'Submit rating'}).click()
  const ratingResponse = await fullResponse(await ratingPromise)
  assert.equal(ratingResponse.status,201, 'rating POST ' + JSON.stringify(ratingResponse))
  assert.equal(ratingResponse.body.meta.thingId,report.entityId)
  report.ratingEventId = ratingResponse.body.data.id
  await expect(page.getByText('Rating submitted',{exact:true})).toBeVisible({timeout:20000})
  await shot('07-initial-rating-complete')
  const rated = await context.request.get(RAT + '/api/v1/rate-anything/standalone-things/' + report.journeyId)
  assert.equal(rated.status(),200)
  const ratedBody = await rated.json()
  assert.equal(ratedBody.data.publicationState, 'published')
  assert.equal(ratedBody.meta.publiclyOpenable, true)
  assert.ok(ratedBody.data.classifications.some(c=>c.key===report.classificationKey))
  report.finalPublication = ratedBody.data.publicationState
  note('initial-rating-submitted-published-readback',report.ratingEventId)
  report.result = 'PASS'
} catch (err) {
  report.result = 'FAIL'
  report.failedStage = lastStage
  const detail = err instanceof Error ? err.stack || err.message : String(err)
  report.errors.push(detail)
  console.error('RAT169_ACCEPTANCE_FAILED_STAGE=' + lastStage)
  console.error('RAT169_ACCEPTANCE_ERROR=' + detail.slice(0,2000))
  await shot('99-failure')
} finally {
  await writeFile(output + '/report.json', JSON.stringify(report,null,2))
  console.log('RAT169_ACCEPTANCE_REPORT=' + JSON.stringify(report))
  await browser.close()
}
if (report.result !== 'PASS') process.exit(1)
NODE
  ;;
*)
  echo 'Unknown mode' >&2; exit 64
  ;;
esac
