#!/usr/bin/env bash
set -euo pipefail

if [ "${1:-}" = "--self-test" ]; then
  [ "$#" -eq 1 ]
  printf 'PROBE_PUBLIC_HTTPS_SELF_TEST_OK\n'
  exit 0
fi

if [ "$#" -ne 2 ]; then
  echo "Usage: probe-public-https.sh <hostname> </path>" >&2
  exit 2
fi

HOST="$1"
PATH_PART="$2"
case "$HOST" in *[!A-Za-z0-9.-]*|'') echo "ERROR: invalid hostname" >&2; exit 2;; esac
case "$PATH_PART" in /*) ;; *) echo "ERROR: path must begin with /" >&2; exit 2;; esac
case "$PATH_PART" in *$'\n'*|*$'\r'*|*' '*) echo "ERROR: invalid path" >&2; exit 2;; esac

if ! command -v curl >/dev/null 2>&1; then
  echo "ERROR: curl unavailable" >&2
  exit 1
fi

URL="https://$HOST$PATH_PART"
printf 'url=%s\n' "$URL"
curl --silent --show-error --location --max-redirs 3 --max-time 20 --proto '=https' --tlsv1.2 --output /dev/null --write-out 'status=%{http_code}\neffective_url=%{url_effective}\nremote_ip=%{remote_ip}\n' "$URL"
