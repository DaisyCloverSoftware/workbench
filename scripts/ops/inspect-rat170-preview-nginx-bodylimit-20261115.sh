#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# -eq 0 ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
[[ "$(kctl -n rum-rate-anything-preview get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}'|sort -u)" == 'dev-rum-ra.daisycloversoftware.uk' ]] || exit 78
conf="$(kctl -n rum-rate-anything-preview exec deployment/rum-rate-anything -c rate-anything -- nginx -T 2>&1)"
grep -q 'server_name dev-rum-ra.daisycloversoftware.uk;' <<<"$conf" || exit 78
if grep -Eq '^[[:space:]]*client_max_body_size[[:space:]]+' <<<"$conf"; then
 echo 'RAT170_RUNTIME_NGINX_BODY_SIZE_DIRECTIVES_PRESENT=true'
 grep -E '^[[:space:]]*client_max_body_size[[:space:]]+' <<<"$conf"
else
 echo 'RAT170_RUNTIME_NGINX_BODY_SIZE_DIRECTIVES_PRESENT=false'
 echo 'RAT170_NGINX_UNCONFIGURED_DEFAULT_1MiB=true'
fi
echo 'RAT170_RUNTIME_NGINX_CONFIG_READONLY=true'
