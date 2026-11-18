#!/usr/bin/env bash
set -Eeuo pipefail
# Source-only syntax and release guards for PR177 API and RAT, deliberately
# separate from the unauthorised/unverified browser acceptance script.
[[ $# -eq 0 ]] || exit 64
api="scripts/ops/deploy-rat177-thing-type-api-isolated-20261118.sh"
rat="scripts/ops/deploy-rat177-type-editor-frontend-isolated-20261118.sh"
for script in "$api" "$rat";do
 [[ -f "$script" && ! -L "$script" ]] || exit 66
 bash -n "$script"
done
python3 - "$api" "$rat" <<'PY'
import pathlib,sys,re
a=pathlib.Path(sys.argv[1]).read_text()
r=pathlib.Path(sys.argv[2]).read_text()
head="628296e5c57665814799bfedf1ab3ef8f6f163df"
api_old="ghcr.io/daisycloversoftware/rum-api@sha256:76731a87574a11547938a398dc15ecf87ced56d63b6622cd50bab5836dfeee71"
rat_old="ghcr.io/daisycloversoftware/rum-rate-anything@sha256:448807a70db93b91bb0fd714b854253c93157ad4d89d96255a385b94bf1258bc"
assert head in a and head in r
assert api_old in a and rat_old in a
assert rat_old in r and "protected_snapshot()" in r
assert "rollback()" in a and "rollback()" in r
for path,source in [(sys.argv[1],a),(sys.argv[2],r)]:
  for chunk in source.split("<<'PY'\n")[1:]:
    compile(chunk.split("\nPY\n",1)[0],path,'exec')
assert "__RAT177_API_DIGEST_PENDING__" not in a
assert "__RAT177_API_DIGEST_PENDING__" not in r
assert "__RAT177_RAT_DIGEST_PENDING__" not in r
print("RAT177_EXACT_GUARDED_API_AND_FRONTEND_RELEASE_PIN_SYNTAX_PASS=true")
PY