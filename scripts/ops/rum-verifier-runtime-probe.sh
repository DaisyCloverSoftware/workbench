#!/usr/bin/env bash
set -euo pipefail

for command in docker podman nerdctl k3s sudo; do
  if command -v "$command" >/dev/null 2>&1; then
    printf '%s=%s\n' "$command" "$(command -v "$command")"
  else
    printf '%s=absent\n' "$command"
  fi
done

if command -v k3s >/dev/null 2>&1; then
  printf '%s\n' '[k3s-ctr-run-help]'
  sudo k3s ctr run --help | sed -n '1,180p'
fi
