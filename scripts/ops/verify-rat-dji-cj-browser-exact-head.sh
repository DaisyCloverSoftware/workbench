#!/usr/bin/env bash
set -euo pipefail
if [[ $# -ne 1 ]]; then echo "usage: $0 <40-char-rum-source-sha>" >&2; exit 64; fi
SOURCE_SHA="$1"
[[ "$SOURCE_SHA" =~ ^[0-9a-f]{40}$ ]] || { echo "invalid source SHA" >&2; exit 64; }
RAT="https://dev-rum-ra.daisycloversoftware.uk"
DEV="https://dev-rum.daisycloversoftware.uk"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q
git -C "$tmp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$tmp" fetch -q --depth=1 origin "$SOURCE_SHA"
git -C "$tmp" checkout -q --detach FETCH_HEAD
APP="$tmp/apps/rate-anything"
cd "$APP"
npm ci --ignore-scripts --no-audit --no-fund >/dev/null
browser_path=""
for candidate in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser; do
  if [[ -x "$candidate" ]]; then browser_path="$candidate"; break; fi
done
if [[ -z "$browser_path" ]]; then npx playwright install chromium >/dev/null; fi
RAT="$RAT" DEV="$DEV" CHROMIUM_PATH="$browser_path" node --input-type=module <<'NODE'
import { chromium } from '@playwright/test'
const rat=process.env.RAT
const dev=process.env.DEV
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROMIUM_PATH||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
const page=await browser.newPage({viewport:{width:1440,height:1000}})
try {
  await page.goto(rat+'/search?q=DJI',{waitUntil:'networkidle',timeout:30000})
  await page.locator('.rat-active-card h1',{hasText:'DJI'}).waitFor({state:'visible',timeout:20000})
  const djiUrl=new URL(page.url())
  if(!djiUrl.searchParams.get('active')) throw new Error('DJI did not establish active state')
  const linked=page.locator('.rat-linked-card').filter({hasText:'DJI Mini Pro 5'}).first()
  await linked.waitFor({state:'visible',timeout:20000})
  await linked.click()
  await page.locator('.rat-active-card h1',{hasText:'DJI Mini Pro 5'}).waitFor({state:'visible',timeout:20000})
  await page.getByRole('button',{name:'Rate',exact:true}).click()
  await page.waitForTimeout(300)
  if(new URL(page.url()).origin!==new URL(rat).origin) throw new Error('Rate navigation left RAT origin')
  if(!page.url().includes('#rate')) throw new Error('Rate navigation did not enter RAT rate flow')
  await page.goto(rat+'/search?q=CJ%20Investigates',{waitUntil:'networkidle',timeout:30000})
  await page.locator('.rat-active-card h1',{hasText:'CJ Investigates'}).waitFor({state:'visible',timeout:20000})
  await page.goto(rat+'/search?q=zzzz-no-such-rat-entity-zzzz',{waitUntil:'networkidle',timeout:30000})
  await page.locator('.rat-message').waitFor({state:'visible',timeout:20000})
  const msg=await page.locator('.rat-message').innerText()
  if(!msg.includes('No RAT-visible Thing matched')) throw new Error('no-match message missing')
  const privacy=await page.request.get(dev+'/api/v1/public/entities/rat-catalogue/search?q=RAT%20Private%20Person%20Sentinel',{headers:{Accept:'application/json'}})
  if(privacy.status()!==200) throw new Error('privacy probe HTTP '+privacy.status())
  const privacyJson=await privacy.json()
  if(privacyJson?.meta?.privacy!=='private-rum-people-excluded') throw new Error('privacy boundary marker missing')
  console.log('RAT_DJI_CJ_BROWSER_VERIFIED')
  console.log('DJI_LINKED_NAVIGATION_VERIFIED')
  console.log('RAT_RATE_FLOW_VERIFIED')
  console.log('RAT_NO_MATCH_VERIFIED')
  console.log('RAT_PRIVACY_MARKER_VERIFIED')
} finally { await browser.close() }
NODE
