#!/usr/bin/env bash
# One-shot exact-source integration; only the dedicated review branch is pushed.
set -euo pipefail
[[ $# -eq 0 ]] || { echo 'This operation accepts no arguments.' >&2; exit 2; }
readonly repository='https://github.com/DaisyCloverSoftware/workbench.git'
readonly branch='agent/override-pr97-sealed-proof-20260914'
readonly head="$(git rev-parse HEAD)"
[[ "$(git ls-remote "$repository" "refs/heads/$branch" | cut -f1)" == "$head" ]] || { echo 'Branch changed; refusing to integrate.' >&2; exit 2; }
[[ "$(git rev-parse HEAD:internal/core/host_bridge_agent_windows.go)" == '6c20358aaadfca9cd45b0f2c786573abd7fe27e2' ]] || exit 2
[[ "$(git rev-parse HEAD:internal/core/host_bridge.go)" == '4d610ef84ecbbf23241d2436efc7edac6cdfd8fb' ]] || exit 2
[[ -z "$(git status --porcelain --untracked-files=no)" ]] || exit 2
python3 - <<'PY'
from pathlib import Path
p=Path('internal/core/host_bridge_agent_windows.go')
s=p.read_text()
old='\tcase HostBridgeToolWorkbench:\n\t\tswitch job.Spec.Operation {\n'
new=old+'''\t\tcase HostBridgeOperationOverridePR97Proof:
            proofCtx, cancel := context.WithTimeout(ctx, overridePR97Timeout)
            defer cancel()
            output, err := runOverridePR97Proof(proofCtx, job.ID)
            if err != nil {
                return HostJobResult{Output: output, ExitCode: 1}, err.Error()
            }
            return HostJobResult{Output: output, ExitCode: 0}, ""
'''
assert s.count(old)==1
s=s.replace(old,new,1)
old='HostCapability{Installed: true, Version: identity}'
assert s.count(old)==1
s=s.replace(old,'HostCapability{Installed: true, Version: identity + " " + overridePR97Capability}',1)
p.write_text(s)
p=Path('internal/core/host_bridge.go');s=p.read_text()
old='\t\t\tjob.ClaimExpiresAt = now.Add(10 * time.Minute).Format(time.RFC3339Nano)\n'
assert s.count(old)==1
s=s.replace(old,old+'''            if job.Spec.Tool == HostBridgeToolWorkbench && job.Spec.Operation == HostBridgeOperationOverridePR97Proof {
                job.ClaimExpiresAt = now.Add(overridePR97Timeout + 5*time.Minute).Format(time.RFC3339Nano)
            }
''',1)
p.write_text(s)
PY
if command -v go >/dev/null 2>&1; then
  go_bin="$(command -v go)"
else
  go_bin="$(find "$HOME/.local/share/workbench/toolchains" -type f -path '*/go/bin/go' -perm -u+x 2>/dev/null | sort -V | tail -n 1 || true)"
fi
[[ -n "${go_bin:-}" && -x "$go_bin" ]] || { echo 'Workbench Go toolchain unavailable' >&2; exit 1; }
"$(dirname "$go_bin")/gofmt" -w internal/core/host_bridge.go internal/core/host_bridge_agent_windows.go internal/core/host_bridge_override_pr97*.go cmd/workbench-override-pr97-submit/main.go
"$go_bin" test ./...
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 "$go_bin" build -o "$tmp/Workbench.exe" ./cmd/workbench
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 "$go_bin" test -c -o "$tmp/core-tests.exe" ./internal/core
sha256sum "$tmp/Workbench.exe"
git diff --check
git rm -- scripts/ops/integrate-override-pr97-proof.sh
git -c user.name='Workbench Engineering' -c user.email='workbench@users.noreply.github.com' commit --only -m 'Override: wire and validate the single PR97 Windows proof operation' -- internal/core/host_bridge.go internal/core/host_bridge_agent_windows.go internal/core/host_bridge_override_pr97.go internal/core/host_bridge_override_pr97_windows.go internal/core/host_bridge_override_pr97_test.go cmd/workbench-override-pr97-submit/main.go scripts/ops/integrate-override-pr97-proof.sh
git push "$repository" "HEAD:refs/heads/$branch"
printf 'SEALED_PR97_OPERATION_COMMIT=%s\n' "$(git rev-parse HEAD)"
