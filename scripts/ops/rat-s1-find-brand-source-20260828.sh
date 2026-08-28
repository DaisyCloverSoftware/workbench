#!/usr/bin/env bash
set -euo pipefail

# Read-only, bounded recovery probe for the two owner-supplied RAT brand images
# that were committed with invalid WebP bytes. Do not print file contents.
# Search only ephemeral/cache/workbench locations; never traverse secrets,
# Kubernetes data, repositories outside the bounded known roots, or / broadly.

roots=(
  "$HOME/.cache"
  "$HOME/.local/share/workbench"
  "$HOME/workbench"
  "$HOME/Workbench"
  "/tmp"
)

printf 'home=%s\n' "$HOME"
for root in "${roots[@]}"; do
  [[ -d "$root" ]] || continue
  printf 'root=%s\n' "$root"
  find "$root" -xdev -type f \
    \( -iname '*rat*wordmark*' -o -iname '*pirate*mascot*' -o -iname '*rat*brand*' -o -iname '*rum*thumb*' \) \
    -newermt '2026-08-25 00:00:00' ! -newermt '2026-08-28 23:59:59' \
    -printf '%TY-%Tm-%TdT%TH:%TM:%TS|%s|%p\n' 2>/dev/null \
    | head -n 200

done

# Also identify image-like files created during the owner-artwork handoff window,
# but only under likely Workbench cache/tmp roots and only by magic signature.
for root in "$HOME/.cache" /tmp; do
  [[ -d "$root" ]] || continue
  while IFS= read -r -d '' path; do
    kind="$(file -b --mime-type "$path" 2>/dev/null || true)"
    case "$kind" in
      image/webp|image/png|image/jpeg)
        printf 'image|%s|%s|%s\n' "$kind" "$(stat -c '%s' "$path" 2>/dev/null || echo '?')" "$path"
        ;;
    esac
  done < <(find "$root" -xdev -type f \
    -newermt '2026-08-26 12:00:00' ! -newermt '2026-08-27 00:00:00' \
    -size +5k -size -20M -print0 2>/dev/null | head -z -n 3000)
done
