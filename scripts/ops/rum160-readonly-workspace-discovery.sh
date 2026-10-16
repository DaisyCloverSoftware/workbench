#!/usr/bin/env bash
# RUM-only, read-only worktree discovery. Execute from reviewed exact Workbench main.
set -euo pipefail
if [ "$#" -ne 0 ]; then echo 'error=arguments-not-accepted' >&2; exit 64; fi
printf 'diagnostic=rum160-readonly-workspaces-v1\n'
printf 'host=%s\n' "$(hostname)"
printf 'execution_head=%s\n' "$(git rev-parse HEAD 2>/dev/null || echo unavailable)"
pr_branch=fix-rat-search-catalogue-state-20260903
reported_pr_sha=b618c7105bfee4a47e1923a9b4b024967d5c58c9
count=0
# Explicitly known RUM roots, not an arbitrary filesystem search.
for candidate in /home/matt/src/rum /Workstations/Projects/rum; do
  if [ ! -d "$candidate" ]; then printf 'candidate_missing=%s\n' "$candidate"; continue; fi
  root="$(cd "$candidate" && pwd -P)"
  repo="$(git -C "$root" rev-parse --show-toplevel 2>/dev/null || true)"
  if [ "$repo" != "$root" ]; then printf 'candidate_unverified=%s\n' "$candidate"; continue; fi
  origin="$(git -C "$root" remote get-url origin 2>/dev/null || true)"
  case "$origin" in
    https://github.com/DaisyCloverSoftware/rum|https://github.com/DaisyCloverSoftware/rum.git|git@github.com:DaisyCloverSoftware/rum|git@github.com:DaisyCloverSoftware/rum.git|ssh://git@github.com/DaisyCloverSoftware/rum|ssh://git@github.com/DaisyCloverSoftware/rum.git) ;;
    *) printf 'candidate_wrong_origin=%s\n' "$candidate"; continue ;;
  esac
  printf 'verified_root=%s\n' "$root"
  # Parsing a Git-owned registry is deliberately confined to the verified RUM checkout.
  while IFS= read -r path; do
    [ -n "$path" ] || continue
    [ "$count" -lt 32 ] || { printf 'error=worktree-limit\n'; exit 70; }
    count=$((count+1))
    printf 'worktree_%d_path=%s\n' "$count" "$path"
    if [ ! -d "$path" ]; then printf 'worktree_%d_missing=1\n' "$count"; continue; fi
    actual="$(git -C "$path" rev-parse --show-toplevel 2>/dev/null || true)"
    if [ "$actual" != "$path" ]; then printf 'worktree_%d_unverified=1\n' "$count"; continue; fi
    wt_origin="$(git -C "$path" remote get-url origin 2>/dev/null || true)"
    if [ "$wt_origin" != "$origin" ]; then printf 'worktree_%d_origin_mismatch=1\n' "$count"; continue; fi
    printf 'worktree_%d_head=%s\n' "$count" "$(git -C "$path" rev-parse HEAD)"
    printf 'worktree_%d_branch=%s\n' "$count" "$(git -C "$path" symbolic-ref --quiet --short HEAD || echo detached)"
    printf 'worktree_%d_staged_files=%s\n' "$count" "$(git -C "$path" diff --cached --name-only | wc -l)"
    printf 'worktree_%d_unstaged_files=%s\n' "$count" "$(git -C "$path" diff --name-only | wc -l)"
    printf 'worktree_%d_untracked_files=%s\n' "$count" "$(git -C "$path" ls-files --others --exclude-standard | wc -l)"
    git -C "$path" status --porcelain=v1 --untracked-files=all | head -n 200 | sed "s/^/worktree_${count}_status=/"
    # No network fetch, no mutation, and no claim that a cached remote ref is fresh.
    if git -C "$path" rev-parse --verify --quiet "refs/remotes/origin/$pr_branch^{commit}" >/dev/null; then
      printf 'worktree_%d_cached_remote_pr_sha=%s\n' "$count" "$(git -C "$path" rev-parse "refs/remotes/origin/$pr_branch")"
      git -C "$path" log --format=%H --max-count=30 "refs/remotes/origin/$pr_branch..HEAD" | sed "s/^/worktree_${count}_local_only_commit=/"
    else
      printf 'worktree_%d_cached_remote_pr_ref=absent\n' "$count"
    fi
    if git -C "$path" cat-file -e "${reported_pr_sha}^{commit}" 2>/dev/null; then
      printf 'worktree_%d_reported_pr_sha_available=1\n' "$count"
    else printf 'worktree_%d_reported_pr_sha_available=0\n' "$count"; fi
  done < <(git -C "$root" worktree list --porcelain | sed -n 's/^worktree //p')
done
printf 'verified_worktrees=%d\n' "$count"
