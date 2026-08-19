#!/usr/bin/env bash
set -euo pipefail

if [ "${1:-}" = "--self-test" ]; then
  [ "$#" -eq 1 ]
  printf 'CLUSTER_HEALTH_SELF_TEST_OK\n'
  exit 0
fi

if [ "$#" -ne 0 ]; then
  echo "Usage: cluster-health.sh [--self-test]" >&2
  exit 2
fi

if sudo -n kubectl version --client >/dev/null 2>&1; then
  KUBECTL=(sudo -n kubectl)
elif sudo -n k3s kubectl version --client >/dev/null 2>&1; then
  KUBECTL=(sudo -n k3s kubectl)
else
  echo "ERROR: no sanctioned non-interactive Kubernetes client is available" >&2
  exit 1
fi

printf 'CLUSTER_HEALTH\n'
printf 'host=%s\n' "$(hostname)"

printf '[nodes]\n'
"${KUBECTL[@]}" get nodes -o wide

printf '[abnormal-pods]\n'
pods="$("${KUBECTL[@]}" get pods -A --no-headers \
  -o 'custom-columns=NAMESPACE:.metadata.namespace,NAME:.metadata.name,PHASE:.status.phase,READY:.status.containerStatuses[*].ready,RESTARTS:.status.containerStatuses[*].restartCount,NODE:.spec.nodeName')"
abnormal="$(printf '%s\n' "$pods" | awk '
  $3 != "Running" && $3 != "Succeeded" { print; next }
  $4 ~ /false/ { print; next }
  $5 !~ /^(0|0,0|0,0,0|0,0,0,0)$/ { print; next }
')"
if [ -n "$abnormal" ]; then
  printf '%s\n' "$abnormal" | head -n 40
else
  printf 'none\n'
fi

printf '[recent-warnings]\n'
warnings="$("${KUBECTL[@]}" get events -A --field-selector type=Warning --sort-by=.lastTimestamp --no-headers \
  -o 'custom-columns=LAST:.lastTimestamp,NAMESPACE:.metadata.namespace,REASON:.reason,KIND:.involvedObject.kind,OBJECT:.involvedObject.name,MESSAGE:.message' 2>&1 || true)"
if [ -n "$warnings" ]; then
  printf '%s\n' "$warnings" | tail -n 20
else
  printf 'none\n'
fi

api_resources="$("${KUBECTL[@]}" api-resources -o name 2>/dev/null || true)"

printf '[arc-runners]\n'
if grep -qx 'ephemeralrunners.actions.github.com' <<<"$api_resources"; then
  arc="$("${KUBECTL[@]}" get ephemeralrunners.actions.github.com -A -o wide 2>&1 || true)"
  if [ -n "$arc" ]; then
    printf '%s\n' "$arc" | head -n 40
  else
    printf 'none\n'
  fi
else
  printf 'not-installed\n'
fi

printf '[longhorn-nodes]\n'
if grep -qx 'nodes.longhorn.io' <<<"$api_resources"; then
  longhorn_nodes="$("${KUBECTL[@]}" get nodes.longhorn.io -n longhorn-system --no-headers \
    -o 'custom-columns=NAME:.metadata.name,ALLOW_SCHEDULING:.spec.allowScheduling,EVICTION:.spec.evictionRequested' 2>&1 || true)"
  if [ -n "$longhorn_nodes" ]; then
    printf '%s\n' "$longhorn_nodes"
  else
    printf 'none\n'
  fi
else
  printf 'not-installed\n'
fi

printf '[longhorn-volumes]\n'
if grep -qx 'volumes.longhorn.io' <<<"$api_resources"; then
  longhorn_volumes="$("${KUBECTL[@]}" get volumes.longhorn.io -n longhorn-system --no-headers \
    -o 'custom-columns=NAME:.metadata.name,STATE:.status.state,ROBUSTNESS:.status.robustness,NODE:.status.currentNodeID,SIZE:.spec.size' 2>&1 || true)"
  if [ -n "$longhorn_volumes" ]; then
    printf '%s\n' "$longhorn_volumes" | head -n 60
  else
    printf 'none\n'
  fi
else
  printf 'not-installed\n'
fi

printf 'snapshot=ok\n'
