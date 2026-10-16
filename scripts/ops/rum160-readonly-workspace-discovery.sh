#!/usr/bin/env bash
# RUM-only inventory. Runtime execution requires reviewed exact Workbench main.
# The sole argument is the independently verified PR head, never a path or code.
set -euo pipefail
[[ $# -eq 1 && "$1" =~ ^[0-9a-f]{40}$ ]] || { echo 'error=exact-pr-head-required' >&2; exit 64; }
pr_head="$1"
for variable in ${!GIT_@}; do unset "$variable"; done
export GIT_OPTIONAL_LOCKS=0 GIT_TERMINAL_PROMPT=0 GIT_CONFIG_NOSYSTEM=1 GIT_CONFIG_GLOBAL=/dev/null LC_ALL=C
rg() {
  local directory="$1"; shift
  timeout 10s git --no-optional-locks -c core.fsmonitor=false -c core.hooksPath=/dev/null \
    -c core.quotePath=true -C "$directory" "$@"
}
output_bytes=0
emit() {
  local formatted
  printf -v formatted '%s=%q\n' "$1" "$2"
  output_bytes=$((output_bytes + ${#formatted}))
  [[ "$output_bytes" -le 196608 ]] || { echo 'error=output-limit' >&2; exit 70; }
  printf '%s' "$formatted"
}
origin_ok() {
  case "$1" in
    https://github.com/DaisyCloverSoftware/rum|https://github.com/DaisyCloverSoftware/rum.git|git@github.com:DaisyCloverSoftware/rum|git@github.com:DaisyCloverSoftware/rum.git|ssh://git@github.com/DaisyCloverSoftware/rum|ssh://git@github.com/DaisyCloverSoftware/rum.git) return 0 ;;
    *) return 1 ;;
  esac
}
# Match Workbench runner root precedence; do not invent workstation paths.
explicit=0
if [[ -n "${WORKBENCH_RUNNER_ROOTS:-}" ]]; then
  [[ "$WORKBENCH_RUNNER_ROOTS" != *$'\n'* && "$WORKBENCH_RUNNER_ROOTS" != *$'\r'* ]] || exit 64
  [[ "$WORKBENCH_RUNNER_ROOTS" != :* && "$WORKBENCH_RUNNER_ROOTS" != *: && "$WORKBENCH_RUNNER_ROOTS" != *::* ]] || exit 64
  IFS=: read -r -a roots <<< "$WORKBENCH_RUNNER_ROOTS"; explicit=1
elif [[ -n "${WORKBENCH_RUNNER_ROOT:-}" ]]; then
  roots=("$WORKBENCH_RUNNER_ROOT"); explicit=1
else
  [[ -n "${HOME:-}" ]] || { echo 'error=home-unavailable' >&2; exit 65; }
  roots=("$HOME/src" "$HOME/projects")
fi
[[ ${#roots[@]} -le 8 ]] || { echo 'error=root-limit' >&2; exit 65; }
for candidate in "${roots[@]}"; do
  [[ "$candidate" != *$'\n'* && "$candidate" != *$'\r'* ]] || exit 64
  if [[ ! -d "$candidate" && "$explicit" -eq 1 ]]; then echo 'error=configured-root-unavailable' >&2; exit 65; fi
done
emit diagnostic rum160-readonly-workspaces-v2
emit host "$(hostname)"
emit execution_head "$(rg . rev-parse HEAD)"
emit comparison_pr_head "$pr_head"
emit scope configured-runner-roots-rum-and-git-registered-worktrees
emit task_ownership not_assessed
emit artifact_contents not_read
attempted=0; verified=0; scope_roots=0
declare -A seen=()
for configured in "${roots[@]}"; do
  [[ -d "$configured" ]] || continue
  parent="$(cd -- "$configured" && pwd -P)"; scope_roots=$((scope_roots+1))
  candidate="$parent/rum"
  if [[ ! -d "$candidate" || -L "$candidate" ]]; then emit candidate_unavailable "$candidate"; continue; fi
  root="$(cd -- "$candidate" && pwd -P)"
  [[ "$root" == "$candidate" ]] || { emit candidate_noncanonical "$candidate"; continue; }
  if [[ "$(rg "$root" rev-parse --show-toplevel 2>/dev/null || true)" != "$root" ]]; then emit candidate_not_repository "$root"; continue; fi
  origin="$(rg "$root" config --local --get remote.origin.url 2>/dev/null || true)"
  if ! origin_ok "$origin"; then emit candidate_wrong_origin "$root"; continue; fi
  common="$(rg "$root" rev-parse --path-format=absolute --git-common-dir)"
  common="$(cd -- "$common" && pwd -P)"
  emit verified_root "$root"
  # Capture command failure before parsing. Never silently accept a failed list.
  records="$(rg "$root" worktree list --porcelain)"
  while IFS= read -r line; do
    [[ "$line" == 'worktree '* ]] || continue
    path="${line#worktree }"
    attempted=$((attempted+1))
    [[ "$attempted" -le 32 ]] || { echo 'error=worktree-limit' >&2; exit 70; }
    [[ -d "$path" && ! -L "$path" ]] || { emit worktree_unavailable "$path"; continue; }
    canonical="$(cd -- "$path" && pwd -P)"
    [[ "$canonical" == "$path" ]] || { emit worktree_noncanonical "$path"; continue; }
    [[ -z "${seen[$canonical]:-}" ]] || continue
    seen[$canonical]=1
    [[ "$(rg "$path" rev-parse --show-toplevel)" == "$path" ]] || { emit worktree_unverified "$path"; continue; }
    wt_common="$(rg "$path" rev-parse --path-format=absolute --git-common-dir)"
    wt_common="$(cd -- "$wt_common" && pwd -P)"
    [[ "$wt_common" == "$common" ]] || { emit worktree_not_linked "$path"; continue; }
    origin="$(rg "$path" config --local --get remote.origin.url)"
    origin_ok "$origin" || { emit worktree_wrong_origin "$path"; continue; }
    head_before="$(rg "$path" rev-parse HEAD)"
    state_before="$(rg "$path" status --porcelain=v1 --untracked-files=all --ignore-submodules=all)"
    verified=$((verified+1)); prefix="worktree_${verified}"
    emit "${prefix}_path" "$path"
    emit "${prefix}_head" "$head_before"
    emit "${prefix}_branch" "$(rg "$path" symbolic-ref --quiet --short HEAD || echo detached)"
    emit "${prefix}_submodules" not_inspected
    entries=0; staged=0; unstaged=0; untracked=0
    while IFS= read -r entry; do
      [[ -n "$entry" ]] || continue
      entries=$((entries+1))
      if [[ "${entry:0:2}" == '??' ]]; then untracked=$((untracked+1)); else
        if [[ "${entry:0:1}" != ' ' ]]; then staged=$((staged+1)); fi
        if [[ "${entry:1:1}" != ' ' ]]; then unstaged=$((unstaged+1)); fi
      fi
      if [[ "$entries" -le 200 ]]; then emit "${prefix}_status_${entries}" "$entry"; fi
    done <<< "$state_before"
    emit "${prefix}_staged_entries" "$staged"
    emit "${prefix}_unstaged_entries" "$unstaged"
    emit "${prefix}_untracked_entries" "$untracked"
    emit "${prefix}_status_total" "$entries"
    if [[ "$entries" -gt 200 ]]; then emit "${prefix}_status_truncated" true; else emit "${prefix}_status_truncated" false; fi
    if rg "$path" cat-file -e "${pr_head}^{commit}" 2>/dev/null; then
      emit "${prefix}_commits_not_in_pr_total" "$(rg "$path" rev-list --count "${pr_head}..HEAD")"
      commits="$(rg "$path" log --format=%H --max-count=30 "${pr_head}..HEAD")"
      emit "${prefix}_commits_not_in_pr_first_30" "$commits"
    else emit "${prefix}_comparison" pr_head_object_unavailable; fi
    [[ "$(rg "$path" rev-parse HEAD)" == "$head_before" && "$(rg "$path" status --porcelain=v1 --untracked-files=all --ignore-submodules=all)" == "$state_before" ]] || { echo 'error=concurrent-workspace-change' >&2; exit 75; }
  done <<< "$records"
done
[[ "$scope_roots" -gt 0 ]] || { echo 'error=no-configured-root-available' >&2; exit 65; }
emit verified_worktrees "$verified"
emit inventory_returned true
emit workspace_ready false
