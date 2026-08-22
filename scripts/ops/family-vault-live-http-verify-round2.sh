#!/usr/bin/env bash
set -euo pipefail
umask 077

LIVE_URL="https://family-vault.co.uk"

for command in curl grep mktemp tr head; do
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

curl --fail --silent --show-error --location --max-time 20 -H 'Cache-Control: no-cache' "$LIVE_URL/api/health" > "$health_body"
curl --fail --silent --show-error --location --max-time 30 -H 'Cache-Control: no-cache' "$LIVE_URL/" > "$home_body"

for marker in '"status":"ok"' '"service":"family-vault-web"'; do
  grep -Fq "$marker" "$health_body" || {
    echo "ERROR: Family Vault LIVE health response is missing $marker" >&2
    exit 4
  }
done

require_home_marker() {
  local marker="$1"
  local label="$2"
  if ! grep -Fq "$marker" "$home_body"; then
    echo "ERROR: Family Vault LIVE homepage is missing marker: $label" >&2
    exit 5
  fi
  printf 'FAMILY_VAULT_LIVE_MARKER_%s=true\n' "$label"
}

require_home_marker 'Relationship Model' 'RELATIONSHIP_MODEL'
require_home_marker 'Archive Content' 'ARCHIVE_CONTENT'
require_home_marker 'data-home-support="design-heritage"' 'DESIGN_HERITAGE'
require_home_marker 'data-faq-slot="primary-left"' 'FAQ_PRIMARY_LEFT'
require_home_marker 'data-faq-slot="primary-right"' 'FAQ_PRIMARY_RIGHT'
require_home_marker 'data-faq-slot="storage"' 'FAQ_STORAGE'
require_home_marker 'data-faq-slot="design-heritage"' 'FAQ_DESIGN_HERITAGE'
require_home_marker 'data-faq-slot="audience"' 'FAQ_AUDIENCE'
require_home_marker 'Why do some Family Vault apps feel familiar?' 'FAQ_FAMILIAR_QUESTION'

if grep -Fq 'data-faq-wide' "$home_body"; then
  echo 'ERROR: retired Round 1 data-faq-wide marker is still present on LIVE' >&2
  exit 6
fi

health_compact="$(tr -d '\r\n' < "$health_body" | head -c 240)"
printf 'FAMILY_VAULT_LIVE_HEALTH_BODY=%s\n' "$health_compact"
echo 'FAMILY_VAULT_LIVE_HEALTH_HTTP_PASS=true'
echo 'FAMILY_VAULT_LIVE_HOMEPAGE_HTTP_PASS=true'
echo 'FAMILY_VAULT_LIVE_ROUND2_SERVER_RENDERED_MARKERS_PASS=true'
