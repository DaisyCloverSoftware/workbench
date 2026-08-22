#!/usr/bin/env bash
set -euo pipefail
umask 077

SECRET_NAME="family-vault-web-secret"
DEV_NS="family-vault-dev"
LIVE_NS="family-vault-live"
APPROVED_KEYS=(SMTP_HOST SMTP_PORT SMTP_SECURE SMTP_FROM)

for command in kubectl base64; do
  command -v "$command" >/dev/null 2>&1 || {
    echo "ERROR: required command is unavailable: $command" >&2
    exit 3
  }
done

for namespace in "$DEV_NS" "$LIVE_NS"; do
  kubectl get namespace "$namespace" >/dev/null
  kubectl -n "$namespace" get secret "$SECRET_NAME" >/dev/null

done

read_approved_value() {
  local namespace="$1"
  local key="$2"
  local encoded=""

  case "$key" in
    SMTP_HOST|SMTP_PORT|SMTP_SECURE|SMTP_FROM) ;;
    *)
      echo "ERROR: attempted to read a non-approved SMTP metadata key" >&2
      return 4
      ;;
  esac

  encoded="$(kubectl -n "$namespace" get secret "$SECRET_NAME" -o "jsonpath={.data.${key}}")"
  if [ -z "$encoded" ]; then
    printf '%s' '(unset)'
    return 0
  fi
  printf '%s' "$encoded" | base64 --decode
}

for namespace in "$DEV_NS" "$LIVE_NS"; do
  case "$namespace" in
    "$DEV_NS") prefix="DEV" ;;
    "$LIVE_NS") prefix="LIVE" ;;
    *) echo "ERROR: unexpected namespace" >&2; exit 5 ;;
  esac

  for key in "${APPROVED_KEYS[@]}"; do
    value="$(read_approved_value "$namespace" "$key")"
    case "$value" in
      *$'\n'*|*$'\r'*)
        echo "ERROR: approved SMTP metadata contains an unexpected newline" >&2
        exit 6
        ;;
    esac
    printf 'FAMILY_VAULT_%s_%s=%s\n' "$prefix" "$key" "$value"
  done
done

dev_host="$(read_approved_value "$DEV_NS" SMTP_HOST)"
live_host="$(read_approved_value "$LIVE_NS" SMTP_HOST)"
dev_port="$(read_approved_value "$DEV_NS" SMTP_PORT)"
live_port="$(read_approved_value "$LIVE_NS" SMTP_PORT)"
dev_secure="$(read_approved_value "$DEV_NS" SMTP_SECURE)"
live_secure="$(read_approved_value "$LIVE_NS" SMTP_SECURE)"
dev_from="$(read_approved_value "$DEV_NS" SMTP_FROM)"
live_from="$(read_approved_value "$LIVE_NS" SMTP_FROM)"

if [ "$dev_host" = "$live_host" ] && [ "$dev_port" = "$live_port" ] && [ "$dev_secure" = "$live_secure" ] && [ "$dev_from" = "$live_from" ]; then
  echo "FAMILY_VAULT_SMTP_METADATA_IDENTICAL=true"
else
  echo "FAMILY_VAULT_SMTP_METADATA_IDENTICAL=false"
fi

echo "FAMILY_VAULT_SMTP_METADATA_KEYS_ONLY=true"
