#!/usr/bin/env bash
set -euo pipefail

valid_namespace() {
  local value="$1"
  [ "${#value}" -le 63 ] || return 1
  [[ "$value" =~ ^[a-z0-9]([-a-z0-9]*[a-z0-9])?$ ]]
}

if [ "${1:-}" = "--self-test" ]; then
  valid_namespace "rum-dev"
  valid_namespace "careload-dev"
  ! valid_namespace "../default"
  ! valid_namespace "UPPERCASE"
  ! valid_namespace "name_with_underscore"
  printf 'NAMESPACE_HEALTH_SELF_TEST_OK\n'
  exit 0
fi

if [ "$#" -ne 1 ] || ! valid_namespace "$1"; then
  echo "Usage: namespace-health.sh <kubernetes-namespace>" >&2
  exit 2
fi

namespace="$1"

if sudo -n kubectl version --client >/dev/null 2>&1; then
  KUBECTL=(sudo -n kubectl)
elif sudo -n k3s kubectl version --client >/dev/null 2>&1; then
  KUBECTL=(sudo -n k3s kubectl)
else
  echo "ERROR: no sanctioned non-interactive Kubernetes client is available" >&2
  exit 1
fi

if ! "${KUBECTL[@]}" get namespace "$namespace" -o name >/dev/null 2>&1; then
  echo "ERROR: namespace does not exist or is not readable: $namespace" >&2
  exit 1
fi

section() {
  local label="$1"
  shift
  local output
  printf '[%s]\n' "$label"
  if ! output="$("${KUBECTL[@]}" "$@" 2>&1)"; then
    printf 'ERROR: %s\n' "$output" >&2
    return 1
  fi
  if [ -n "$output" ]; then
    printf '%s\n' "$output"
  else
    printf 'none\n'
  fi
}

printf 'NAMESPACE_HEALTH\n'
printf 'namespace=%s\n' "$namespace"
printf 'host=%s\n' "$(hostname)"

section deployments get deployments -n "$namespace" --no-headers \
  -o 'custom-columns=NAME:.metadata.name,READY:.status.readyReplicas,DESIRED:.spec.replicas,AVAILABLE:.status.availableReplicas,UPDATED:.status.updatedReplicas'

section statefulsets get statefulsets -n "$namespace" --no-headers \
  -o 'custom-columns=NAME:.metadata.name,READY:.status.readyReplicas,DESIRED:.spec.replicas,CURRENT:.status.currentReplicas,UPDATED:.status.updatedReplicas'

section pods get pods -n "$namespace" --no-headers \
  -o 'custom-columns=NAME:.metadata.name,PHASE:.status.phase,READY:.status.containerStatuses[*].ready,RESTARTS:.status.containerStatuses[*].restartCount,NODE:.spec.nodeName'

section jobs get jobs -n "$namespace" --no-headers \
  -o 'custom-columns=NAME:.metadata.name,SUCCEEDED:.status.succeeded,ACTIVE:.status.active,FAILED:.status.failed'

section pvcs get persistentvolumeclaims -n "$namespace" --no-headers \
  -o 'custom-columns=NAME:.metadata.name,STATUS:.status.phase,CAPACITY:.status.capacity.storage,CLASS:.spec.storageClassName'

printf '[recent-warnings]\n'
warnings="$("${KUBECTL[@]}" get events -n "$namespace" --field-selector type=Warning --sort-by=.lastTimestamp --no-headers \
  -o 'custom-columns=LAST:.lastTimestamp,REASON:.reason,KIND:.involvedObject.kind,OBJECT:.involvedObject.name,MESSAGE:.message' 2>&1 || true)"
if [ -n "$warnings" ]; then
  printf '%s\n' "$warnings" | tail -n 12
else
  printf 'none\n'
fi

printf 'snapshot=ok\n'
