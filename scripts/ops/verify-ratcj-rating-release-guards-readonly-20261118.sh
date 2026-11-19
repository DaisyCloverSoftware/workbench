#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only syntax and current isolated deployment baseline. No release operation performed.
[[ $# -eq 0 ]] || exit 64
root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
for file in \
 "$root/deploy-ratcj-public-rating-contract-isolated-20261118.sh" \
 "$root/verify-ratcj-public-rating-3width-chromium-20261118.sh" \
 "$root/export-ratcj-final-rating-3width-screenshot-20261118.sh" \
 "$root/stage-ratcj-independent-accepted-card-qa-20261119.sh" \
 "$root/retire-ratcj-independent-accepted-card-qa-20261119.sh" \
 "$root/verify-ratcj-independent-accepted-qa-browser-20261119.sh" \
 "$root/export-ratcj-accepted-qa-3width-screenshot-20261119.sh";do
 [[ -f "$file" && ! -L "$file" ]] || exit 78
 bash -n "$file"
done
echo RATCJ_DEPLOY_AND_BROWSER_SCRIPTS_SYNTAX_PASS=true
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
image(){ kctl -n "$1" get deployment "$2" -o "jsonpath={.spec.template.spec.containers[?(@.name==\"$3\")].image}"; }
api="$(image rum-dev-isolated rum-api php-fpm)"
rat="$(image rum-rate-anything-preview rum-rate-anything rate-anything)"
[[ "$api" == 'ghcr.io/daisycloversoftware/rum-api@sha256:a23e36323b55f3bce46d5b229146e2e7a9d70bdff9e16ea366380940e0629430' ]] || { echo "API drift=$api"; exit 78; }
[[ "$rat" == 'ghcr.io/daisycloversoftware/rum-rate-anything@sha256:ec11bfec432ff10d508850a0a071b3af3787bff656d2decca90eed5951cd0003' ]] || { echo "RAT drift=$rat"; exit 78; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.status.readyReplicas}')" == '1' ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.status.readyReplicas}')" == '1' ]] || exit 78
for pair in 'rum-dev:rateurmate.online' 'rum-dev-isolated:dev-rum.daisycloversoftware.uk' 'rum-rate-anything-preview:dev-rum-ra.daisycloversoftware.uk';do
 ns="${pair%%:*}";host="${pair#*:}"
 actual="$(kctl -n "$ns" get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)"
 [[ "$actual" == "$host" ]] || { echo "Unexpected ingress $ns: $actual"; exit 78; }
done
echo "RATCJ_RELEASE_SAFE_BASELINE_API=$api"
echo "RATCJ_RELEASE_SAFE_BASELINE_RAT=$rat"
echo RATCJ_ISOLATED_RELEASE_PREFLIGHT_PASS=true
