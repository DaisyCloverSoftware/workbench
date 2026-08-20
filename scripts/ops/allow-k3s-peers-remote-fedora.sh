#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 3 || $# -gt 5 ]]; then
  echo "usage: $0 <user> <host-ipv4> <peer-ipv4> [peer-ipv4...]" >&2
  exit 2
fi
USER_NAME="$1"
HOST_IP="$2"
shift 2
[[ "$HOST_IP" =~ ^192\.168\.1\.[0-9]{1,3}$ ]] || { echo "host must be on 192.168.1.0/24" >&2; exit 2; }
for peer in "$@"; do
  [[ "$peer" =~ ^192\.168\.1\.[0-9]{1,3}$ ]] || { echo "peer must be on 192.168.1.0/24" >&2; exit 2; }
done

KEY=""
for candidate in "$HOME"/.ssh/id_*; do
  [[ -f "$candidate" && "$candidate" != *.pub ]] || continue
  if ssh -i "$candidate" -o IdentitiesOnly=yes -o BatchMode=yes -o ConnectTimeout=4 -o StrictHostKeyChecking=yes "$USER_NAME@$HOST_IP" true >/dev/null 2>&1; then
    KEY="$candidate"
    break
  fi
done
[[ -n "$KEY" ]] || { echo "no usable SSH identity for $USER_NAME@$HOST_IP" >&2; exit 1; }

for peer in "$@"; do
  ssh -i "$KEY" -o IdentitiesOnly=yes -o BatchMode=yes -o ConnectTimeout=8 -o StrictHostKeyChecking=yes "$USER_NAME@$HOST_IP" \
    sudo -n firewall-cmd --permanent --add-rich-rule="rule family=ipv4 source address=${peer}/32 port port=8472 protocol=udp accept" >/dev/null
  ssh -i "$KEY" -o IdentitiesOnly=yes -o BatchMode=yes -o ConnectTimeout=8 -o StrictHostKeyChecking=yes "$USER_NAME@$HOST_IP" \
    sudo -n firewall-cmd --permanent --add-rich-rule="rule family=ipv4 source address=${peer}/32 port port=10250 protocol=tcp accept" >/dev/null
done
ssh -i "$KEY" -o IdentitiesOnly=yes -o BatchMode=yes -o ConnectTimeout=8 -o StrictHostKeyChecking=yes "$USER_NAME@$HOST_IP" sudo -n firewall-cmd --reload >/dev/null
ssh -i "$KEY" -o IdentitiesOnly=yes -o BatchMode=yes -o ConnectTimeout=8 -o StrictHostKeyChecking=yes "$USER_NAME@$HOST_IP" sudo -n firewall-cmd --list-rich-rules
