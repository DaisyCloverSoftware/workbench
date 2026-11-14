#!/usr/bin/env bash
set -euo pipefail
# Isolated RAT only. Verify the REAL persisted YouTube Channel catalogue option
# can be selected and reselected with keyboard in owner-form search context.
# No type write, Thing write, migration, seed, reset, deployment or LIVE action.
[[ $# -ge 1 ]] || exit 64
mode="$1"; shift
root='/tmp/rum-rat169-youtube-existing-reuse-20261114'
case "$mode" in
image)
  [[ $# -eq 2 ]] || exit 64
  rid="$1"; stage="$2"
  [[ "$rid" =~ ^[a-z0-9-]{8,42}$ && "$stage" =~ ^[0-9]{2}-[a-z-]+$ ]] || exit 64
  f="$root/$rid/$stage.jpg"
  [[ -f "$f" && ! -L "$f" ]] || exit 66
  echo "RAT169_REUSE_IMAGE_STAGE=$stage"
  echo "RAT169_REUSE_IMAGE_BYTES=$(wc -c < "$f")"
  printf 'RAT169_REUSE_IMAGE_BASE64='
  base64 -w0 "$f"; printf '\n'
  exit 0
  ;;
run)
  [[ $# -eq 2 ]] || exit 64
  source="$1"; rid="$2"
  [[ "$source" =~ ^[0-9a-f]{40}$ && "$rid" =~ ^[a-z0-9-]{8,42}$ ]] || exit 64
  mkdir -p "$root"; chmod 700 "$root"
  dir="$root/$rid"; [[ ! -e "$dir" ]] || { echo 'Previous evidence exists: do not rerun'; exit 65; }
  mkdir -m 700 "$dir"
  scratch="$(mktemp -d)"; trap 'rm -rf "$scratch"' EXIT
  git -C "$scratch" init -q
  git -C "$scratch" remote add origin https://github.com/DaisyCloverSoftware/rum.git
  git -C "$scratch" fetch -q --depth=1 origin "$source"
  git -C "$scratch" checkout -q --detach FETCH_HEAD
  [[ "$(git -C "$scratch" rev-parse HEAD)" == "$source" ]] || exit 78
  cd "$scratch/apps/rate-anything"
  npm ci --ignore-scripts --no-audit --no-fund >/dev/null
  chrome=""
  for p in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser; do
    if [[ -x "$p" ]]; then chrome="$p"; break; fi
  done
  [[ -n "$chrome" ]] || npx playwright install chromium >/dev/null
  SOURCE="$source" RID="$rid" OUT="$dir" CHROMIUM_PATH="$chrome" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import { randomBytes } from 'node:crypto'
import { writeFile } from 'node:fs/promises'
import { chromium, expect } from '@playwright/test'
const rat='https://dev-rum-ra.daisycloversoftware.uk', dev='https://dev-rum.daisycloversoftware.uk'
const source=process.env.SOURCE, out=process.env.OUT, rid=process.env.RID
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROMIUM_PATH||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
const context=await browser.newContext({viewport:{width:1100,height:900}})
const page=await context.newPage()
const report={source,runId:rid,result:'INCOMPLETE',screenshots:[],checks:[],blockedWrites:[],errors:[],typePosts:0,thingPosts:0}
const name='The Professional Idiots',label='YouTube Channel',description='This is a read-only type-reuse acceptance form; do not create the Thing.'
const note=(name)=>{report.checks.push(name);console.log('RAT169_REUSE_PASS='+name)}
const shot=async name=>{await page.getByRole('dialog',{name:'Add a Thing',exact:true}).screenshot({path:out+'/'+name+'.jpg',type:'jpeg',quality:45,animations:'disabled'});report.screenshots.push(name);console.log('RAT169_REUSE_SCREENSHOT='+name)}
const normalize=s=>s.normalize('NFC').toLowerCase().replace(/\s+/g,' ').trim()
let stage='preflight'
await context.route('**/*',route=>{
  let u
  try{u=new URL(route.request().url())}catch{return route.continue()}
  const method=route.request().method()
  if(u.hostname==='rateurmate.online')return route.abort('blockedbyclient')
  if(!['GET','HEAD','OPTIONS'].includes(method)&&![rat,dev].includes(u.origin))return route.abort('blockedbyclient')
  if(method==='POST'&&u.pathname==='/api/v1/rate-anything/thing-types'){
    report.typePosts+=1;report.blockedWrites.push('unexpected type POST');return route.abort('blockedbyclient')
  }
  if(method==='POST'&&u.pathname==='/api/v1/rate-anything/standalone-things'){
    report.thingPosts+=1;report.blockedWrites.push('unexpected Thing POST');return route.abort('blockedbyclient')
  }
  return route.continue()
})
try{
  const version=await context.request.get(rat+'/VERSION',{headers:{'Cache-Control':'no-cache'}})
  assert.equal(version.status(),200)
  assert.equal((await version.text()).trim(),'rat-candidate+'+source)
  const health=await context.request.get(dev+'/api/v1/health')
  assert.equal(health.status(),200)
  assert.ok(String((await health.json()).version).includes(source))
  note('served-versions-match-source')
  stage='owner-no-match'
  await page.goto(rat+'/search?q='+encodeURIComponent(name),{waitUntil:'domcontentloaded'})
  await expect(page.getByRole('button',{name:'Add a Thing',exact:true})).toBeVisible({timeout:25000})
  await page.getByRole('button',{name:'Add a Thing',exact:true}).click()
  const dialog=page.getByRole('dialog',{name:'Add a Thing',exact:true})
  await expect(dialog).toBeVisible()
  stage='auth'
  await dialog.getByRole('button',{name:'Create account',exact:true}).click()
  const username='ratreuse'+rid.replace(/[^a-z0-9]/g,'').slice(-12)
  const password=randomBytes(20).toString('base64url')+'Aa1!'
  await dialog.getByLabel('Email',{exact:true}).fill(username+'@example.com')
  await dialog.getByLabel('Username',{exact:true}).fill(username)
  await dialog.getByLabel('Password',{exact:true}).fill(password)
  await dialog.getByLabel('Confirm password').fill(password)
  await dialog.getByRole('checkbox',{name:/Terms and Community Rules/}).check()
  await dialog.getByRole('checkbox',{name:/Privacy Notice/}).check()
  await dialog.getByRole('button',{name:'Create RAT account and continue'}).click()
  await expect(dialog.locator('.contribution-member')).toBeVisible({timeout:25000})
  await expect(dialog.getByLabel('Thing name',{exact:true})).toHaveValue(name)
  await dialog.getByLabel('Thing description').fill(description)
  await dialog.getByLabel('Optional supporting media URL').fill('https://example.org/reuse-only')
  const check=async()=>{
    await expect(dialog.getByLabel('Thing name',{exact:true})).toHaveValue(name)
    await expect(dialog.getByLabel('Thing description')).toHaveValue(description)
    await expect(dialog.getByLabel('Optional supporting media URL')).toHaveValue('https://example.org/reuse-only')
  }
  stage='real-server-option'
  const picker=dialog.getByRole('combobox',{name:'What kind of thing is it?'})
  const query='  yOUTuBE   chaNNel '
  const response=page.waitForResponse(r=>new URL(r.url()).pathname==='/api/v1/rate-anything/thing-types'
    && r.request().method()==='GET'&&normalize(new URL(r.url()).searchParams.get('q')||'')===normalize(label))
  await picker.fill(query)
  const fetched=await response
  assert.equal(fetched.status(),200)
  const catalogue=await fetched.json()
  const existing=catalogue.data.find(x=>normalize(x.label)===normalize(label))
  assert.ok(existing?.classificationKey&&existing.entityType==='digital_resource','recognised existing catalogue option missing')
  assert.equal(catalogue.meta.creation.mode,'recognised')
  report.catalogueClassification={id:existing.id,key:existing.classificationKey,kind:existing.entityType,label:existing.label}
  await expect(dialog.getByRole('button',{name:/^YouTube Channel Existing type/})).toBeVisible({timeout:12000})
  await expect(dialog.getByRole('button',{name:/^Add “/})).toHaveCount(0)
  await check()
  await shot('01-existing-type-choice')
  await picker.press('Enter')
  await expect(dialog.getByText('Selected: YouTube Channel',{exact:true})).toBeVisible()
  await check()
  await shot('02-existing-type-selected')
  note('real-server-classification-reused-with-keyboard-without-writes')
  stage='refresh-repeat'
  await dialog.getByRole('button',{name:'Close',exact:true}).click()
  await page.reload({waitUntil:'domcontentloaded'})
  await page.getByRole('button',{name:'Add a Thing',exact:true}).click()
  const later=page.getByRole('dialog',{name:'Add a Thing',exact:true})
  await expect(later.locator('.contribution-member')).toBeVisible({timeout:14000})
  const laterPicker=later.getByRole('combobox',{name:'What kind of thing is it?'})
  const reRequest=page.waitForResponse(r=>new URL(r.url()).pathname==='/api/v1/rate-anything/thing-types'
    && r.request().method()==='GET'&&new URL(r.url()).searchParams.get('q')===label)
  await laterPicker.fill(label)
  const reResp=await reRequest
  assert.equal(reResp.status(),200)
  const now=await reResp.json()
  assert.ok(now.data.some(x=>x.id===existing.id&&x.classificationKey===existing.classificationKey))
  await expect(later.getByRole('button',{name:/^YouTube Channel Existing type/})).toBeVisible()
  await laterPicker.press('Enter')
  await expect(later.getByText('Selected: YouTube Channel',{exact:true})).toBeVisible()
  await expect(later.getByLabel('Thing name',{exact:true})).toHaveValue(name)
  await shot('03-server-reuse-after-refresh')
  note('refresh-reloads-same-canonical-classification')
  assert.equal(report.typePosts,0);assert.equal(report.thingPosts,0)
  report.result='PASS'
}catch(e){
  report.result='FAIL';report.failedStage=stage
  report.errors.push(e instanceof Error?e.stack||e.message:String(e))
  console.error('RAT169_REUSE_FAILED_STAGE='+stage)
  console.error('RAT169_REUSE_ERROR='+String(e).slice(0,2000))
  try{await shot('99-failure')}catch{}
}finally{
  await writeFile(out+'/report.json',JSON.stringify(report,null,2))
  console.log('RAT169_REUSE_REPORT='+JSON.stringify(report))
  await browser.close()
}
if(report.result!=='PASS')process.exit(1)
NODE
;;
*) exit 64;;
esac
