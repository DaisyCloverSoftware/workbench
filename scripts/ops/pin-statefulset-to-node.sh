#!/usr/bin/env bash
set -euo pipefail

if [ "${1:-}" = "--self-test" ]; then
  [ "$#" -eq 1 ]
  printf 'PIN_STATEFULSET_TO_NODE_SELF_TEST_OK\n'
  exit 0
fi

if [ "$#" -ne 3 ]; then
  echo "Usage: pin-statefulset-to-node.sh <namespace> <statefulset> <node>" >&2
  exit 2
fi

NAMESPACE="$1"
NAME="$2"
NODE="$3"
for value in "$NAMESPACE" "$NAME" "$NODE"; do
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

replicas="$("${KUBECTL[@]}" get statefulset "$NAME" -n "$NAMESPACE" -o jsonpath='{.spec.replicas}')"
if [ "$replicas" != "0" ]; then
  echo "ERROR: statefulset must be scaled to zero before changing placement; current replicas=$replicas" >&2
  exit 1
fi

current_selector="$("${KUBECTL[@]}" get statefulset "$NAME" -n "$NAMESPACE" -o jsonpath='{.spec.template.spec.nodeSelector}')"
case "$current_selector" in ""|"map[]"|"{}") ;; *) echo "ERROR: refusing to overwrite existing nodeSelector: $current_selector" >&2; exit 1;; esac

ready="$("${KUBECTL[@]}" get node "$NODE" -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}')"
if [ "$ready" != "True" ]; then
  echo "ERROR: target node is not Ready: $NODE ($ready)" >&2
  exit 1
fi

printf '[before]\n'
"${KUBECTL[@]}" get statefulset "$NAME" -n "$NAMESPACE" -o 'custom-columns=NAME:.metadata.name,REPLICAS:.spec.replicas,NODESELECTOR:.spec.template.spec.nodeSelector'

patch="{\"spec\":{\"template\":{\"spec\":{\"nodeSelector\":{\"kubernetes.io/hostname\":\"$NODE\"}}}}}"
"${KUBECTL[@]}" patch statefulset "$NAME" -n "$NAMESPACE" --type=merge -p "$patch"

printf '[after]\n'
"${KUBECTL[@]}" get statefulset "$NAME" -n "$NAMESPACE" -o 'custom-columns=NAME:.metadata.name,REPLICAS:.spec.replicas,NODESELECTOR:.spec.template.spec.nodeSelector'
