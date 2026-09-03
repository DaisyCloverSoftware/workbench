#!/usr/bin/env bash
set -euo pipefail

if [ "${1:-}" = "--self-test" ]; then
  [ "$#" -eq 1 ]
  printf 'INSPECT_LONGHORN_NODE_VOLUME_SELF_TEST_OK\n'
  exit 0
fi

if [ "$#" -ne 2 ]; then
  echo "Usage: inspect-longhorn-node-volume.sh <node> <longhorn-volume>" >&2
  exit 2
fi

TARGET_NODE="$1"
VOLUME="$2"

case "$TARGET_NODE" in
  *[!A-Za-z0-9.-]*|'') echo "ERROR: invalid node name" >&2; exit 2 ;;
esac
case "$VOLUME" in
  *[!A-Za-z0-9._-]*|'') echo "ERROR: invalid Longhorn volume name" >&2; exit 2 ;;
esac

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

printf 'LONGHORN_NODE_VOLUME_INSPECTION\n'
printf 'target=%s\nvolume=%s\n' "$TARGET_NODE" "$VOLUME"
printf '[node-condition]\n'
"${KUBECTL[@]}" get node "$TARGET_NODE" -o 'custom-columns=NAME:.metadata.name,READY:.status.conditions[?(@.type=="Ready")].status,DISK_PRESSURE:.status.conditions[?(@.type=="DiskPressure")].status,MEMORY_PRESSURE:.status.conditions[?(@.type=="MemoryPressure")].status,PID_PRESSURE:.status.conditions[?(@.type=="PIDPressure")].status'

"${SSH[@]}" sudo -n bash -s -- "$VOLUME" <<'REMOTE'
set -euo pipefail
volume="$1"
device="/dev/longhorn/$volume"

printf '[identity]\n'
hostname -f || hostname
date -u '+utc=%Y-%m-%dT%H:%M:%SZ'

printf '[device]\n'
if [ -e "$device" ]; then
  ls -l "$device"
  printf 'resolved_device='
  readlink -f "$device" || true
  printf 'block_read_only='
  blockdev --getro "$device" || true
else
  printf 'device_missing=%s\n' "$device"
fi

printf '[mounts-for-device]\n'
findmnt -S "$device" -o SOURCE,TARGET,FSTYPE,OPTIONS || true

printf '[block-layout]\n'
lsblk -o NAME,KNAME,PATH,TYPE,FSTYPE,RO,SIZE,MOUNTPOINTS

printf '[longhorn-filesystem-capacity]\n'
df -hT /var/lib/longhorn 2>/dev/null || true

printf '[kernel-storage-events-last-48h]\n'
journalctl -k --since '-48 hours' --no-pager -g 'EXT4|I/O error|Buffer I/O|blk_update|read-only|remount|longhorn|sd[a-z]|nvme|virtio|scsi' -n 800 || true
REMOTE
