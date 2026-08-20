#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <peer-ipv4>" >&2
  exit 2
fi
PEER="$1"
[[ "$PEER" =~ ^192\.168\.1\.[0-9]{1,3}$ ]] || { echo "peer must be on 192.168.1.0/24" >&2; exit 2; }

sudo -n ufw allow from "$PEER" to any port 8472 proto udp comment 'flannel vxlan k3s peer' >/dev/null
sudo -n ufw allow from "$PEER" to any port 10250 proto tcp comment 'kubelet k3s peer' >/dev/null
sudo -n ufw status numbered | grep -F "$PEER"
