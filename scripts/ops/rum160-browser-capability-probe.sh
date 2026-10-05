#!/usr/bin/env bash
set -euo pipefail

[[ $# -eq 0 ]] || { echo 'usage: rum160-browser-capability-probe.sh' >&2; exit 64; }

for command in node npm npx python3 curl; do
  if path="$(command -v "$command" 2>/dev/null)"; then
    printf '%s=%s\n' "${command^^}_PATH" "$path"
    "$command" --version 2>/dev/null | head -n 1 | sed "s/^/${command^^}_VERSION=/" || true
  else
    printf '%s=unavailable\n' "${command^^}_PATH"
  fi
done

browser=''
for candidate in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser; do
  if [[ -x "$candidate" ]]; then
    browser="$candidate"
    break
  fi
done
if [[ -n "$browser" ]]; then
  printf 'BROWSER_PATH=%s\n' "$browser"
  "$browser" --version | sed 's/^/BROWSER_VERSION=/'
else
  echo 'BROWSER_PATH=unavailable'
fi

node - <<'NODE' || true
for (const name of ['@playwright/test','playwright','puppeteer','ws']) {
  try {
    const resolved = require.resolve(name)
    console.log(`NODE_MODULE_${name.replace(/[^A-Za-z0-9]/g, '_').toUpperCase()}=${resolved}`)
  } catch (_) {
    console.log(`NODE_MODULE_${name.replace(/[^A-Za-z0-9]/g, '_').toUpperCase()}=unavailable`)
  }
}
NODE
