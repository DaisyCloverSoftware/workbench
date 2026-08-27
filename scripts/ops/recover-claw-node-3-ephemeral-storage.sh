#!/usr/bin/env bash
set -euo pipefail

TARGET_NODE="claw-node-3"
CI_USER="daisyclover-ci"
CI_NPM_CACHE="/home/daisyclover-ci/.npm"

if [ "${1:-}" = "--self-test" ]; then
  [ "$#" -eq 1 ]
  [ "$TARGET_NODE" = "claw-node-3" ]
  [ "$CI_USER" = "daisyclover-ci" ]
  [ "$CI_NPM_CACHE" = "/home/daisyclover-ci/.npm" ]
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
# query the exact k3s/containerd runtime we intend to prune.
"${SSH[@]}" 'hostname; sudo -n k3s crictl version >/dev/null'

# This incident's read-only diagnosis proved Longhorn replicas are the dominant
# /var consumer and are protected application data. The only additional reclaim
# target here is the dedicated CI user's stale npm cache. Validate the exact path,
# ownership and absence of symlink redirection before deleting cache contents.
# The operator account cannot traverse the CI home directly, so path inspection is
# deliberately performed through the same non-interactive sudo boundary used by
# the diagnostics; deletion itself still runs as the CI user.
printf '[ci-cache-safety]\n'
"${SSH[@]}" "set -eu
  sudo -n test -d '$CI_NPM_CACHE'
  sudo -n test ! -L '$CI_NPM_CACHE'
  resolved=\$(sudo -n readlink -f '$CI_NPM_CACHE')
  [ \"\$resolved\" = '$CI_NPM_CACHE' ]
  owner=\$(sudo -n stat -c '%U:%G' '$CI_NPM_CACHE')
  [ \"\$owner\" = '$CI_USER:$CI_USER' ]
  test -x /usr/bin/find
  test -x /usr/bin/rm
  foreign=\$(sudo -n /usr/bin/find '$CI_NPM_CACHE' -xdev -mindepth 1 -maxdepth 1 ! -user '$CI_USER' -print -quit)
  [ -z \"\$foreign\" ]
  printf 'cache=%s\\nowner=%s\\nresolved=%s\\n' '$CI_NPM_CACHE' \"\$owner\" \"\$resolved\"
"

active_arc="$("${KUBECTL[@]}" get pods -n arc-runners-node3 \
  -o jsonpath='{range .items[*]}{.spec.nodeName}{"|"}{.status.phase}{"\n"}{end}' 2>/dev/null \
  | awk -F'|' -v node="$TARGET_NODE" '$1 == node && $2 == "Running" {count++} END {print count+0}')"
if [ "$active_arc" != "0" ]; then
  echo "ERROR: an ARC runner is currently Running on $TARGET_NODE; cache cleanup aborted" >&2
  exit 1
fi

if "${SSH[@]}" 'ps -eo args= | grep -E "Runner[.](Listener|Worker)|runsvc[.]sh|actions-runner/(bin/)?Runner" | grep -v grep >/dev/null'; then
  echo "ERROR: a legacy host runner process is active on $TARGET_NODE; cache cleanup aborted" >&2
  exit 1
fi

printf '[before-filesystem]\n'
"${SSH[@]}" 'df -h / /var/lib/rancher/k3s 2>/dev/null || df -h /'

printf '[before-images-summary]\n'
"${SSH[@]}" 'sudo -n k3s crictl images | awk '\''NR == 1 {print; next} {count++; size=$NF; print $1, $2, $3, size} END {print "images_total=" count+0}'\'' | head -n 80'

printf '[prune-unused-k3s-images]\n'
"${SSH[@]}" 'sudo -n k3s crictl rmi --prune'

printf '[ci-npm-cache-before]\n'
"${SSH[@]}" "sudo -n du -sh '$CI_NPM_CACHE'"

printf '[clear-ci-npm-cache]\n'
"${SSH[@]}" "cd / && sudo -n -u '$CI_USER' /usr/bin/find '$CI_NPM_CACHE' -xdev -mindepth 1 -maxdepth 1 -exec /usr/bin/rm -rf -- {} +"

printf '[ci-npm-cache-after]\n'
"${SSH[@]}" "set -eu
  remaining=\$(sudo -n /usr/bin/find '$CI_NPM_CACHE' -xdev -mindepth 1 -maxdepth 1 -print -quit)
  [ -z \"\$remaining\" ]
  sudo -n du -sh '$CI_NPM_CACHE'
"

printf '[after-filesystem]\n'
"${SSH[@]}" 'df -h / /var/lib/rancher/k3s 2>/dev/null || df -h /'

# Do not remove the Kubernetes DiskPressure taint manually. Kubelet owns that
# condition; wait only for automatic recovery after safe cache reclamation.
pressure="True"
for _ in $(seq 1 24); do
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
  echo "ERROR: unused-image and dedicated CI npm-cache cleanup completed but node remains under DiskPressure; Longhorn, Docker volumes, runner workspaces, pods and taints were not touched" >&2
  exit 3
fi

printf 'recovery=ok\n'
