#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 3 ]]; then
  echo "usage: $0 <namespace> <pod> <expected-node>" >&2
  exit 2
fi
NS="$1"
POD="$2"
NODE="$3"
K=(sudo -n k3s kubectl)

actual_node="$("${K[@]}" get pod -n "$NS" "$POD" -o jsonpath='{.spec.nodeName}')"
ephemeral="$("${K[@]}" get pod -n "$NS" "$POD" -o jsonpath='{.metadata.labels.actions-ephemeral-runner}')"
[[ "$actual_node" == "$NODE" ]] || { echo "refusing: pod is on $actual_node, not $NODE" >&2; exit 1; }
[[ "$ephemeral" == "True" || "$ephemeral" == "true" ]] || { echo "refusing: pod is not labelled ephemeral runner" >&2; exit 1; }

"${K[@]}" delete pod -n "$NS" "$POD" --wait=false
for _ in $(seq 1 60); do
  replacement="$("${K[@]}" get pods -n "$NS" -l actions-ephemeral-runner=True -o jsonpath='{range .items[*]}{.metadata.name}{" "}{.spec.nodeName}{"\n"}{end}' 2>/dev/null | grep -v "^${POD} " | head -1 || true)"
  if [[ -n "$replacement" ]]; then
    echo "replacement=$replacement"
    exit 0
  fi
  sleep 2
done
echo "replacement not observed within 120s" >&2
exit 1
