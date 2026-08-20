#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: $0 <namespace> <pod>" >&2
  exit 2
fi
NS="$1"
POD="$2"
K=(sudo -n k3s kubectl)

"${K[@]}" get nodes -o custom-columns='NAME:.metadata.name,INTERNAL:.status.addresses[?(@.type=="InternalIP")].address,PODCIDR:.spec.podCIDR' --no-headers
printf '%s\n' '--- resolv.conf ---'
"${K[@]}" exec -n "$NS" "$POD" -c runner -- cat /etc/resolv.conf
printf '%s\n' '--- DNS github ---'
"${K[@]}" exec -n "$NS" "$POD" -c runner -- getent ahostsv4 github.com || true
printf '%s\n' '--- DNS actions endpoint ---'
"${K[@]}" exec -n "$NS" "$POD" -c runner -- getent ahostsv4 pipelines.actions.githubusercontent.com || true
printf '%s\n' '--- service DNS IP route ---'
"${K[@]}" exec -n "$NS" "$POD" -c runner -- sh -c 'ip route 2>/dev/null || true' || true
printf '%s\n' '--- HTTPS github ---'
"${K[@]}" exec -n "$NS" "$POD" -c runner -- curl -fsSI --connect-timeout 5 https://github.com/ | head -5 || true
printf '%s\n' '--- HTTPS api by name ---'
"${K[@]}" exec -n "$NS" "$POD" -c runner -- curl -fsSI --connect-timeout 5 https://api.github.com/ | head -5 || true
