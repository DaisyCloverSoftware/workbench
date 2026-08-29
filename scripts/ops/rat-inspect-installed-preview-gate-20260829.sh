#!/usr/bin/env bash
set -euo pipefail

for name in workbench-relay workbench-runner workbench-server; do
  bin="$HOME/.local/bin/$name"
  printf '%s_bin=%s\n' "$name" "$bin"
  if [[ ! -x "$bin" ]]; then
    printf '%s_executable=no\n' "$name"
    continue
  fi
  printf '%s_executable=yes\n' "$name"
  for pair in \
    'cf92:cf92e170c8b8728cb59c5b22c424e6472d048b49' \
    '9aa8:9aa8a7a65707c4dabc4d32d58def100e4c1cefaad907d5d4390543b7c9b4df54' \
    '60cab55:60cab55d5bd868913da833e60c15cbe938afa494' \
    '7b3576:7b3576d66dd288ec8309b6c62a67d81592ae896176420c0ec8f4cbae58d7e74e'
  do
    key="${pair%%:*}"
    value="${pair#*:}"
    if strings "$bin" | grep -Fq "$value"; then
      printf '%s_has_%s=yes\n' "$name" "$key"
    else
      printf '%s_has_%s=no\n' "$name" "$key"
    fi
  done
done
