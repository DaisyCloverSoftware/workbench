#!/usr/bin/env bash
set -euo pipefail

if [ "${1:-}" = "--self-test" ]; then
  [ "$#" -eq 1 ]
  printf 'SCALE_KUBERNETES_WORKLOAD_SELF_TEST_OK\n'
  exit 0
fi

if [ "$#" -ne 4 ]; then
  echo "Usage: scale-kubernetes-workload.sh <namespace> <deployment|statefulset> <name> <replicas>" >&2
  exit 2
fi

NAMESPACE="$1"
KIND="$2"
NAME="$3"
REPLICAS="$4"

case "$NAMESPACE" in *[!A-Za-z0-9._-]*|'') echo "ERROR: invalid namespace" >&2; exit 2;; esac
case "$NAME" in *[!A-Za-z0-9._-]*|'') echo "ERROR: invalid workload name" >&2; exit 2;; esac
case "$KIND" in deployment|statefulset) ;; *) echo "ERROR: kind must be deployment or statefulset" >&2; exit 2;; esac
case "$REPLICAS" in ''|*[!0-9]*) echo "ERROR: replicas must be an integer" >&2; exit 2;; esac
if [ "$REPLICAS" -gt 20 ]; then echo "ERROR: replicas exceeds bounded maximum 20" >&2; exit 2; fi

if sudo -n kubectl version --client >/dev/null 2>&1; then
  KUBECTL=(sudo -n kubectl)
elif sudo -n k3s kubectl version --client >/dev/null 2>&1; then
  KUBECTL=(sudo -n k3s kubectl)
else
  echo "ERROR: no sanctioned non-interactive Kubernetes client is available" >&2
  exit 1
fi

printf '[before]\n'
"${KUBECTL[@]}" get "$KIND" "$NAME" -n "$NAMESPACE" -o 'custom-columns=NAME:.metadata.name,REPLICAS:.spec.replicas,READY:.status.readyReplicas,AVAILABLE:.status.availableReplicas,IMAGE:.spec.template.spec.containers[*].image'

"${KUBECTL[@]}" scale "$KIND/$NAME" -n "$NAMESPACE" --replicas="$REPLICAS"

printf '[after]\n'
"${KUBECTL[@]}" get "$KIND" "$NAME" -n "$NAMESPACE" -o 'custom-columns=NAME:.metadata.name,REPLICAS:.spec.replicas,READY:.status.readyReplicas,AVAILABLE:.status.availableReplicas,IMAGE:.spec.template.spec.containers[*].image'
