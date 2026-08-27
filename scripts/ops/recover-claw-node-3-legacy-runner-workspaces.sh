#!/usr/bin/env bash
set -euo pipefail

TARGET_NODE="claw-node-3"
CI_USER="daisyclover-ci"
RUNNER_ONE_UNIT="actions.runner.DaisyCloverSoftware.claw-node-3-daisyclover-ci.service"
RUNNER_TWO_UNIT="actions.runner.DaisyCloverSoftware.claw-node-3-daisyclover-ci-2.service"
WORKSPACE_ONE="/home/daisyclover-ci/actions-runner/_work"
WORKSPACE_TWO="/home/daisyclover-ci/actions-runner-2/_work"
MIN_RECLAIM_BYTES=$((10 * 1024 * 1024 * 1024))

if [ "${1:-}" = "--self-test" ]; then
  [ "$#" -eq 1 ]
  [ "$TARGET_NODE" = "claw-node-3" ]
  [ "$CI_USER" = "daisyclover-ci" ]
  [ "$RUNNER_ONE_UNIT" = "actions.runner.DaisyCloverSoftware.claw-node-3-daisyclover-ci.service" ]
  [ "$RUNNER_TWO_UNIT" = "actions.runner.DaisyCloverSoftware.claw-node-3-daisyclover-ci-2.service" ]
  [ "$WORKSPACE_ONE" = "/home/daisyclover-ci/actions-runner/_work" ]
  [ "$WORKSPACE_TWO" = "/home/daisyclover-ci/actions-runner-2/_work" ]
  [ "$MIN_RECLAIM_BYTES" -eq 10737418240 ]
  printf 'RECOVER_CLAW_NODE_3_LEGACY_RUNNER_WORKSPACES_SELF_TEST_OK\n'
  exit 0
fi

if [ "$#" -ne 0 ]; then
  echo "Usage: recover-claw-node-3-legacy-runner-workspaces.sh [--self-test]" >&2
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

pressure="$("${KUBECTL[@]}" get node "$TARGET_NODE" -o jsonpath='{.status.conditions[?(@.type=="DiskPressure")].status}')"
if [ "$pressure" != "True" ]; then
  printf 'disk_pressure=%s\naction=none-required\n' "$pressure"
  exit 0
fi

active_arc="$("${KUBECTL[@]}" get pods -n arc-runners-node3 \
  -o jsonpath='{range .items[*]}{.spec.nodeName}{"|"}{.status.phase}{"\n"}{end}' 2>/dev/null \
  | awk -F'|' -v node="$TARGET_NODE" '$1 == node && $2 == "Running" {count++} END {print count+0}')"
if [ "$active_arc" != "0" ]; then
  echo "ERROR: an ARC runner is currently Running on $TARGET_NODE; legacy workspace cleanup aborted" >&2
  exit 1
fi

SSH=(ssh -o BatchMode=yes -o ConnectTimeout=10 -o StrictHostKeyChecking=yes "$TARGET_NODE")

printf 'LEGACY_RUNNER_WORKSPACE_RECOVERY\n'
printf 'target=%s\n' "$TARGET_NODE"
printf '[before-node-condition]\n'
"${KUBECTL[@]}" get node "$TARGET_NODE" \
  -o 'custom-columns=NAME:.metadata.name,READY:.status.conditions[?(@.type=="Ready")].status,DISK_PRESSURE:.status.conditions[?(@.type=="DiskPressure")].status,TAINTS:.spec.taints[*].key'

# Fail closed unless both known legacy host runner services exist and are inactive,
# no legacy runner process is alive, and both exact _work directories are ordinary
# CI-user-owned directories that resolve to the reviewed paths.
printf '[legacy-runner-safety]\n'
"${SSH[@]}" "set -eu
  for unit in '$RUNNER_ONE_UNIT' '$RUNNER_TWO_UNIT'; do
    load=\$(sudo -n systemctl show -p LoadState --value \"\$unit\")
    active=\$(sudo -n systemctl is-active \"\$unit\" 2>/dev/null || true)
    [ \"\$load\" = loaded ]
    [ \"\$active\" != active ]
    printf 'unit=%s load=%s active=%s\\n' \"\$unit\" \"\$load\" \"\$active\"
  done
  if ps -eo args= | grep -E 'Runner[.](Listener|Worker)|runsvc[.]sh|actions-runner/(bin/)?Runner' | grep -v grep >/dev/null; then
    echo 'ERROR: legacy runner process detected' >&2
    exit 1
  fi
  for workspace in '$WORKSPACE_ONE' '$WORKSPACE_TWO'; do
    sudo -n test -d \"\$workspace\"
    sudo -n test ! -L \"\$workspace\"
    resolved=\$(sudo -n readlink -f \"\$workspace\")
    [ \"\$resolved\" = \"\$workspace\" ]
    owner=\$(sudo -n stat -c '%U:%G' \"\$workspace\")
    [ \"\$owner\" = '$CI_USER:$CI_USER' ]
    bytes=\$(sudo -n du -x -B1 -s \"\$workspace\" | awk '{print \$1}')
    printf 'workspace=%s owner=%s bytes=%s\\n' \"\$workspace\" \"\$owner\" \"\$bytes\"
  done
"

workspace_one_bytes="$("${SSH[@]}" "sudo -n du -x -B1 -s '$WORKSPACE_ONE' | awk '{print \$1}'")"
workspace_two_bytes="$("${SSH[@]}" "sudo -n du -x -B1 -s '$WORKSPACE_TWO' | awk '{print \$1}'")"
total_bytes=$((workspace_one_bytes + workspace_two_bytes))
if [ "$total_bytes" -lt "$MIN_RECLAIM_BYTES" ]; then
  echo "ERROR: reviewed legacy workspaces no longer contain at least 10 GiB total; cleanup aborted to avoid racing another recovery" >&2
  exit 1
fi

printf '[before-filesystem]\n'
"${SSH[@]}" 'df -h /'

printf '[clear-workspace-one]\n'
"${SSH[@]}" "cd / && sudo -n -u '$CI_USER' /usr/bin/find '$WORKSPACE_ONE' -xdev -mindepth 1 -maxdepth 1 -exec /usr/bin/rm -rf -- {} +"

printf '[clear-workspace-two]\n'
"${SSH[@]}" "cd / && sudo -n -u '$CI_USER' /usr/bin/find '$WORKSPACE_TWO' -xdev -mindepth 1 -maxdepth 1 -exec /usr/bin/rm -rf -- {} +"

printf '[after-workspaces]\n'
"${SSH[@]}" "set -eu
  for workspace in '$WORKSPACE_ONE' '$WORKSPACE_TWO'; do
    remaining=\$(sudo -n /usr/bin/find \"\$workspace\" -xdev -mindepth 1 -maxdepth 1 -print -quit)
    [ -z \"\$remaining\" ]
    sudo -n du -sh \"\$workspace\"
  done
"

printf '[after-filesystem]\n'
"${SSH[@]}" 'df -h /'

# Kubelet owns DiskPressure and its NoSchedule taint. Never remove the taint
# manually; wait for the node to recover naturally after reclaiming only the
# reviewed inactive runner workspaces.
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
  echo "ERROR: legacy runner _work cleanup completed but node remains under DiskPressure; no Longhorn, Docker-volume, pod, manifest, database or taint mutation was attempted" >&2
  exit 3
fi

printf 'recovery=ok\n'
