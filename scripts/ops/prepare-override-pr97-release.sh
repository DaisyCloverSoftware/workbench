#!/usr/bin/env bash
# One-shot release metadata update for the sealed Override PR97 handler.
set -euo pipefail
[[ $# -eq 0 ]] || { echo 'This operation accepts no arguments.' >&2; exit 2; }
readonly repository='https://github.com/DaisyCloverSoftware/workbench.git'
readonly branch='agent/override-pr97-sealed-proof-20260914'
readonly head="$(git rev-parse HEAD)"
[[ "$(git ls-remote "$repository" "refs/heads/$branch" | cut -f1)" == "$head" ]] || exit 2
[[ -z "$(git status --porcelain --untracked-files=no)" ]] || exit 2
grep -Fx 'const Version = "0.9.63"' internal/core/version.go >/dev/null
if command -v go >/dev/null 2>&1; then
  go_bin="$(command -v go)"
else
  go_bin="$(find "$HOME/.local/share/workbench/toolchains" -type f -path '*/go/bin/go' -perm -u+x 2>/dev/null | sort -V | tail -n 1 || true)"
fi
[[ -n "${go_bin:-}" && -x "$go_bin" ]] || exit 1
request="$(mktemp)"
trap 'rm -f "$request"' EXIT
cat > "$request" <<'JSON'
{"version":"0.9.64","date":"2026-09-14","notes":["Add one sealed Override PR97 native proof operation for reviewed source bde855d7511ef45dbc9ec8aff67ebb92df25a579, using the existing authenticated host bridge and committed-ops path.","Preserve full LFS materialisation, runtime staging, Editor compile, RAINLINE automation, Win64 packaging and packaged ZeroDay proof under the fixed Unreal Engine 5.8.1 project association. No arbitrary executable, command, script or filesystem target is accepted.","Advertise the fixed handler on the Windows client, reject unsupported or conflicting hosts before queuing, and retain per-job logs and hashed evidence in isolated Workbench storage. Source tests and cross-builds do not constitute Unreal or visual acceptance."]}
JSON
"$go_bin" run ./cmd/workbench-release-bump "$request"
"$go_bin" test ./cmd/workbench-release-bump
"$go_bin" test ./internal/core -run '^TestOverridePR97'
git diff --check
git rm -- scripts/ops/prepare-override-pr97-release.sh
git -c user.name='Workbench Engineering' -c user.email='workbench@users.noreply.github.com' commit --only -m 'Release: Workbench 0.9.64 for the sealed Override PR97 handler' -- .codex-plugin/plugin.json cmd/workbench/main_windows.go cmd/workbench-runner/main.go cmd/workbench-server/main.go cmd/workbench-relay/main.go internal/core/version.go internal/mcp/server.go CHANGELOG.md scripts/ops/prepare-override-pr97-release.sh
git push "$repository" "HEAD:refs/heads/$branch"
printf 'OVERRIDE_HANDLER_RELEASE_COMMIT=%s\n' "$(git rev-parse HEAD)"
