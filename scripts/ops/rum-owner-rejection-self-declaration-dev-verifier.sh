#!/usr/bin/env bash
set -euo pipefail
umask 077
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SOURCE="$ROOT/scripts/ops/rum-self-declaration-dev-verifier.sh"
[[ -f "$SOURCE" && ! -L "$SOURCE" ]] || { echo "VERIFY BLOCKED: base self-declaration verifier unavailable" >&2; exit 78; }
work="$(mktemp -d)"; trap 'rm -rf "$work"' EXIT HUP INT TERM
COPY="$work/rum-self-declaration-owner-rejection.sh"; cp "$SOURCE" "$COPY"
python3 - "$COPY" <<'PY'
from pathlib import Path
import sys
p=Path(sys.argv[1]); text=p.read_text()

def replace_once(old,new,label):
    global text
    count=text.count(old)
    if count != 1: raise SystemExit(f'{label} marker count={count}')
    text=text.replace(old,new,1)

old='''    print("self_declared_profile_row_visible_ok")
    print("self_declared_verification_prompt_absent_ok")

    known=[message for message in console_errors if baseline in message and "script-src 'self'" in message]
'''
new='''    print("self_declared_profile_row_visible_ok")
    print("self_declared_verification_prompt_absent_ok")

    # Explicit owner acceptance: reload, then navigate away and back. The
    # relationship must remain established and no verification UI may appear.
    page.reload(wait_until="networkidle", timeout=60000)
    reloaded=page.locator(".my-identity-row").filter(has_text=gamertag).first
    reloaded.wait_for(state="visible", timeout=30000)
    reloaded.get_by_text("Mine · self-declared", exact=True).wait_for(state="visible", timeout=15000)
    if reloaded.get_by_role("button", name="Verify identity", exact=True).count() != 0:
        raise RuntimeError("Self-declared identity exposed Verify identity after reload.")
    print("self_declared_reload_persistence_ok")

    page.goto(f"{base}/rate", wait_until="networkidle", timeout=60000)
    page.goto(f"{base}/me", wait_until="networkidle", timeout=60000)
    returned=page.locator(".my-identity-row").filter(has_text=gamertag).first
    returned.wait_for(state="visible", timeout=30000)
    returned.get_by_text("Mine · self-declared", exact=True).wait_for(state="visible", timeout=15000)
    if returned.get_by_role("button", name="Verify identity", exact=True).count() != 0:
        raise RuntimeError("Self-declared identity exposed Verify identity after navigation.")
    returned_text=returned.inner_text().lower()
    for rejected in ("verification pending","not verified","unverified","verification: not started"):
        if rejected in returned_text: raise RuntimeError(f"Self-declared identity reverted after navigation: {rejected}")
    print("self_declared_navigation_persistence_ok")

    known=[message for message in console_errors if baseline in message and "script-src 'self'" in message]
'''
replace_once(old,new,'browser persistence')
old='''probe="$(kctl -n "$DEV_NAMESPACE" exec "$api_pod" -c php-fpm -- php -r "$PHP_BOOT" "\\$u=App\\\\Models\\\\User::where('email',base64_decode('${email_b64}'))->firstOrFail(); \\$e=App\\\\Models\\\\Entity::where('canonical_name',base64_decode('${gamertag_b64}'))->firstOrFail(); \\$c=App\\\\Models\\\\EntityClaim::where('entity_id',\\$e->id)->where('claimant_user_id',\\$u->id)->firstOrFail(); echo \\$c->status.' '.\\$c->verification_state.' '.App\\\\Models\\\\EntityClaimVerification::where('entity_claim_id',\\$c->id)->count();" 2>/dev/null | tail -n 1)"
read -r claim_status verification_state verification_count <<<"$probe"
[[ "$claim_status" == "approved" && "$verification_state" == "self_declared" && "$verification_count" == "0" ]] || {
  echo "VERIFY FAILED: self-declaration persistence mismatch" >&2
  exit 1
}
'''
new='''probe="$(kctl -n "$DEV_NAMESPACE" exec "$api_pod" -c php-fpm -- php -r "$PHP_BOOT" "\\$u=App\\\\Models\\\\User::where('email',base64_decode('${email_b64}'))->firstOrFail(); \\$e=App\\\\Models\\\\Entity::where('canonical_name',base64_decode('${gamertag_b64}'))->firstOrFail(); \\$c=App\\\\Models\\\\EntityClaim::where('entity_id',\\$e->id)->where('claimant_user_id',\\$u->id)->firstOrFail(); \\$l=App\\\\Models\\\\AccountEntityLink::where('entity_id',\\$e->id)->where('user_id',\\$u->id)->where('status','active')->firstOrFail(); \\$r=App\\\\Models\\\\EntityRelationship::where('source_entity_id',\\$e->id)->where('created_by_user_id',\\$u->id)->whereNull('valid_to')->firstOrFail(); echo \\$c->status.' '.\\$c->verification_state.' '.App\\\\Models\\\\EntityClaimVerification::where('entity_claim_id',\\$c->id)->count().' '.\\$l->verification_state.' '.\\$r->verification_state.' '.\\$e->verification_state;" 2>/dev/null | tail -n 1)"
read -r claim_status verification_state verification_count link_state relationship_state entity_state <<<"$probe"
[[ "$claim_status" == "approved" && "$verification_state" == "self_declared" && "$verification_count" == "0" && "$link_state" == "self_declared" && "$relationship_state" == "self_declared" && "$entity_state" == "self_claimed" ]] || {
  echo "VERIFY FAILED: self-declaration persisted-state mismatch: $probe" >&2
  exit 1
}
printf 'self_declared_persisted_states=claim:%s link:%s relationship:%s entity:%s verification_records:%s\\n' "$verification_state" "$link_state" "$relationship_state" "$entity_state" "$verification_count"
'''
replace_once(old,new,'persisted state probe')
p.write_text(text)
PY
chmod 0700 "$COPY"
bash "$COPY" "$@"
