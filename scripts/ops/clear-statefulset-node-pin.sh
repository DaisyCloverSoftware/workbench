#!/usr/bin/env bash
set -euo pipefail

if [ "${1:-}" = "--self-test" ]; then
  [ "$#" -eq 1 ]
  printf 'CLEAR_STATEFULSET_NODE_PIN_SELF_TEST_OK\n'
  exit 0
fi

if [ "$#" -ne 3 ]; then
  echo "Usage: clear-statefulset-node-pin.sh <namespace> <statefulset> <expected-node>" >&2
  exit 2
fi

NAMESPACE="$1"
NAME="$2"
EXPECTED_NODE="$3"
for value in "$NAMESPACE" "$NAME" "$EXPECTED_NODE"; do
  case "$value" in *[!A-Za-z0-9._-]*|'') echo "ERROR: invalid identifier" >&2; exit 2;; esac
done

if sudo -n kubectl version --client >/dev/null 2>&1; then
  KUBECTL=(sudo -n kubectl)
elif sudo -n k3s kubectl version --client >/dev/null 2>&1; then
  KUBECTL=(sudo -n k3s kubectl)
else
  echo "ERROR: no sanctioned non-interactive Kubernetes client is available" >&2
  exit 1
fi

current_node="$("${KUBECTL[@]}" get statefulset "$NAME" -n "$NAMESPACE" -o jsonpath='{.spec.template.spec.nodeSelector.kubernetes\.io/hostname}')"
if [ "$current_node" != "$EXPECTED_NODE" ]; then
  echo "ERROR: current node pin does not match expected node: current=$current_node expected=$EXPECTED_NODE" >&2
  exit 1
fi

printf '[before]\n'
"${KUBECTL[@]}" get statefulset "$NAME" -n "$NAMESPACE" -o 'custom-columns=NAME:.metadata.name,REPLICAS:.spec.replicas,NODESELECTOR:.spec.template.spec.nodeSelector'

"${KUBECTL[@]}" patch statefulset "$NAME" -n "$NAMESPACE" --type=merge -p '{"spec":{"template":{"spec":{"nodeSelector":null}}}}'

printf '[after]\n'
"${KUBECTL[@]}" get statefulset "$NAME" -n "$NAMESPACE" -o 'custom-columns=NAME:.metadata.name,REPLICAS:.spec.replicas,NODESELECTOR:.spec.template.spec.nodeSelector'
