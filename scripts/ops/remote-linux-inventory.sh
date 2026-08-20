#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 2 || $# -gt 3 ]]; then
  echo "usage: $0 <user@host> <identity-file> [connect-timeout-seconds]" >&2
  exit 2
fi

host="$1"
key="$2"
timeout="${3:-8}"
ssh_opts=(-i "$key" -o IdentitiesOnly=yes -o BatchMode=yes -o ConnectTimeout="$timeout" -o StrictHostKeyChecking=yes)

run() {
  local title="$1"
  local command="$2"
  printf '\n=== %s ===\n' "$title"
  ssh "${ssh_opts[@]}" "$host" "$command"
}

run transport 'hostname; id; printf "SSH_OK\n"'
run os 'cat /etc/os-release; uname -a'
run cpu 'lscpu'
run memory 'free -h'
run disk 'lsblk -o NAME,SIZE,TYPE,FSTYPE,MOUNTPOINTS; df -hT /'
run network 'hostname -I; ip -brief address; ip route'
run security-services 'getenforce 2>/dev/null || true; systemctl is-enabled firewalld 2>/dev/null || true; systemctl is-active firewalld 2>/dev/null || true; timedatectl status'
run runtimes 'for x in k3s kubectl containerd docker podman; do if command -v "$x" >/dev/null 2>&1; then printf "%s=" "$x"; command -v "$x"; else printf "%s=absent\n" "$x"; fi; done'
printf '\n=== sudo-noninteractive ===\n'
if ssh "${ssh_opts[@]}" "$host" 'sudo -n true' >/dev/null 2>&1; then
  echo 'sudo_noninteractive=yes'
else
  echo 'sudo_noninteractive=no'
fi
