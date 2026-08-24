#!/usr/bin/env bash
set -euo pipefail
umask 077

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <exact-rum-candidate-sha>" >&2
  exit 64
fi

CANDIDATE_SHA="$1"
[[ "$CANDIDATE_SHA" =~ ^[0-9a-f]{40}$ ]] || {
  echo "full lowercase candidate SHA required" >&2
  exit 64
}

for command in bash python3 mktemp; do
  command -v "$command" >/dev/null 2>&1 || {
    echo "VERIFY BLOCKED: required command unavailable: $command" >&2
    exit 2
  }
done

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BASE="$ROOT/scripts/ops/rum-live-derived-rating-dev-verifier.sh"
[[ -f "$BASE" && ! -L "$BASE" ]] || {
  echo "VERIFY BLOCKED: base rating verifier is missing or not a regular file" >&2
  exit 78
}

tmp="$(mktemp)"
cleanup() { rm -f "$tmp"; }
trap cleanup EXIT HUP INT TERM

python3 - "$BASE" "$tmp" <<'PY'
from pathlib import Path
import sys

src = Path(sys.argv[1]).read_text()

init_old = 'console_errors=[]; page_errors=[]; request_failures=[]; api_failures=[]; identity_used=None'
init_new = 'console_errors=[]; page_errors=[]; request_failures=[]; benign_request_aborts=[]; api_failures=[]; identity_used=None'
event_old = 'page.on("console", lambda msg: console_errors.append(msg.text) if msg.type=="error" else None); page.on("pageerror", lambda err: page_errors.append(str(err))); page.on("requestfailed", lambda req: request_failures.append(f"{req.method} {req.url}")); page.on("response", lambda res: api_failures.append(f"{res.status} {res.url}") if res.status>=400 and "/api/" in res.url else None)'
event_new = 'page.on("console", lambda msg: console_errors.append(msg.text) if msg.type=="error" else None); page.on("pageerror", lambda err: page_errors.append(str(err))); page.on("requestfailed", lambda req: benign_request_aborts.append(f"{req.method} {req.url}") if req.failure == "net::ERR_ABORTED" else request_failures.append(f"{req.failure or \'unknown\'} {req.method} {req.url}")); page.on("response", lambda res: api_failures.append(f"{res.status} {res.url}") if res.status>=400 and "/api/" in res.url else None)'
print_old = 'print(f"known_live_baseline_csp_console_errors={len(known)}")'
print_new = 'print(f"known_live_baseline_csp_console_errors={len(known)}"); print(f"benign_navigation_request_aborts={len(benign_request_aborts)}")'

for needle, expected in ((init_old, 1), (event_old, 1), (print_old, 1)):
    actual = src.count(needle)
    if actual != expected:
        raise SystemExit(f"VERIFY BLOCKED: expected {expected} occurrence of compatibility needle, found {actual}")

out = src.replace(init_old, init_new).replace(event_old, event_new).replace(print_old, print_new)
Path(sys.argv[2]).write_text(out)
PY

chmod 700 "$tmp"

grep -Fq 'req.failure == "net::ERR_ABORTED"' "$tmp" || {
  echo "VERIFY BLOCKED: benign navigation-abort classification was not installed" >&2
  exit 78
}
grep -Fq 'if api_failures: raise RuntimeError' "$tmp" || {
  echo "VERIFY BLOCKED: strict API failure gate is missing" >&2
  exit 78
}
grep -Fq 'if request_failures: raise RuntimeError' "$tmp" || {
  echo "VERIFY BLOCKED: strict non-aborted network failure gate is missing" >&2
  exit 78
}
grep -Fq 'if page_errors: raise RuntimeError' "$tmp" || {
  echo "VERIFY BLOCKED: strict page-error gate is missing" >&2
  exit 78
}
grep -Fq 'if unexpected: raise RuntimeError' "$tmp" || {
  echo "VERIFY BLOCKED: strict unexpected-console gate is missing" >&2
  exit 78
}

printf 'RUM_RATING_NAVIGATION_ABORT_COMPAT=ERR_ABORTED_ONLY\n'
printf 'RUM_RATING_REAL_NETWORK_FAILURES=STRICT\n'
printf 'RUM_RATING_API_FAILURES=STRICT\n'

bash "$tmp" "$CANDIDATE_SHA"
