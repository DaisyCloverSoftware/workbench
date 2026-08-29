#!/usr/bin/env bash
set -euo pipefail

for name in workbench-relay workbench-runner workbench-server; do
  bin="$(command -v "$name" 2>/dev/null || true)"
  printf '%s_bin=%s\n' "$name" "${bin:-missing}"
  if [[ -z "$bin" ]]; then
    continue
  fi
  if strings "$bin" | grep -Fq 'cf92e170c8b8728cb59c5b22c424e6472d048b49'; then
    printf '%s_has_cf92=yes\n' "$name"
  else
    printf '%s_has_cf92=no\n' "$name"
  fi
  if strings "$bin" | grep -Fq '9aa8a7a65707c4dabc4d32d58def100e4c1cefaad907d5d4390543b7c9b4df54'; then
    printf '%s_has_9aa8=yes\n' "$name"
  else
    printf '%s_has_9aa8=no\n' "$name"
  fi
  if strings "$bin" | grep -Fq '60cab55d5bd868913da833e60c15cbe938afa494'; then
    printf '%s_has_60cab55=yes\n' "$name"
  else
    printf '%s_has_60cab55=no\n' "$name"
  fi
  if strings "$bin" | grep -Fq '7b3576d66dd288ec8309b6c62a67d81592ae896176420c0ec8f4cbae58d7e74e'; then
    printf '%s_has_7b3576=yes\n' "$name"
  else
    printf '%s_has_7b3576=no\n' "$name"
  fi
done
