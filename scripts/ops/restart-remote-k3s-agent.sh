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
  if ssh -i "$candidate" -o IdentitiesOnly=yes -o BatchMode=yes -o ConnectTimeout=4 -o StrictHostKeyChecking=yes "$USER_NAME@$HOST_IP" true >/dev/null 2>&1; then KEY="$candidate"; break; fi
done
[[ -n "$KEY" ]] || { echo "no usable SSH identity" >&2; exit 1; }
SSH=(ssh -i "$KEY" -o IdentitiesOnly=yes -o BatchMode=yes -o ConnectTimeout=8 -o StrictHostKeyChecking=yes "$USER_NAME@$HOST_IP")
"${SSH[@]}" 'sudo -n systemctl restart k3s-agent'
for _ in $(seq 1 30); do
  state="$("${SSH[@]}" 'sudo -n systemctl is-active k3s-agent' 2>/dev/null || true)"
  if [[ "$state" == active ]]; then
    "${SSH[@]}" 'sudo -n systemctl --no-pager -l status k3s-agent | head -20'
    exit 0
  fi
  sleep 2
done
echo "k3s-agent did not become active" >&2
exit 1
