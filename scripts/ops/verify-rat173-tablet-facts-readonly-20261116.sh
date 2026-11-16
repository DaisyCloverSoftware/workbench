#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# -eq 4 ]] || { echo "usage: <exact-source> <api-image> <rat-image> <run-id>" >&2; exit 64; }
src="$1"; api="$2"; rat="$3"; run="$4"
[[ "$src" =~ ^[0-9a-f]{40}$ ]] || exit 64
[[ "$api" =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ ]] || exit 64
[[ "$rat" =~ ^ghcr\.io/daisycloversoftware/rum-rate-anything@sha256:[0-9a-f]{64}$ ]] || exit 64
[[ "$run" =~ ^[a-z0-9-]{10,44}$ ]] || exit 64
out="/tmp/rat173-tablet-$run"
[[ ! -e "$out" ]] || { echo "Duplicate run id"; exit 65; }
mkdir -m 700 "$out"
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api" ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == "$rat" ]] || exit 78
[[ "$(curl -fsS --max-time 20 -H 'Cache-Control: no-cache' https://dev-rum-ra.daisycloversoftware.uk/VERSION | tr -d '\r\n')" == "rat-candidate+$src" ]] || exit 78
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q
git -C "$tmp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$tmp" fetch -q --depth=1 origin "$src"
git -C "$tmp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$tmp" rev-parse HEAD)" == "$src" ]] || exit 78
cd "$tmp/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >/dev/null
chrome=''
for c in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser; do if [[ -x "$c" ]]; then chrome="$c"; break; fi; done
[[ -n "$chrome" ]] || npx playwright install chromium >/dev/null
RAT173_OUT="$out" CHROME="$chrome" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import {chromium,expect} from '@playwright/test'
import {stat} from 'node:fs/promises'
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
const ctx=await browser.newContext({viewport:{width:768,height:920}})
const page=await ctx.newPage()
const out=process.env.RAT173_OUT
try {
 for(const [name,width,expectedCols] of [['tablet-768',768,3],['tablet-720',720,1]]){
  await page.setViewportSize({width,height:920})
  await page.goto('https://dev-rum-ra.daisycloversoftware.uk/e/the-professional-idiots~01m7bswfh0a3evf271ye4taaf0',{waitUntil:'domcontentloaded'})
  await expect(page.locator('.rat-workspace-header h1')).toContainText('The Professional Idiots',{timeout:30000})
  const facts=page.locator('.rat-workspace-profile > .rat-workspace-facts')
  await expect(facts.locator(':scope > div')).toHaveCount(3,{timeout:20000})
  const labels=await facts.locator('dt').allTextContents()
  assert.deepEqual(labels.map(x=>x.trim()),['Type of Thing','Rateability','Canonical ID'])
  await expect(facts.locator('.rat-workspace-id')).toContainText('01m7bswfh0a3evf271ye4taaf0')
  const measures=await facts.evaluate(el=>{
   const boxes=[...el.querySelectorAll(':scope > div')].map(x=>x.getBoundingClientRect())
   return {columns:getComputedStyle(el).gridTemplateColumns.split(' ').filter(Boolean).length,widths:boxes.map(b=>b.width),idScroll:el.querySelector('.rat-workspace-id').scrollWidth,idWidth:el.querySelector('.rat-workspace-id').clientWidth}
  })
  assert.equal(measures.columns,expectedCols,JSON.stringify(measures))
  if(expectedCols===3)assert.ok(Math.max(...measures.widths)-Math.min(...measures.widths)<3,'unbalanced tablet facts '+JSON.stringify(measures))
  const overflow=await page.evaluate(()=>document.documentElement.scrollWidth-innerWidth)
  assert.ok(overflow<=2,'tablet horizontal overflow '+overflow)
  const filename=out+'/'+name+'.jpg'
  await page.screenshot({path:filename,type:'jpeg',quality:50,fullPage:true,animations:'disabled'})
  const f=await stat(filename)
  console.log('RAT173_TABLET_PASS='+name+' columns='+measures.columns+' overflow='+overflow+' screenshot='+filename+' bytes='+f.size)
 }
 console.log('RAT173_TABLET_READONLY_BROWSER_PASS=true')
}finally{await browser.close()}
NODE
