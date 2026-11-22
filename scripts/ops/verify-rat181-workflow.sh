#!/usr/bin/env bash
set -Eeuo pipefail
mode=${1:-};shift || exit 64
case "$mode" in source) [[ $# == 1 ]]||exit 64; entry=test-rat181-source-browser.sh;; images) [[ $# == 4 ]]||exit 64;entry=verify-rat181-disposable-image-browser.sh;; *)exit 64;;esac
ops=$(cd "$(dirname "$0")" && pwd);t=$(mktemp -d);trap 'rm -rf "$t"' EXIT
for f in test-rat181-source-browser.sh verify-rat181-disposable-image-browser.sh rat181-fixture-probe-20261122.sh rat181-fixture-loopback.mjs rat181-acceptance-20261122.mjs;do cp "$ops/$f" "$t/$f";done
python3 - "$t" "$mode" <<'PY'
from pathlib import Path
import sys
t=Path(sys.argv[1]);p=t/'rat181-acceptance-20261122.mjs';s=p.read_text();old="const option=form.getByRole('option').filter({has:page.getByText(label,{exact:true})}).first();await option.getByRole('button').click();";new="const option=form.locator('li[role=option] > button').filter({hasText:new RegExp('^'+label+'\\\\s*(?:General Thing kind|Existing type)')});await expect(option).toHaveCount(1);await option.click();";assert s.count(old)==1;s=s.replace(old,new)
old="const wr=waitPost(`standalone-things/${c.j}/retry-connection`);await page.getByRole('button',{name:'Retry connection only'}).click();assert.equal((await wr).status(),200);"
new="""const retryOnly=async()=>{const response=waitPost(`standalone-things/${c.j}/retry-connection`);await page.getByRole('button',{name:'Retry connection only'}).click();return await response;};let response=await retryOnly();if(response.status()===429){const serverDelay=response.headers()['retry-after'];const delay=serverDelay===undefined?60:(/^\\d+$/.test(serverDelay)?Number(serverDelay):Math.ceil((Date.parse(serverDelay)-Date.now())/1000));assert.ok(Number.isFinite(delay)&&delay>=0&&delay<=150,'unbounded Retry-After; stopping');console.log('HONOUR_RETRY_AFTER_SECONDS',delay+2);const beforeRetry=db('inspect');edge(beforeRetry,c.id,0);assert.deepEqual(beforeRetry.entities.find(x=>x.id===c.id).ratingIds,original.ratingIds);await new Promise(done=>setTimeout(done,(delay+2)*1000));response=await retryOnly();}assert.equal(response.status(),200);"""
assert s.count(old)==1;p.write_text(s.replace(old,new))
if sys.argv[2]=='images':
 p=t/'verify-rat181-disposable-image-browser.sh';s=p.read_text().replace('"$tag" "$t" &tls=$!', '"$tag" "$t" </dev/null >"$t/tls.log" 2>&1 &tls=$!').replace(':18081 >"$t/pf"',':18081 </dev/null >"$t/pf"').replace('"SESSION_DRIVER":"file"','"SESSION_DRIVER":"file","MODERATION_EVIDENCE_KEY":"base64:"+base64.b64encode(secrets.token_bytes(32)).decode(),"MODERATION_AUDIT_IP_HASH_KEY":"base64:"+base64.b64encode(secrets.token_bytes(32)).decode()');p.write_text(s)
PY
node --check "$t/rat181-acceptance-20261122.mjs";bash -n "$t/$entry";bash "$t/$entry" "$@";echo "RAT181_BROWSER_TASK_COMPLETED=$mode"
