#!/usr/bin/env bash
set -euo pipefail

TARGET_NODE="claw-node-3"

if [ "${1:-}" = "--self-test" ]; then
  [ "$#" -eq 1 ]
  [ "$TARGET_NODE" = "claw-node-3" ]
  printf 'DIAGNOSE_CLAW_NODE_3_STORAGE_SELF_TEST_OK\n'
  exit 0
fi

if [ "$#" -ne 0 ]; then
  echo "Usage: diagnose-claw-node-3-storage.sh [--self-test]" >&2
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

SSH=(ssh -o BatchMode=yes -o ConnectTimeout=10 -o StrictHostKeyChecking=yes "$TARGET_NODE")
"${SSH[@]}" 'hostname; sudo -n true'

printf 'NODE_STORAGE_DIAGNOSTIC\n'
printf 'target=%s\n' "$TARGET_NODE"

printf '[node-condition]\n'
"${KUBECTL[@]}" get node "$TARGET_NODE" \
  -o 'custom-columns=NAME:.metadata.name,READY:.status.conditions[?(@.type=="Ready")].status,DISK_PRESSURE:.status.conditions[?(@.type=="DiskPressure")].status,TAINTS:.spec.taints[*].key'

printf '[filesystem-bytes]\n'
"${SSH[@]}" 'df -h / /var/lib/kubelet /var/lib/rancher/k3s 2>/dev/null || df -h /'

printf '[filesystem-inodes]\n'
"${SSH[@]}" 'df -ih / /var/lib/kubelet /var/lib/rancher/k3s 2>/dev/null || df -ih /'

printf '[top-root-directories]\n'
"${SSH[@]}" 'sudo -n du -x -B1G -d1 / 2>/dev/null | sort -n | tail -n 20'

for path in /var /var/lib /home /var/lib/kubelet /var/lib/rancher/k3s /var/log /var/tmp /tmp; do
  printf '[usage:%s]\n' "$path"
  "${SSH[@]}" "if [ -e '$path' ]; then sudo -n du -x -B1G -d1 '$path' 2>/dev/null | sort -n | tail -n 40; else echo missing; fi"
done

printf '[home-second-level-largest]\n'
"${SSH[@]}" 'if [ -d /home ]; then sudo -n du -x -B1G -d2 /home 2>/dev/null | sort -n | tail -n 50; else echo missing; fi'

printf '[var-lib-second-level-largest]\n'
"${SSH[@]}" 'if [ -d /var/lib ]; then sudo -n du -x -B1G -d2 /var/lib 2>/dev/null | sort -n | tail -n 60; else echo missing; fi'

printf '[kubelet-pod-directories-largest]\n'
"${SSH[@]}" 'if [ -d /var/lib/kubelet/pods ]; then sudo -n du -x -B1G -d1 /var/lib/kubelet/pods 2>/dev/null | sort -n | tail -n 30; else echo missing; fi'

printf 'diagnostic=ok\n'
