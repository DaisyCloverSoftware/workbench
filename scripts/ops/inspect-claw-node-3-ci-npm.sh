#!/usr/bin/env bash
set -euo pipefail

TARGET_NODE="claw-node-3"
CI_USER="daisyclover-ci"
CI_HOME="/home/daisyclover-ci"

if [ "${1:-}" = "--self-test" ]; then
  [ "$#" -eq 1 ]
  [ "$TARGET_NODE" = "claw-node-3" ]
  [ "$CI_USER" = "daisyclover-ci" ]
  [ "$CI_HOME" = "/home/daisyclover-ci" ]
  printf 'INSPECT_CLAW_NODE_3_CI_NPM_SELF_TEST_OK\n'
  exit 0
fi

if [ "$#" -ne 0 ]; then
  echo "Usage: inspect-claw-node-3-ci-npm.sh [--self-test]" >&2
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
[ "$node_name" = "$TARGET_NODE" ] || { echo "ERROR: target node identity mismatch" >&2; exit 1; }

SSH=(ssh -o BatchMode=yes -o ConnectTimeout=10 -o StrictHostKeyChecking=yes "$TARGET_NODE")

printf 'CI_NPM_DIAGNOSTIC\n'
printf 'target=%s\n' "$TARGET_NODE"
"${SSH[@]}" 'set -eu
printf "[cache]\n"
sudo -n du -sh /home/daisyclover-ci/.npm 2>/dev/null || echo missing
printf "[system-paths]\n"
for path in /usr/bin/npm /usr/local/bin/npm /bin/npm /usr/bin/node /usr/local/bin/node /bin/node; do
  if [ -e "$path" ] || [ -L "$path" ]; then
    ls -l "$path"
  fi
done
printf "[ci-login-path]\n"
sudo -n -u daisyclover-ci sh -lc "printf \"PATH=%s\\n\" \"\$PATH\"; command -v node || true; command -v npm || true"
printf "[nvm-bins]\n"
if [ -d /home/daisyclover-ci/.nvm/versions/node ]; then
  sudo -n find /home/daisyclover-ci/.nvm/versions/node -maxdepth 3 -type f \( -name npm -o -name node \) -print 2>/dev/null | sort
else
  echo missing
fi
printf "[npm-cache-metadata]\n"
sudo -n find /home/daisyclover-ci/.npm -mindepth 1 -maxdepth 1 -printf "%f|modified=%TY-%Tm-%TdT%TH:%TM:%TS|owner=%u:%g\n" 2>/dev/null | sort || true
'
