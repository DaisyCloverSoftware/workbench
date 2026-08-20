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
SSH=(ssh -i "$KEY" -o IdentitiesOnly=yes -o BatchMode=yes -o ConnectTimeout=8 -o StrictHostKeyChecking=yes "$USER_NAME@$HOST_IP")

for peer in "$@"; do
  for spec in '8472 udp' '10250 tcp'; do
    read -r port proto <<<"$spec"
    rule="rule family=ipv4 source address=${peer}/32 port port=${port} protocol=${proto} accept"
    printf -v qrule '%q' "$rule"
    "${SSH[@]}" "sudo -n firewall-cmd --permanent --add-rich-rule=$qrule" >/dev/null
  done
done
"${SSH[@]}" 'sudo -n firewall-cmd --reload' >/dev/null
"${SSH[@]}" 'sudo -n firewall-cmd --list-rich-rules'
