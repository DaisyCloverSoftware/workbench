#!/usr/bin/env bash
set -Eeuo pipefail
# One-shot authenticated acceptance on isolated RAT DEV only.
# Creates at most one real Colin Furze YouTube Channel and one initial QA rating.
# No person rating, no LIVE, no database reset, and no automatic retry.
[[ $# -eq 4 ]] || exit 64
source_sha="$1";api_image="$2";rat_image="$3";run="$4"
[[ "$source_sha" == '061beefd23bd71cf5902ff2a7724ff78acb515f6' ]] || exit 78
[[ "$api_image" == 'ghcr.io/daisycloversoftware/rum-api@sha256:7411cc02eb6568e6c6d228b551683294bfac49a6f89c320078af5550adce5d56' ]] || exit 78
[[ "$rat_image" == 'ghcr.io/daisycloversoftware/rum-rate-anything@sha256:9a33ea315f52d5e14a5ece0e57895aae8d4eb4d7d7e9955110d49fc9676c64d9' ]] || exit 78
[[ "$run" =~ ^rat180-[a-z0-9-]{10,36}$ ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api_image" ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == "$rat_image" ]] || exit 78
[[ "$(curl -fsS --retry 4 --max-time 25 https://dev-rum-ra.daisycloversoftware.uk/VERSION | tr -d '\r\n')" == "rat-candidate+$source_sha" ]] || exit 78
out="/tmp/rat180-colin-auth-$run"
[[ ! -e "$out" ]] || { echo "RAT180_RUN_ALREADY_EXISTS=$run"; exit 65; }
mkdir -m 700 "$out"
work="$(mktemp -d)";trap 'rm -rf "$work"' EXIT
git -C "$work" init -q
git -C "$work" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$work" fetch -q --depth=1 origin "$source_sha"
git -C "$work" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$work" rev-parse HEAD)" == "$source_sha" ]] || exit 78
cd "$work/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >"$work/npm.log" 2>&1 || { tail -n 12 "$work/npm.log"; exit 1; }
npx playwright install chromium >"$work/chrome.log" 2>&1 || { tail -n 12 "$work/chrome.log"; exit 1; }
RAT180_REPORT_DIR="$out" RAT180_RUN="$run" node --input-type=module <<'NODE'
import {chromium,expect} from '@playwright/test'
import assert from 'node:assert/strict'
import {randomBytes,createHash} from 'node:crypto'
import {stat,writeFile,readFile} from 'node:fs/promises'
const RAT='https://dev-rum-ra.daisycloversoftware.uk'
const DEV='https://dev-rum.daisycloversoftware.uk'
const PERSON='01m0mw9hkbynssr7kbc0gk534j'
const CHANNEL='UCp68_FLety0O-n9QU6phsgw'
const CLASS='community_788e6ae3bfcd391d111e3ca31a1332ebf92906d0d0868e7cf7bb'
const dir=process.env.RAT180_REPORT_DIR
const report={status:'NOT_ASSESSED',run:process.env.RAT180_RUN,personId:PERSON,channelId:null,channelIdentifier:CHANNEL,steps:[],postCount:0,screenshots:[],errors:[],accountCreated:false,ratingSubmitted:false}
const pass=(name)=>{report.steps.push(name);console.log('RAT180_AUTH_PASS='+name)}
const screenshot=async(page,name)=>{
 const path=dir+'/'+name+'.jpg'
 await page.screenshot({path,type:'jpeg',quality:70,fullPage:true,animations:'disabled'})
 const bytes=(await stat(path)).size
 const hash=createHash('sha256').update(await readFile(path)).digest('hex')
 assert.ok(bytes>5000&&bytes<950000,'Invalid screenshot size '+name)
 report.screenshots.push({name:name+'.jpg',bytes,sha256:hash})
 console.log('RAT180_AUTH_SCREENSHOT='+name+' bytes='+bytes+' sha256='+hash)
}
let browser=null,stage='preflight'
try{
 browser=await chromium.launch({headless:true,args:['--no-sandbox','--disable-dev-shm-usage']})
 const context=await browser.newContext({viewport:{width:1440,height:940}})
 const page=await context.newPage()
 const pre=await context.request.get(DEV+'/api/v1/public/entities/rat-catalogue/search?q=Colin%20Furze')
 assert.equal(pre.status(),200)
 const before=await pre.json()
 assert.equal(before.data?.active?.id,PERSON)
 assert.equal(before.data?.active?.ratingOwner,'rum')
 const prior=before.meta?.matches?.filter(x=>x.name==='Colin Furze'&&x.entityType==='digital_resource'&&x.classifications?.some(c=>c.primary&&c.key===CLASS))??[]
 assert.ok(prior.length<=1,'Existing channel already duplicated')
 if(prior.length===1){report.channelId=prior[0].id;report.status='ALREADY_EXISTS_NO_WRITE';pass('channel-already-exists-no-creation');}
 else{
  const byIdentifier=await context.request.get(DEV+'/api/v1/public/entities/rat-catalogue/search?q='+encodeURIComponent(CHANNEL))
  assert.equal(byIdentifier.status(),200)
  assert.equal((await byIdentifier.json()).meta.found,false,'Authoritative channel ID already resolves; do not create')
  pass('public-canonical-person-preserved-and-channel-identifier-no-match')
  const writes=[]
  await page.route('**/*',async route=>{
   const req=route.request(),u=new URL(req.url()),method=req.method()
   if(['GET','HEAD','OPTIONS'].includes(method)){
    if(['dev-rum-ra.daisycloversoftware.uk','dev-rum.daisycloversoftware.uk'].includes(u.hostname))return route.continue()
    return route.abort('blockedbyclient')
   }
   const allowed=u.hostname==='dev-rum-ra.daisycloversoftware.uk'&&method==='POST'&&[
    '/api/v1/rate-anything/auth/register',
    '/api/v1/rate-anything/standalone-things',
    '/api/v1/rate-anything/rating',
   ].includes(u.pathname)
   if(!allowed)return route.abort('blockedbyclient')
   if(u.pathname.endsWith('/standalone-things')){
    const body=req.postDataJSON()
    assert.equal(body.entityType,'digital_resource')
    assert.equal(body.name,'Colin Furze')
    assert.deepEqual(body.classificationKeys,[CLASS])
    assert.deepEqual(body.identifiers,[{namespace:'youtube_channel_id',value:CHANNEL}])
    assert.ok(typeof body.idempotencyKey==='string'&&body.idempotencyKey.length>15)
    report.postCount++
    assert.ok(report.postCount<=2,'Only one candidate check + one distinct-Thing confirmation allowed')
    writes.push('create')
   }
   if(u.pathname.endsWith('/rating')){assert.ok(report.channelId,'Never rate an unknown entity')}
   return route.continue()
  })
  stage='sign-up'
  await page.goto(RAT+'/search?q=Colin%20Furze&active='+PERSON,{waitUntil:'domcontentloaded'})
  await expect(page.locator('.rat-active-card h1')).toHaveText('Colin Furze',{timeout:30000})
  await page.getByRole('button',{name:'Add another Thing with this name'}).click()
  const dialog=page.getByRole('dialog',{name:'Add a Thing'})
  await expect(dialog).toBeVisible()
  await dialog.getByRole('button',{name:'Create account',exact:true}).click()
  const seed=randomBytes(6).toString('hex')
  const username='rat180qa'+seed
  const password=randomBytes(24).toString('base64url')+'aA1!'
  // Keep QA recovery credentials exclusively in a private runner-local file,
  // never in GitHub, terminal output, reports or screenshot metadata.
  await writeFile(dir+'/qa-account-private.json',JSON.stringify({username,email:username+'@example.com',password}),{mode:0o600})
  await dialog.getByLabel('Email',{exact:true}).fill(username+'@example.com')
  await dialog.getByLabel('Username',{exact:true}).fill(username)
  await dialog.getByLabel('Password',{exact:true}).fill(password)
  await dialog.getByLabel('Confirm password').fill(password)
  await dialog.getByLabel('I accept the Terms and Community Rules.').check()
  await dialog.getByLabel('I have read the Privacy Notice.').check()
  const register=page.waitForResponse(r=>new URL(r.url()).pathname==='/api/v1/rate-anything/auth/register'&&r.request().method()==='POST',{timeout:30000})
  await dialog.getByRole('button',{name:'Create RAT account and continue'}).click()
  const auth=await register
  assert.ok([200,201].includes(auth.status()),'QA account registration status '+auth.status())
  report.accountCreated=true
  await expect(dialog.locator('.contribution-member')).toContainText('@'+username,{timeout:20000})
  pass('isolated-dev-test-account-authenticated')
  stage='choose-canonical-type'
  const picker=dialog.getByRole('combobox',{name:'What kind of thing is it?'})
  await picker.fill('YouTube Channel')
  const option=dialog.getByRole('button',{name:/^YouTube Channel/}).first()
  await expect(option).toBeVisible({timeout:30000})
  await option.click()
  await expect(dialog.getByText('Selected: YouTube Channel')).toBeVisible()
  await dialog.getByLabel(/YouTube channel ID/).fill(CHANNEL)
  await dialog.getByLabel('Thing name').fill('Colin Furze')
  await dialog.getByLabel('Thing description').fill('Official Colin Furze YouTube channel, separate from the public creator persona.')
  await dialog.getByLabel('Optional supporting media URL').fill('https://www.youtube.com/channel/'+CHANNEL)
  pass('canonical-existing-youtube-channel-type-selected-exact-name-and-id')
  stage='create-distinct-channel'
  const predicate=r=>new URL(r.url()).pathname==='/api/v1/rate-anything/standalone-things'&&r.request().method()==='POST'
  const firstResponse=page.waitForResponse(predicate,{timeout:45000})
  await dialog.getByRole('button',{name:'Check and add'}).click()
  let creation=await firstResponse
  if(creation.status()===409){
   await expect(dialog.getByRole('button',{name:'No — this is a different thing'})).toBeVisible({timeout:25000})
   const nextResponse=page.waitForResponse(predicate,{timeout:45000})
   await dialog.getByRole('button',{name:'No — this is a different thing'}).click()
   creation=await nextResponse
  }
  const result=await creation.json()
  assert.equal(creation.status(),201,'Channel creation did not succeed: '+JSON.stringify(result).slice(0,500))
  assert.equal(result.meta?.created,true)
  assert.equal(result.data?.name,'Colin Furze')
  assert.equal(result.data?.entityType,'digital_resource')
  assert.ok(result.data?.id&&result.data.id!==PERSON)
  assert.equal(result.data?.publicationState,'pending')
  report.channelId=result.data.id
  pass('one-distinct-canonical-channel-created-pending-with-verified-submission-id')
  await screenshot(page,'colin-channel-pending-after-create')
  stage='initial-rating'
  const flow=page.locator('.rat-rating-flow')
  await expect(flow).toBeVisible({timeout:35000})
  await expect(flow.getByRole('radio',{name:/^3: Good/})).toBeEnabled({timeout:30000})
  await flow.getByRole('radio',{name:/^3: Good/}).click()
  await flow.getByRole('button',{name:'Continue'}).click()
  const category=flow.locator('.rat-rating-choice-grid button').first()
  await expect(category).toBeVisible({timeout:30000})
  await category.click()
  await flow.getByRole('button',{name:'Continue'}).click()
  await flow.getByLabel('Why this verdict?').fill('DEV acceptance: identified the official Colin Furze channel independently from the creator person.')
  await flow.getByRole('button',{name:'Continue'}).click()
  const rateResp=page.waitForResponse(r=>new URL(r.url()).pathname==='/api/v1/rate-anything/rating'&&r.request().method()==='POST',{timeout:45000})
  await flow.getByRole('button',{name:'Submit rating'}).click()
  const rated=await rateResp
  assert.ok([200,201].includes(rated.status()),'Initial channel-only QA rating failed '+rated.status())
  const ratingBody=await rated.json()
  assert.equal(ratingBody.data?.targetThingId??ratingBody.data?.targetEntityId??report.channelId,report.channelId)
  report.ratingSubmitted=true
  await expect(flow).toContainText('Rating saved for Colin Furze',{timeout:30000})
  pass('channel-only-first-rating-submitted-no-person-rating-post')
  await screenshot(page,'colin-channel-initial-rating-complete')
  stage='canonical-after'
  const afterRes=await context.request.get(DEV+'/api/v1/public/entities/rat-catalogue/search?q=Colin%20Furze')
  assert.equal(afterRes.status(),200)
  const after=await afterRes.json()
  assert.equal(after.data?.active?.id,PERSON)
  const named=after.meta?.matches??[]
  assert.ok(named.some(x=>x.id===PERSON&&x.ratingOwner==='rum'))
  assert.ok(named.some(x=>x.id===report.channelId&&x.ratingOwner==='rat'&&x.name==='Colin Furze'&&x.classifications?.some(c=>c.primary&&c.key===CLASS)),'Persisted canonical channel not publicly discoverable')
  pass('both-same-name-canonical-ids-now-publicly-discoverable')
  report.status='PASS_CREATED_AND_PUBLISHED'
 }
}catch(e){
 report.status='FAIL'
 report.errors.push({stage,message:String(e).slice(0,1700)})
 console.log('RAT180_AUTH_ERROR='+JSON.stringify(report.errors.at(-1)))
 process.exitCode=1
}finally{
 await writeFile(dir+'/report.json',JSON.stringify(report,null,2))
 console.log('RAT180_AUTH_REPORT='+JSON.stringify(report))
 if(browser)await browser.close()
}
NODE
