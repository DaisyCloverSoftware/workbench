#!/usr/bin/env bash
set -euo pipefail
umask 077

DEV_URL="https://dev.family-vault.co.uk"
IMAGE="mcr.microsoft.com/playwright:v1.60.0-noble"
TMP="$(mktemp -d)"
cleanup(){ rm -rf "$TMP"; }
trap cleanup EXIT HUP INT TERM

cat >"$TMP/package.json" <<'JSON'
{"private":true,"devDependencies":{"@playwright/test":"1.60.0"}}
JSON
cat >"$TMP/verify.spec.js" <<'JS'
const { test, expect } = require('@playwright/test');
const base = 'https://dev.family-vault.co.uk';

test.use({ viewport: { width: 1440, height: 1000 }, ignoreHTTPSErrors: true });
test('deployed Round 2 FAQ geometry', async ({ page }) => {
  await page.goto(base + '/', { waitUntil: 'domcontentloaded' });
  const beta = page.getByRole('dialog', { name: 'Family Vault Beta Notice' });
  if (await beta.isVisible().catch(() => false)) {
    await beta.getByRole('checkbox').check();
    await beta.getByRole('button', { name: 'Continue' }).click();
  }
  const health = await page.request.get(base + '/api/health');
  expect(health.ok()).toBeTruthy();
  const items = page.locator('#faq .faq-item');
  await expect(items).toHaveCount(5);
  const expectedSlots = ['primary-left','primary-right','storage','design-heritage','audience'];
  for (let i=0;i<5;i++) await expect(items.nth(i)).toHaveAttribute('data-faq-slot', expectedSlots[i]);
  const b = await Promise.all([...Array(5)].map((_,i)=>items.nth(i).boundingBox()));
  if (b.some(x=>!x)) throw new Error('FAQ bounding box missing');
  const [a,p,s,d,u]=b;
  expect(Math.abs(a.y-p.y)).toBeLessThanOrEqual(2);
  expect(Math.abs(a.width-p.width)).toBeLessThanOrEqual(4);
  expect(Math.abs(s.x-u.x)).toBeLessThanOrEqual(2);
  expect(Math.abs(s.width-u.width)).toBeLessThanOrEqual(4);
  expect(u.y).toBeGreaterThan(s.y+s.height);
  expect(d.x).toBeGreaterThan(s.x+s.width);
  expect(Math.abs(d.y-s.y)).toBeLessThanOrEqual(2);
  expect(Math.abs((d.y+d.height)-(u.y+u.height))).toBeLessThanOrEqual(4);
  await expect(items.nth(4)).not.toHaveAttribute('data-faq-wide', /.+/);
  const visible = await items.evaluateAll(nodes=>nodes.map(n=>({question:n.querySelector('h3')?.textContent?.trim()||'',answer:n.querySelector('p')?.textContent?.trim()||''})));
  const structured = await page.locator('script[type="application/ld+json"]').evaluateAll(scripts=>{
    for(const s of scripts){const v=JSON.parse(s.textContent||'{}'); const faq=(v['@graph']||[]).find(e=>e['@type']==='FAQPage'); if(faq) return faq.mainEntity.map(e=>({question:e.name,answer:e.acceptedAnswer.text}));} return [];
  });
  expect(structured).toEqual(visible);
  console.log('FAMILY_VAULT_DEV_FAQ_GEOMETRY=' + JSON.stringify(b));
  console.log('FAMILY_VAULT_DEV_FAQ_JSONLD_MATCH=true');
  console.log('FAMILY_VAULT_DEV_HEALTH=' + JSON.stringify(await health.json()));
});
JS

docker run --rm --network host --ipc=host --init --user root \
  -v "$TMP:/work" -w /work "$IMAGE" \
  bash -lc 'npm install --silent --no-audit --no-fund && npx playwright test verify.spec.js --reporter=line'

echo "FAMILY_VAULT_DEV_ROUND2_PUBLIC_BROWSER_PASS=true"
