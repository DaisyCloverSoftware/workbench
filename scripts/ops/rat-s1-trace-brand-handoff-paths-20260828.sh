#!/usr/bin/env bash
set -eu

ROOTS=("$HOME/.cache/Workbench" "$HOME/.local/share/workbench")
PATTERN='6fd1bba95e9eedb28b33d4b00bd4a8b465924213|owner-approved RAT brand|owner approved RAT brand|rat-wordmark|rat-pirate-mascot|pirate-rat|pirate rat'

echo '--- matching metadata/log paths only ---'
for root in "${ROOTS[@]}"; do
  [[ -d "$root" ]] || continue
  if command -v rg >/dev/null 2>&1; then
    rg -l -I -i --hidden --max-filesize 5M \
      -g '!**/.git/objects/**' -g '!**/node_modules/**' -g '!**/vendor/**' \
      -e "$PATTERN" "$root" 2>/dev/null | while IFS= read -r path; do
        printf '%s|%s|%s\n' "$(stat -c '%s' "$path" 2>/dev/null || echo '?')" "$(stat -c '%y' "$path" 2>/dev/null | cut -d'.' -f1 || true)" "$path"
      done | head -n 300 || true
  else
    find "$root" -xdev -type f -size -5M ! -path '*/.git/objects/*' ! -path '*/node_modules/*' ! -path '*/vendor/*' -print0 2>/dev/null \
      | xargs -0 grep -Eil "$PATTERN" 2>/dev/null | while IFS= read -r path; do
        printf '%s|%s|%s\n' "$(stat -c '%s' "$path" 2>/dev/null || echo '?')" "$(stat -c '%y' "$path" 2>/dev/null | cut -d'.' -f1 || true)" "$path"
      done | head -n 300 || true
  fi
done

echo 'RAT_BRAND_HANDOFF_PATH_TRACE_DONE'
