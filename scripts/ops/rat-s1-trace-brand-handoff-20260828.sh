#!/usr/bin/env bash
set -eu

ROOTS=(
  "$HOME/.cache/Workbench"
  "$HOME/.local/share/workbench"
)
PATTERN='6fd1bba95e9eedb28b33d4b00bd4a8b465924213|owner-approved RAT brand|owner approved RAT brand|rat-wordmark|rat-pirate-mascot|pirate-rat|pirate rat|RUM.*Thumb.*RAT'

echo '--- RAT brand handoff metadata matches ---'
for root in "${ROOTS[@]}"; do
  [[ -d "$root" ]] || continue
  if command -v rg >/dev/null 2>&1; then
    rg -n -I -i --hidden --max-filesize 5M \
      -g '!**/.git/objects/**' -g '!**/node_modules/**' -g '!**/vendor/**' \
      -e "$PATTERN" "$root" 2>/dev/null | head -n 300 || true
  else
    find "$root" -xdev -type f -size -5M \
      ! -path '*/.git/objects/*' ! -path '*/node_modules/*' ! -path '*/vendor/*' -print0 2>/dev/null \
      | xargs -0 grep -Eni "$PATTERN" 2>/dev/null | head -n 300 || true
  fi
done

echo '--- Candidate RAT-related task metadata filenames ---'
for root in "${ROOTS[@]}"; do
  [[ -d "$root" ]] || continue
  find "$root" -xdev -type f \
    \( -iname '*rat*' -o -iname '*rate*thing*' -o -iname '*brand*' -o -iname '*asset*' \) \
    -newermt '2026-08-26 00:00:00' ! -newermt '2026-08-27 23:59:59' \
    -printf '%TY-%Tm-%TdT%TH:%TM:%TS|%s|%p\n' 2>/dev/null | head -n 400 || true
done

echo 'RAT_BRAND_HANDOFF_TRACE_DONE'
