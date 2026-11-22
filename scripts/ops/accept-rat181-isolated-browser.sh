#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 4 && $1 =~ ^[0-9a-f]{40}$ && $2 =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[a-f0-9]{64}$ && $3 =~ ^ghcr\.io/daisycloversoftware/rum-rate-anything@sha256:[a-f0-9]{64}$ && $4 =~ ^qa181[a-z0-9]{8}$ ]] || exit 64
sha=$1; api=$2; rat=$3; tag=$4; ops=$(cd "$(dirname "$0")" && pwd)
k(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ $(k -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}') == "$api" && $(k -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}') == "$rat" ]] || exit 78
[[ $(curl -fsS --max-time 20 https://dev-rum-ra.daisycloversoftware.uk/VERSION) == "rat-candidate+$sha" ]] || exit 78
curl -fsS --max-time 20 https://dev-rum.daisycloversoftware.uk/api/v1/health|python3 -c 'import sys,json; d=json.load(sys.stdin);assert d["status"]=="ok" and sys.argv[1] in d["version"]' "$sha"
t=$(mktemp -d);trap 'rm -rf "$t"' EXIT; git -C "$t" init -q; git -C "$t" remote add origin https://github.com/DaisyCloverSoftware/rum.git; git -C "$t" fetch -q --depth=1 origin "$sha"; git -C "$t" checkout -q --detach FETCH_HEAD; [[ $(git -C "$t" rev-parse HEAD) == "$sha" ]] || exit 78
cd "$t/apps/rate-anything"; cp "$ops/rat181-acceptance-20261122.mjs" ./acceptance.mjs; node --check acceptance.mjs; npm ci --no-audit --no-fund --ignore-scripts >"$t/npm.log" 2>&1; npx playwright install chromium >"$t/browser.log" 2>&1
node acceptance.mjs "$sha" "$tag" "$ops/rat181-fixture-probe-20261122.sh"
