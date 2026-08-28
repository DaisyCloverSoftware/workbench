#!/usr/bin/env bash
set -eu

ROOT="$HOME/.cache/Workbench/task-workspaces"
[[ -d "$ROOT" ]] || { echo 'NO_TASK_WORKSPACES'; exit 0; }

echo '--- cached RUM workspaces ---'
for ws in "$ROOT"/*; do
  [[ -d "$ws/.git" || -f "$ws/.git" ]] || continue
  origin="$(git -C "$ws" remote get-url origin 2>/dev/null || true)"
  case "$origin" in
    *DaisyCloverSoftware/rum*|*daisycloversoftware/rum*) ;;
    *) continue ;;
  esac
  head="$(git -C "$ws" rev-parse HEAD 2>/dev/null || true)"
  branch="$(git -C "$ws" branch --show-current 2>/dev/null || true)"
  printf 'workspace|%s|%s|%s\n' "$head" "$branch" "$ws"

  count=0
  while IFS= read -r -d '' path; do
    case "$path" in */.git/*|*/node_modules/*|*/vendor/*) continue ;; esac
    mime="$(file -b --mime-type "$path" 2>/dev/null || true)"
    case "$mime" in
      image/webp|image/png|image/jpeg)
        bytes="$(stat -c '%s' "$path" 2>/dev/null || echo '?')"
        sha="$(sha256sum "$path" 2>/dev/null | awk '{print $1}' || true)"
        printf 'image|%s|%s|%s|%s\n' "$mime" "$bytes" "$sha" "$path"
        count=$((count+1))
        [[ "$count" -lt 1000 ]] || break
        ;;
    esac
  done < <(find "$ws" -xdev -type f -size +5k -size -30M -print0 2>/dev/null)
done

echo 'RUM_WORKSPACE_ARTWORK_SCAN_DONE'
