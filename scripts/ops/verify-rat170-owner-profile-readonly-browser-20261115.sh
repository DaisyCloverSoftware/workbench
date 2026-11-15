#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only on The Professional Idiots: never submit any mutation, create
# an account, enter moderation, or alter an existing revision or rating.
if [[ $# -gt 0 && "$1" == image ]]; then
  [[ $# -eq 3 ]] || exit 64
  id="$2"; name="$3"
  [[ "$id" =~ ^[a-z0-9-]{8,42}$ && "$name" =~ ^[0-9]{2}-[a-z0-9-]+$ ]] || exit 64
  file="/tmp/rat170-owner-readonly-$id/$name.jpg"
  [[ -f "$file" && ! -L "$file" ]] || exit 66
  echo "RAT170_OWNER_READONLY_SCREENSHOT_STAGE=$name"
  printf 'RAT170_OWNER_READONLY_SCREENSHOT_BASE64='; base64 -w0 "$file"; printf '\n'
  exit 0
fi
[[ $# -eq 2 ]] || exit 64
source="$1"; runid="$2"
[[ "$source" =~ ^[0-9a-f]{40}$ && "$runid" =~ ^[a-z0-9-]{8,42}$ ]] || exit 64
dir="/tmp/rat170-owner-readonly-$runid"
[[ ! -e "$dir" ]] || { echo 'Read-only report ID already exists: inspect existing evidence instead'; exit 65; }
mkdir -m 700 "$dir"
tmp="$(mktemp -d)";trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q
git -C "$tmp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$tmp" fetch -q --depth=1 origin "$source"
git -C "$tmp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$tmp" rev-parse HEAD)" == "$source" ]] || exit 78
cd "$tmp/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >/dev/null
chrome=''
for b in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser; do
  if [[ -x "$b" ]]; then chrome="$b";break;fi
done
[[ -n "$chrome" ]] || npx playwright install chromium >/dev/null
RAT170_OWNER_SOURCE="$source" RAT170_OWNER_RUN="$runid" RAT170_OWNER_OUT="$dir" CHROME="$chrome" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import { writeFile,stat } from 'node:fs/promises'
import { chromium, expect } from '@playwright/test'
const RAT='https://dev-rum-ra.daisycloversoftware.uk',DEV='https://dev-rum.daisycloversoftware.uk'
const ownerId='01m7bswfh0a3evf271ye4taaf0'
const sha=process.env.RAT170_OWNER_SOURCE,run=process.env.RAT170_OWNER_RUN,out=process.env.RAT170_OWNER_OUT
const report={source:sha,runId:run,ownerThingId:ownerId,status:'UNVERIFIED',checks:[],screenshots:[],blockedMutations:[],errors:[]}
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
const context=await browser.newContext({viewport:{width:1440,height:960}})
const page=await context.newPage()
const mark=x=>{report.checks.push(x);console.log('RAT170_OWNER_READONLY_PASS='+x)}
let stage='version'
const shot=async name=>{
  const path=out+'/'+name+'.jpg'
  await page.screenshot({path,type:'jpeg',quality:48,animations:'disabled',fullPage:true})
  const meta=await stat(path)
  report.screenshots.push({stage:name,bytes:meta.size,path})
  console.log('RAT170_OWNER_READONLY_SCREENSHOT='+name+' bytes='+meta.size)
}
await page.route('**/*',route=>{
 const req=route.request(),u=new URL(req.url()),method=req.method()
 if(u.hostname==='rateurmate.online'||!['GET','HEAD','OPTIONS'].includes(method)){
  report.blockedMutations.push({method,host:u.hostname,path:u.pathname})
  return route.abort('blockedbyclient')
 }
 return route.continue()
})
try{
 const version=await context.request.get(RAT+'/VERSION',{headers:{'Cache-Control':'no-cache'}})
 assert.equal(version.status(),200);assert.equal((await version.text()).trim(),'rat-candidate+'+sha)
 mark('exact-rat-frontend')
 stage='owner-profile'
 await page.goto(RAT+'/e/owner-review~'+ownerId,{waitUntil:'domcontentloaded',timeout:30000})
 await expect(page.locator('.rat-workspace-header h1')).toHaveText('The Professional Idiots',{timeout:30000})
 const dt=page.locator('.rat-workspace-facts dt').filter({hasText:'Type of Thing'})
 await expect(dt).toHaveCount(1)
 await expect(dt.first().locator('xpath=following-sibling::dd')).toHaveText('YouTube Channel')
 const extras=page.locator('.rat-workspace-facts dt').filter({hasText:'Other classifications'})
 if(await extras.count()){
  const values=(await extras.first().locator('xpath=following-sibling::dd').innerText()).split(',').map(v=>v.trim().toLowerCase())
  assert.ok(!values.includes('youtube channel'),'Primary YouTube Channel classification duplicated')
 }
 mark('professional-idiots-type-youtube-channel-single-classification')
 await expect(page.getByRole('button',{name:'Change profile picture',exact:true})).toBeVisible()
 await expect(page.getByRole('button',{name:/^(Add|Change) cover photo$/})).toBeVisible()
 mark('profile-and-cover-image-actions-obvious')
 await shot('01-owner-desktop')
 stage='owner-mobile'
 await page.setViewportSize({width:390,height:844})
 await expect(page.locator('.rat-workspace-header h1')).toHaveText('The Professional Idiots')
 await expect(dt.first().locator('xpath=following-sibling::dd')).toHaveText('YouTube Channel')
 await expect(page.getByRole('button',{name:'Change profile picture',exact:true})).toBeVisible()
 await shot('02-owner-mobile')
 const overflow=await page.evaluate(()=>document.documentElement.scrollWidth-window.innerWidth)
 assert.ok(overflow<=2,'Owner mobile horizontal overflow: '+overflow)
 mark('owner-mobile-responsive-no-overflow')
 stage='regressions'
 await page.setViewportSize({width:1440,height:960})
 await page.goto(RAT+'/search?q=DJI',{waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-active-card h1').filter({hasText:'DJI'})).toBeVisible({timeout:20000})
 mark('DJI-search-and-detail')
 await page.goto(RAT+'/search?q=CJ%20Investigates',{waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-active-card h1').filter({hasText:'CJ Investigates'})).toBeVisible({timeout:20000})
 mark('CJ-search-and-detail')
 await page.goto(RAT+'/search?q=zzzz-no-such-entity-rat170-'+run,{waitUntil:'domcontentloaded'})
 await expect(page.getByRole('button',{name:'Add a Thing',exact:true})).toBeVisible({timeout:22000})
 mark('no-match-add-a-thing-visible')
 const privacy=await context.request.get(DEV+'/api/v1/public/entities/rat-catalogue/search?q=RAT%20Private%20Person%20Sentinel')
 assert.equal(privacy.status(),200)
 const j=await privacy.json()
 assert.equal(j.meta.privacy,'private-rum-people-excluded')
 mark('private-people-filtered-from-public-catalogue')
 assert.equal(report.blockedMutations.length,0,'Browser attempted a mutating or LIVE request')
 report.status='PASS'
 console.log('RAT170_OWNER_READONLY_BROWSER_PASS=true')
}catch(e){
 report.status='FAILED'
 const msg=String(e).slice(0,1100);report.errors.push({stage,message:msg})
 console.error('RAT170_OWNER_READONLY_FAILED_STAGE='+stage)
 console.error('RAT170_OWNER_READONLY_ERROR='+msg)
 try{await shot('99-failure')}catch{}
 process.exitCode=1
}finally{
 await writeFile(out+'/report.json',JSON.stringify(report,null,2))
 console.log('RAT170_OWNER_READONLY_REPORT='+JSON.stringify(report))
 await browser.close()
}
NODE
