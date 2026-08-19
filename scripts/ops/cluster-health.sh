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

limit_width() {
  awk '{ if (length($0) > 260) print substr($0, 1, 257) "..."; else print }'
}

printf 'CLUSTER_HEALTH\n'
printf 'host=%s\n' "$(hostname)"

printf '[nodes]\n'
"${KUBECTL[@]}" get nodes -o wide

printf '[abnormal-pods]\n'
pods="$("${KUBECTL[@]}" get pods -A --no-headers \
  -o 'custom-columns=NAMESPACE:.metadata.namespace,NAME:.metadata.name,PHASE:.status.phase,READY:.status.containerStatuses[*].ready,NODE:.spec.nodeName')"
abnormal="$(printf '%s\n' "$pods" | awk '
  $3 != "Running" && $3 != "Succeeded" { print; next }
  $3 == "Running" && $4 ~ /false/ { print; next }
')"
if [ -n "$abnormal" ]; then
  printf '%s\n' "$abnormal" | head -n 40 | limit_width
else
  printf 'none\n'
fi

printf '[recent-warnings]\n'
warnings="$("${KUBECTL[@]}" get events -A --field-selector type=Warning --sort-by=.lastTimestamp --no-headers \
  -o 'custom-columns=LAST:.lastTimestamp,NAMESPACE:.metadata.namespace,REASON:.reason,KIND:.involvedObject.kind,OBJECT:.involvedObject.name,MESSAGE:.message' 2>&1 || true)"
if [ -n "$warnings" ]; then
  printf '%s\n' "$warnings" | tail -n 20 | limit_width
else
  printf 'none\n'
fi

api_resources="$("${KUBECTL[@]}" api-resources -o name 2>/dev/null || true)"

printf '[arc-runners]\n'
if grep -qx 'ephemeralrunners.actions.github.com' <<<"$api_resources"; then
  arc="$("${KUBECTL[@]}" get ephemeralrunners.actions.github.com -A --no-headers \
    -o 'custom-columns=NAMESPACE:.metadata.namespace,NAME:.metadata.name,STATUS:.status.phase,REPOSITORY:.status.jobRepositoryName,JOB:.status.jobDisplayName,WORKFLOW_RUN:.status.workflowRunId' 2>&1 || true)"
  if [ -n "$arc" ]; then
    printf '%s\n' "$arc" | head -n 40 | limit_width
  else
    printf 'none\n'
  fi
else
  printf 'not-installed\n'
fi

printf '[longhorn-nodes]\n'
if grep -qx 'nodes.longhorn.io' <<<"$api_resources"; then
  longhorn_nodes="$("${KUBECTL[@]}" get nodes.longhorn.io -n longhorn-system --no-headers \
    -o 'custom-columns=NAME:.metadata.name,ALLOW_SCHEDULING:.spec.allowScheduling,EVICTION:.spec.evictionRequested,CONDITIONS:.status.conditions[*].status' 2>&1 || true)"
  if [ -n "$longhorn_nodes" ]; then
    printf '%s\n' "$longhorn_nodes" | limit_width
  else
    printf 'none\n'
  fi
else
  printf 'not-installed\n'
fi

printf '[longhorn-attached-unhealthy]\n'
if grep -qx 'volumes.longhorn.io' <<<"$api_resources"; then
  longhorn_volumes="$("${KUBECTL[@]}" get volumes.longhorn.io -n longhorn-system --no-headers \
    -o 'custom-columns=NAME:.metadata.name,STATE:.status.state,ROBUSTNESS:.status.robustness,NODE:.status.currentNodeID' 2>&1 || true)"
  volume_count="$(printf '%s\n' "$longhorn_volumes" | awk 'NF {count++} END {print count+0}')"
  printf 'volumes_total=%s\n' "$volume_count"
  unhealthy="$(printf '%s\n' "$longhorn_volumes" | awk '$2 == "attached" && $3 != "healthy" { print }')"
  if [ -n "$unhealthy" ]; then
    printf '%s\n' "$unhealthy" | head -n 40 | limit_width
  else
    printf 'none\n'
  fi
else
  printf 'not-installed\n'
fi

printf 'snapshot=ok\n'
