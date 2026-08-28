#!/usr/bin/env bash
set -eu

WB_CACHE="$HOME/.cache/Workbench"
WB_DATA="$HOME/.local/share/workbench"

printf 'home=%s\n' "$HOME"
printf '%s\n' '--- Workbench cache roots ---'
if [[ -d "$WB_CACHE" ]]; then
  find "$WB_CACHE" -mindepth 1 -maxdepth 2 -type d -printf '%p\n' 2>/dev/null | sort | head -n 300 || true
fi

printf '%s\n' '--- RAT brand paths in task/scratch trees ---'
for root in "$WB_CACHE/task-workspaces" "$WB_CACHE/scratch"; do
  [[ -d "$root" ]] || continue
  while IFS= read -r -d '' path; do
    mime="$(file -b --mime-type "$path" 2>/dev/null || true)"
    bytes="$(stat -c '%s' "$path" 2>/dev/null || echo '?')"
    sha="$(sha256sum "$path" 2>/dev/null | awk '{print $1}' || true)"
    printf '%s|%s|%s|%s\n' "$mime" "$bytes" "$sha" "$path"
  done < <(find "$root" -xdev -type f \
    \( -iname 'rat-wordmark.webp' -o -iname 'rat-pirate-mascot.webp' -o -iname 'rum-thumb.webp' \
       -o -iname '*rat*wordmark*' -o -iname '*pirate*mascot*' \) \
    -print0 2>/dev/null)
done

printf '%s\n' '--- likely image handoff files (Aug 26) ---'
for root in "$WB_CACHE" "$WB_DATA"; do
  [[ -d "$root" ]] || continue
  count=0
  while IFS= read -r -d '' path; do
    case "$path" in
      */.git/*|*/node_modules/*|*/vendor/*) continue ;;
    esac
    mime="$(file -b --mime-type "$path" 2>/dev/null || true)"
    case "$mime" in
      image/webp|image/png|image/jpeg)
        bytes="$(stat -c '%s' "$path" 2>/dev/null || echo '?')"
        sha="$(sha256sum "$path" 2>/dev/null | awk '{print $1}' || true)"
        printf '%s|%s|%s|%s\n' "$mime" "$bytes" "$sha" "$path"
        count=$((count+1))
        [[ "$count" -lt 500 ]] || break
        ;;
    esac
  done < <(find "$root" -xdev -type f \
    -newermt '2026-08-26 00:00:00' ! -newermt '2026-08-27 23:59:59' \
    -size +5k -size -30M -print0 2>/dev/null)
done

echo 'BRAND_SOURCE_PROBE_DONE'
