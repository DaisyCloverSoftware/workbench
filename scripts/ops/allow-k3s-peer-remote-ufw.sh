#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 3 ]]; then
  echo "usage: $0 <user> <host-ipv4> <peer-ipv4>" >&2
  exit 2
fi
USER_NAME="$1"
HOST_IP="$2"
PEER="$3"
for ip in "$HOST_IP" "$PEER"; do
  [[ "$ip" =~ ^192\.168\.1\.[0-9]{1,3}$ ]] || { echo "addresses must be on 192.168.1.0/24" >&2; exit 2; }
done
KEY=""
for candidate in "$HOME"/.ssh/id_*; do
  [[ -f "$candidate" && "$candidate" != *.pub ]] || continue
  if ssh -i "$candidate" -o IdentitiesOnly=yes -o BatchMode=yes -o ConnectTimeout=4 -o StrictHostKeyChecking=accept-new "$USER_NAME@$HOST_IP" true >/dev/null 2>&1; then KEY="$candidate"; break; fi
done
[[ -n "$KEY" ]] || { echo "no usable SSH identity for $USER_NAME@$HOST_IP" >&2; exit 1; }
SSH=(ssh -i "$KEY" -o IdentitiesOnly=yes -o BatchMode=yes -o ConnectTimeout=8 -o StrictHostKeyChecking=accept-new "$USER_NAME@$HOST_IP")
"${SSH[@]}" "sudo -n ufw allow from $PEER to any port 8472 proto udp comment 'flannel vxlan claw-runner-1'" >/dev/null
"${SSH[@]}" "sudo -n ufw allow from $PEER to any port 10250 proto tcp comment 'kubelet claw-runner-1'" >/dev/null
"${SSH[@]}" "sudo -n ufw status numbered | grep -F '$PEER'"
