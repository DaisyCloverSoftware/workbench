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
case "$origin" in
  https://github.com/DaisyCloverSoftware/workbench|https://github.com/DaisyCloverSoftware/workbench.git|git@github.com:DaisyCloverSoftware/workbench|git@github.com:DaisyCloverSoftware/workbench.git|ssh://git@github.com/DaisyCloverSoftware/workbench|ssh://git@github.com/DaisyCloverSoftware/workbench.git)
    ;;
  *)
    printf 'error=unexpected-origin\n' >&2
    exit 68
    ;;
esac

count=0
head=''
branch=''
detached=0
prunable=0

emit_record() {
  if [ "$count" -eq 0 ]; then
    return
  fi
  printf 'worktree_%d_head=%s\n' "$count" "$head"
  if [ -n "$branch" ]; then
    printf 'worktree_%d_branch=%s\n' "$count" "$branch"
  else
    printf 'worktree_%d_branch=detached\n' "$count"
  fi
  printf 'worktree_%d_prunable=%d\n' "$count" "$prunable"
}

while IFS= read -r line || [ -n "$line" ]; do
  case "$line" in
    worktree\ *)
      if [ "$count" -gt 0 ]; then
        emit_record
      fi
      count=$((count + 1))
      head=''
      branch=''
      detached=0
      prunable=0
      ;;
    HEAD\ *)
      head="${line#HEAD }"
      ;;
    branch\ refs/heads/*)
      branch="${line#branch refs/heads/}"
      ;;
    detached)
      detached=1
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
