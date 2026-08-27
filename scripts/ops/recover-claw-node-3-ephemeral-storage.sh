#!/usr/bin/env bash
set -euo pipefail

TARGET_NODE="claw-node-3"

if [ "${1:-}" = "--self-test" ]; then
  [ "$#" -eq 1 ]
  [ "$TARGET_NODE" = "claw-node-3" ]
  printf 'RECOVER_CLAW_NODE_3_EPHEMERAL_STORAGE_SELF_TEST_OK\n'
  exit 0
fi

if [ "$#" -ne 0 ]; then
  echo "Usage: recover-claw-node-3-ephemeral-storage.sh [--self-test]" >&2
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

if ! command -v ssh >/dev/null 2>&1; then
  echo "ERROR: ssh is unavailable on the Workbench operator host" >&2
  exit 1
fi

node_name="$("${KUBECTL[@]}" get node "$TARGET_NODE" -o jsonpath='{.metadata.name}')"
if [ "$node_name" != "$TARGET_NODE" ]; then
  echo "ERROR: target node identity mismatch" >&2
  exit 1
fi

SSH=(ssh -o BatchMode=yes -o ConnectTimeout=10 -o StrictHostKeyChecking=yes "$TARGET_NODE")

printf 'NODE_STORAGE_RECOVERY\n'
printf 'target=%s\n' "$TARGET_NODE"

printf '[before-node-condition]\n'
"${KUBECTL[@]}" get node "$TARGET_NODE" \
  -o 'custom-columns=NAME:.metadata.name,READY:.status.conditions[?(@.type=="Ready")].status,DISK_PRESSURE:.status.conditions[?(@.type=="DiskPressure")].status'

before_pressure="$("${KUBECTL[@]}" get node "$TARGET_NODE" -o jsonpath='{.status.conditions[?(@.type=="DiskPressure")].status}')"
if [ "$before_pressure" != "True" ]; then
  printf 'disk_pressure=%s\n' "$before_pressure"
  printf 'action=none-required\n'
  exit 0
fi

# Fail closed unless the reviewed target is reachable and non-interactive sudo can
# query the exact k3s/containerd runtime we intend to clean.
"${SSH[@]}" 'hostname; sudo -n k3s crictl version >/dev/null'

printf '[before-filesystem]\n'
"${SSH[@]}" 'df -h / /var/lib/rancher/k3s 2>/dev/null || df -h /'

printf '[before-images-summary]\n'
"${SSH[@]}" 'sudo -n k3s crictl images | awk '\''NR == 1 {print; next} {count++; size=$NF; print $1, $2, $3, size} END {print "images_total=" count+0}'\'' | head -n 80'

printf '[prune-unused-k3s-images]\n'
"${SSH[@]}" 'sudo -n k3s crictl rmi --prune'

printf '[after-filesystem]\n'
"${SSH[@]}" 'df -h / /var/lib/rancher/k3s 2>/dev/null || df -h /'

# Do not remove the Kubernetes DiskPressure taint manually. Kubelet owns that
# condition; wait only for the automatic condition/taint recovery after space is
# reclaimed.
pressure="True"
for _ in $(seq 1 18); do
  pressure="$("${KUBECTL[@]}" get node "$TARGET_NODE" -o jsonpath='{.status.conditions[?(@.type=="DiskPressure")].status}')"
  if [ "$pressure" != "True" ]; then
    break
  fi
  sleep 10
done

printf '[after-node-condition]\n'
"${KUBECTL[@]}" get node "$TARGET_NODE" \
  -o 'custom-columns=NAME:.metadata.name,READY:.status.conditions[?(@.type=="Ready")].status,DISK_PRESSURE:.status.conditions[?(@.type=="DiskPressure")].status,TAINTS:.spec.taints[*].key'

if [ "$pressure" = "True" ]; then
  echo "ERROR: unused-image prune completed but node remains under DiskPressure; no broader cleanup was attempted" >&2
  exit 3
fi

printf 'recovery=ok\n'
