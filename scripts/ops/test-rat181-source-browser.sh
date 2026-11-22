#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 1 && $1 =~ ^qa181[a-z0-9]{8}$ ]]||exit 64
sha=b135949c8d97072cca3b24c2611984ae6d6fc30c;t=$(mktemp -d);trap 'rm -rf "$t"' EXIT;src=$(cd "$(dirname "$0")" && pwd);cp "$src"/{rat181-acceptance-20261122.mjs,rat181-fixture-loopback.mjs,rat181-fixture-probe-20261122.sh} "$t"/
mkdir "$t/checkout";git -C "$t/checkout" init -q;git -C "$t/checkout" remote add origin https://github.com/DaisyCloverSoftware/rum.git;git -C "$t/checkout" fetch --depth=1 -q origin "$sha";git -C "$t/checkout" checkout -q --detach FETCH_HEAD;[[ $(git -C "$t/checkout" rev-parse HEAD) == "$sha" ]]||exit 78
(cd "$t/checkout/apps/rate-anything";npm ci --no-audit --no-fund --ignore-scripts >"$t/npm" 2>&1;npm run build >"$t/build" 2>&1||{ tail -20 "$t/build";exit 1; });printf 'source-test+%s\n' "$sha" >"$t/checkout/apps/rate-anything/dist/VERSION"
# Reuse the same assertions and pod-isolation checks. Only the source of API code/static assets changes; this operation does not publish or deploy a candidate.
python3 - "$src/verify-rat181-disposable-image-browser.sh" "$t" <<'PY'
from pathlib import Path
import sys
s=Path(sys.argv[1]).read_text();t=Path(sys.argv[2]);s=s[:s.index('[[ $(k -n "$ns" get deployment rum-rate-anything')]+s[s.index('# No application secrets'):];s=s.replace('ghcr\\.io/daisycloversoftware/rum-rate-anything@sha256:', 'ghcr\\.io/daisycloversoftware/rum-api@sha256:');s=s.replace('["php","artisan","serve","--host=127.0.0.1","--port=18081","--no-reload"]','["sleep","900"]');s=s.replace('"SESSION_DRIVER":"file"','"SESSION_DRIVER":"file","HOME":"/tmp","COMPOSER_HOME":"/tmp/composer","MODERATION_EVIDENCE_KEY":"base64:"+base64.b64encode(secrets.token_bytes(32)).decode(),"MODERATION_AUDIT_IP_HASH_KEY":"base64:"+base64.b64encode(secrets.token_bytes(32)).decode()');s=s.replace('cs[-1].update(', 'cs[0]["volumeMounts"]=[{"name":"work","mountPath":"/work"}];cs[0]["workingDir"]="/work";cs[-1].update(');s=s.replace('"volumes":[{"name":"db"','"volumes":[{"name":"work","emptyDir":{"sizeLimit":"2Gi"}},{"name":"db"');s=s.replace('k -n "$ns" exec "$pod" -c api -- sh -ec', '''git -C "$ops/checkout" archive --format=tar HEAD:apps/api | k -n "$ns" exec -i "$pod" -c api -- tar -xf - -C /work --no-same-owner
k -n "$ns" exec "$pod" -c api -- sh -ec 'composer install --no-interaction --prefer-dist --no-progress >/tmp/install.log 2>&1 || { tail -12 /tmp/install.log;exit 1; }'
k -n "$ns" exec "$pod" -c api -- sh -ec''',1);i=s.index('mkdir "$t/assets"');j=s.index('k -n "$ns" port-forward',i);s=s[:i]+'''cp -r "$ops/checkout/apps/rate-anything/dist" "$t/assets"; k -n "$ns" exec "$pod" -c api -- sh -c 'php artisan serve --host=127.0.0.1 --port=18081 --no-reload >/tmp/serve.log 2>&1' & apipid=$!
'''+s[j:];s=s.replace("'rat-candidate+'+sha","'source-test+'+sha");s=s.replace('DISPOSABLE_DEPLOYED_IMAGE_BROWSER_PASS','DISPOSABLE_SOURCE_BROWSER_PASS');(t/'loopback.sh').write_text(s)
PY
runtime=ghcr.io/daisycloversoftware/rum-api@sha256:f4e700314dd2fe2b8b6dabee5f640f35b9463d914f8a2be79ff781862a80e978
bash "$t/loopback.sh" "$sha" "$runtime" "$runtime" "$1"
