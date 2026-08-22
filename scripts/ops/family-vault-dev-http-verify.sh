#!/usr/bin/env bash
set -euo pipefail
umask 077

DEV_URL="https://dev.family-vault.co.uk"

for command in curl grep mktemp; do
  command -v "$command" >/dev/null 2>&1 || {
    echo "ERROR: required command is unavailable: $command" >&2
    exit 3
  }
done

health_body="$(mktemp)"
home_body="$(mktemp)"
cleanup() {
  rm -f "$health_body" "$home_body"
}
trap cleanup EXIT HUP INT TERM

curl --fail --silent --show-error --location --max-time 20 "$DEV_URL/api/health" > "$health_body"
curl --fail --silent --show-error --location --max-time 30 "$DEV_URL/" > "$home_body"

[ -s "$health_body" ] || {
  echo "ERROR: Family Vault DEV health response was empty" >&2
  exit 4
}
[ -s "$home_body" ] || {
  echo "ERROR: Family Vault DEV homepage response was empty" >&2
  exit 5
}

require_home_marker() {
  local marker="$1"
  local label="$2"
  if ! grep -Fq "$marker" "$home_body"; then
    echo "ERROR: Family Vault DEV homepage is missing marker: $label" >&2
    exit 6
  fi
  printf 'FAMILY_VAULT_DEV_MARKER_%s=true\n' "$label"
}

require_home_marker 'Relationship Model' 'RELATIONSHIP_MODEL'
require_home_marker 'Archive Content' 'ARCHIVE_CONTENT'
require_home_marker 'data-home-support="design-heritage"' 'DESIGN_HERITAGE'
require_home_marker 'data-faq-wide' 'FAQ_WIDE'
require_home_marker 'Cloudflare Turnstile' 'CLOUDFLARE_TURNSTILE'

printf 'FAMILY_VAULT_DEV_HEALTH_BODY=%s\n' "$(tr -d '\r\n' < "$health_body" | head -c 240)"
echo "FAMILY_VAULT_DEV_HEALTH_HTTP_PASS=true"
echo "FAMILY_VAULT_DEV_HOMEPAGE_HTTP_PASS=true"
echo "FAMILY_VAULT_DEV_SPRINT0_HOMEPAGE_MARKERS_PASS=true"
