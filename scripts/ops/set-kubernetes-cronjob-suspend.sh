#!/usr/bin/env bash
set -euo pipefail

if [ "${1:-}" = "--self-test" ]; then
  [ "$#" -eq 1 ]
  printf 'SET_KUBERNETES_CRONJOB_SUSPEND_SELF_TEST_OK\n'
  exit 0
fi

if [ "$#" -ne 3 ]; then
  echo "Usage: set-kubernetes-cronjob-suspend.sh <namespace> <cronjob> <true|false>" >&2
  exit 2
fi

NAMESPACE="$1"
NAME="$2"
SUSPEND="$3"
case "$NAMESPACE" in *[!A-Za-z0-9._-]*|'') echo "ERROR: invalid namespace" >&2; exit 2;; esac
case "$NAME" in *[!A-Za-z0-9._-]*|'') echo "ERROR: invalid cronjob name" >&2; exit 2;; esac
case "$SUSPEND" in true|false) ;; *) echo "ERROR: suspend must be true or false" >&2; exit 2;; esac

if sudo -n kubectl version --client >/dev/null 2>&1; then
  KUBECTL=(sudo -n kubectl)
elif sudo -n k3s kubectl version --client >/dev/null 2>&1; then
  KUBECTL=(sudo -n k3s kubectl)
else
  echo "ERROR: no sanctioned non-interactive Kubernetes client is available" >&2
  exit 1
fi

printf '[before]\n'
"${KUBECTL[@]}" get cronjob "$NAME" -n "$NAMESPACE" -o 'custom-columns=NAME:.metadata.name,SUSPEND:.spec.suspend,SCHEDULE:.spec.schedule,ACTIVE:.status.active[*].name'
"${KUBECTL[@]}" patch cronjob "$NAME" -n "$NAMESPACE" --type=merge -p "{\"spec\":{\"suspend\":$SUSPEND}}"
printf '[after]\n'
"${KUBECTL[@]}" get cronjob "$NAME" -n "$NAMESPACE" -o 'custom-columns=NAME:.metadata.name,SUSPEND:.spec.suspend,SCHEDULE:.spec.schedule,ACTIVE:.status.active[*].name'
