#!/usr/bin/env bash
set -euo pipefail

if [ "${1:-}" = "--self-test" ]; then
  [ "$#" -eq 1 ]
  printf 'SET_KUBERNETES_NODE_CORDON_SELF_TEST_OK\n'
  exit 0
fi

if [ "$#" -ne 2 ]; then
  echo "Usage: set-kubernetes-node-cordon.sh <node> <cordon|uncordon>" >&2
  exit 2
fi

NODE="$1"
ACTION="$2"
case "$NODE" in *[!A-Za-z0-9._-]*|'') echo "ERROR: invalid node" >&2; exit 2;; esac
case "$ACTION" in cordon|uncordon) ;; *) echo "ERROR: action must be cordon or uncordon" >&2; exit 2;; esac

if sudo -n kubectl version --client >/dev/null 2>&1; then
  KUBECTL=(sudo -n kubectl)
elif sudo -n k3s kubectl version --client >/dev/null 2>&1; then
  KUBECTL=(sudo -n k3s kubectl)
else
  echo "ERROR: no sanctioned non-interactive Kubernetes client is available" >&2
  exit 1
fi

printf '[before]\n'
"${KUBECTL[@]}" get node "$NODE" -o 'custom-columns=NAME:.metadata.name,UNSCHEDULABLE:.spec.unschedulable,READY:.status.conditions[?(@.type=="Ready")].status'

if [ "$ACTION" = "cordon" ]; then
  "${KUBECTL[@]}" cordon "$NODE"
else
  "${KUBECTL[@]}" uncordon "$NODE"
fi

printf '[after]\n'
"${KUBECTL[@]}" get node "$NODE" -o 'custom-columns=NAME:.metadata.name,UNSCHEDULABLE:.spec.unschedulable,READY:.status.conditions[?(@.type=="Ready")].status'
