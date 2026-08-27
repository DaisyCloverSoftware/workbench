#!/usr/bin/env bash
set -euo pipefail

TARGET_NODE="claw-node-3"

if [ "${1:-}" = "--self-test" ]; then
  [ "$#" -eq 1 ]
  [ "$TARGET_NODE" = "claw-node-3" ]
  printf 'INSPECT_CLAW_NODE_3_STORAGE_SELF_TEST_OK\n'
  exit 0
fi

if [ "$#" -ne 0 ]; then
  echo "Usage: inspect-claw-node-3-storage.sh [--self-test]" >&2
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

node_name="$("${KUBECTL[@]}" get node "$TARGET_NODE" -o jsonpath='{.metadata.name}')"
if [ "$node_name" != "$TARGET_NODE" ]; then
  echo "ERROR: target node identity mismatch" >&2
  exit 1
fi

if ! command -v ssh >/dev/null 2>&1; then
  echo "ERROR: ssh is unavailable on the Workbench operator host" >&2
  exit 1
fi

SSH=(ssh -o BatchMode=yes -o ConnectTimeout=10 -o StrictHostKeyChecking=yes "$TARGET_NODE")

printf 'NODE_STORAGE_DIAGNOSTIC\n'
printf 'target=%s\n' "$TARGET_NODE"
printf '[node-condition]\n'
"${KUBECTL[@]}" get node "$TARGET_NODE" \
  -o 'custom-columns=NAME:.metadata.name,READY:.status.conditions[?(@.type=="Ready")].status,DISK_PRESSURE:.status.conditions[?(@.type=="DiskPressure")].status,TAINTS:.spec.taints[*].key'

printf '[arc-runner-pods]\n'
"${KUBECTL[@]}" get pods -n arc-runners-node3 \
  -o 'custom-columns=NAME:.metadata.name,PHASE:.status.phase,NODE:.spec.nodeName,CREATED:.metadata.creationTimestamp' 2>/dev/null || true

"${SSH[@]}" 'set -eu
printf "[host]\n"
hostname
printf "[filesystem]\n"
df -h / /var/lib/rancher/k3s 2>/dev/null || df -h /
printf "[major-directories-bytes]\n"
for root in /home /opt /srv /var /tmp; do
  if [ -e "$root" ]; then
    sudo -n du -x -B1 -d1 "$root" 2>/dev/null || true
  fi
done | sort -nr | head -n 100
printf "[runner-workspaces-bytes]\n"
for root in /home /opt /srv /var /tmp; do
  [ -e "$root" ] || continue
  sudo -n find "$root" -xdev -type d -name _work -print 2>/dev/null || true
done | while IFS= read -r workspace; do
  [ -n "$workspace" ] || continue
  bytes="$(sudo -n du -x -B1 -s "$workspace" 2>/dev/null | awk "{print \$1}" || true)"
  [ -n "$bytes" ] || bytes=0
  printf "%s\t%s\n" "$bytes" "$workspace"
done | sort -nr
printf "[kubelet-storage-bytes]\n"
sudo -n du -x -B1 -d2 /var/lib/kubelet 2>/dev/null | sort -nr | head -n 80 || true
printf "[k3s-storage-bytes]\n"
sudo -n du -x -B1 -d2 /var/lib/rancher/k3s 2>/dev/null | sort -nr | head -n 80 || true
printf "[largest-logs-bytes]\n"
sudo -n find /var/log -xdev -type f -printf "%s\t%p\n" 2>/dev/null | sort -nr | head -n 40 || true
'
