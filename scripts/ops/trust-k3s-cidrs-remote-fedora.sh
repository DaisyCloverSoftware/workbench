#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: $0 <user> <host-ipv4>" >&2
  exit 2
fi
USER_NAME="$1"
HOST_IP="$2"
[[ "$HOST_IP" =~ ^192\.168\.1\.[0-9]{1,3}$ ]] || { echo "host must be on 192.168.1.0/24" >&2; exit 2; }

KEY=""
for candidate in "$HOME"/.ssh/id_*; do
  [[ -f "$candidate" && "$candidate" != *.pub ]] || continue
  if ssh -i "$candidate" -o IdentitiesOnly=yes -o BatchMode=yes -o ConnectTimeout=4 -o StrictHostKeyChecking=yes "$USER_NAME@$HOST_IP" true >/dev/null 2>&1; then
    KEY="$candidate"
    break
  fi
done
[[ -n "$KEY" ]] || { echo "no usable SSH identity for $USER_NAME@$HOST_IP" >&2; exit 1; }
SSH=(ssh -i "$KEY" -o IdentitiesOnly=yes -o BatchMode=yes -o ConnectTimeout=8 -o StrictHostKeyChecking=yes "$USER_NAME@$HOST_IP")

"${SSH[@]}" 'sudo -n firewall-cmd --permanent --zone=trusted --add-source=10.42.0.0/16' >/dev/null
"${SSH[@]}" 'sudo -n firewall-cmd --permanent --zone=trusted --add-source=10.43.0.0/16' >/dev/null
"${SSH[@]}" 'sudo -n firewall-cmd --reload' >/dev/null
"${SSH[@]}" 'sudo -n firewall-cmd --zone=trusted --list-sources'
