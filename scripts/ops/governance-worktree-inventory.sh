#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 1 ]; then
  printf 'error=target-required\n' >&2
  exit 64
fi

target="$1"
if [ ! -d "$target" ]; then
  printf 'error=target-missing\n' >&2
  exit 65
fi

target="$(cd "$target" && pwd -P)"
root="$(git -C "$target" rev-parse --show-toplevel 2>/dev/null || true)"
if [ -z "$root" ]; then
  printf 'error=not-git-checkout\n' >&2
  exit 66
fi
root="$(cd "$root" && pwd -P)"
if [ "$root" != "$target" ]; then
  printf 'error=target-not-repository-root\n' >&2
  exit 67
fi

origin="$(git -C "$root" remote get-url origin 2>/dev/null || true)"
origin_ok=0
case "$origin" in
  https://github.com/DaisyCloverSoftware/workbench|https://github.com/DaisyCloverSoftware/workbench.git|git@github.com:DaisyCloverSoftware/workbench|git@github.com:DaisyCloverSoftware/workbench.git|ssh://git@github.com/DaisyCloverSoftware/workbench|ssh://git@github.com:DaisyCloverSoftware/workbench.git)
    origin_ok=1
    ;;
esac
if [ "$origin_ok" -ne 1 ] && [ "${WORKBENCH_GOVERNANCE_TEST_ALLOW_LOCAL_ORIGIN:-0}" = "1" ] && [ "${WORKBENCH_OPERATION_SCRIPT:-0}" != "1" ]; then
  origin_ok=1
fi
if [ "$origin_ok" -ne 1 ]; then
  printf 'error=unexpected-origin\n' >&2
  exit 68
fi

count=0
head=''
branch=''
prunable=0
worktree_path=''

emit_record() {
  if [ "$count" -eq 0 ]; then
    return
  fi
  local status_text=''
  local status_error=0
  if [ ! -d "$worktree_path" ]; then
    status_error=1
  else
    if ! status_text="$(git -C "$worktree_path" status --porcelain=v1 --untracked-files=all 2>/dev/null)"; then
      status_error=1
    fi
  fi
  local dirty=0
  if [ "$status_error" -ne 0 ] || [ -n "$status_text" ]; then
    dirty=1
  fi
  printf 'worktree_%d_head=%s\n' "$count" "$head"
  if [ -n "$branch" ]; then
    printf 'worktree_%d_branch=%s\n' "$count" "$branch"
  else
    printf 'worktree_%d_branch=detached\n' "$count"
  fi
  printf 'worktree_%d_prunable=%d\n' "$count" "$prunable"
  printf 'worktree_%d_dirty=%d\n' "$count" "$dirty"
  if [ "$status_error" -ne 0 ]; then
    printf 'worktree_%d_dirty_entry_1=status-unavailable\n' "$count"
  elif [ -n "$status_text" ]; then
    local entry=0
    while IFS= read -r status_line || [ -n "$status_line" ]; do
      entry=$((entry + 1))
      # Git porcelain paths are repository-relative. Do not emit worktree paths.
      printf 'worktree_%d_dirty_entry_%d=%s\n' "$count" "$entry" "$status_line"
    done <<< "$status_text"
  fi
}

while IFS= read -r line || [ -n "$line" ]; do
  case "$line" in
    worktree\ *)
      if [ "$count" -gt 0 ]; then
        emit_record
      fi
      count=$((count + 1))
      worktree_path="${line#worktree }"
      head=''
      branch=''
      prunable=0
      ;;
    HEAD\ *)
      head="${line#HEAD }"
      ;;
    branch\ refs/heads/*)
      branch="${line#branch refs/heads/}"
      ;;
    prunable*)
      prunable=1
      ;;
  esac
done < <(git -C "$root" worktree list --porcelain)

if [ "$count" -gt 0 ]; then
  emit_record
fi
printf 'worktree_count=%d\n' "$count"
