#!/usr/bin/env bash
set -euo pipefail
umask 077

REPOSITORY="DaisyCloverSoftware/rum"
BRANCH="sprint-0-rum-owner-rating-flow-20260823"
EXPECTED_HEAD="ffe04a6bec937e3d6aa5e867f8d994531c689080"

for command in gh jq python3 base64 mktemp; do
  command -v "$command" >/dev/null 2>&1 || { echo "required command unavailable: $command" >&2; exit 2; }
done
TOKEN="${GH_TOKEN:-}"
if [[ -z "$TOKEN" ]]; then TOKEN="$(gh auth token 2>/dev/null || true)"; fi
[[ -n "$TOKEN" ]] || { echo "no GitHub token available" >&2; exit 2; }

head="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/${BRANCH}" --jq '.object.sha')"
[[ "$head" == "$EXPECTED_HEAD" ]] || { echo "PATCH BLOCKED: expected=${EXPECTED_HEAD} actual=${head}" >&2; exit 78; }

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT HUP INT TERM

patch_file() {
  local path="$1" transform="$2" message="$3"
  local meta="$tmp/meta.json" file="$tmp/file.txt" sha encoded
  GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/contents/${path}?ref=${BRANCH}" >"$meta"
  sha="$(jq -r '.sha' "$meta")"
  jq -r '.content' "$meta" | tr -d '\n' | base64 -d >"$file"
  python3 - "$file" "$transform" <<'PY'
from pathlib import Path
import sys
p=Path(sys.argv[1]); mode=sys.argv[2]; s=p.read_text()

def once(old,new,label):
    global s
    n=s.count(old)
    if n != 1:
        raise SystemExit(f'{label}: expected 1 match, found {n}')
    s=s.replace(old,new)

if mode == 'persona':
    once('test_member_can_submit_an_idempotent_claim_but_cannot_self_verify_it',
         'test_self_added_persona_claim_is_idempotently_self_declared_without_verification', 'persona name')
    once("        ])->assertCreated()\n            ->assertJsonPath('data.status', 'pending')\n            ->assertJsonPath('data.verificationLevel', 0);",
         "        ])->assertCreated()\n            ->assertJsonPath('data.status', 'approved')\n            ->assertJsonPath('data.verificationLevel', 1)\n            ->assertJsonPath('data.verificationState', 'self_declared')\n            ->assertJsonPath('meta.selfDeclared', true)\n            ->assertJsonPath('meta.verificationPromptRequired', false);", 'persona response')
    once("        $second = $this->postJson(\"/api/v1/entities/{$entityId}/claims\", [\n            'claimType' => 'controls_identity',\n        ])->assertOk();",
         "        $second = $this->postJson(\"/api/v1/entities/{$entityId}/claims\", [\n            'claimType' => 'controls_identity',\n        ])->assertOk()\n            ->assertJsonPath('data.status', 'approved')\n            ->assertJsonPath('data.verificationState', 'self_declared');", 'persona idempotent response')
    once("        $this->assertDatabaseCount('entity_managers', 0);\n        $this->assertDatabaseCount('account_entity_links', 0);",
         "        $this->assertDatabaseHas('entity_managers', [\n            'entity_id' => $entityId,\n            'user_id' => $user->id,\n            'role' => 'controller',\n            'status' => 'active',\n        ]);\n        $this->assertDatabaseHas('account_entity_links', [\n            'entity_id' => $entityId,\n            'user_id' => $user->id,\n            'link_type' => 'controls_identity',\n            'status' => 'active',\n            'verification_state' => 'claimed',\n            'source' => 'claim',\n        ]);\n        $this->assertDatabaseHas('entities', [\n            'id' => $entityId,\n            'verification_state' => 'self_claimed',\n        ]);", 'persona db')
elif mode == 'person':
    once('test_public_person_can_submit_this_is_me_claim_but_cannot_claim_controls_identity',
         'test_self_added_public_person_is_self_declared_but_cannot_claim_controls_identity', 'person name')
    once("        ])->assertCreated()\n            ->assertJsonPath('data.status', 'pending')\n            ->assertJsonPath('data.verificationLevel', 0);",
         "        ])->assertCreated()\n            ->assertJsonPath('data.status', 'approved')\n            ->assertJsonPath('data.verificationLevel', 1)\n            ->assertJsonPath('data.verificationState', 'self_declared')\n            ->assertJsonPath('meta.selfDeclared', true)\n            ->assertJsonPath('meta.verificationPromptRequired', false);", 'person response')
    once("        $this->assertDatabaseCount('entity_claims', 1);\n        $this->assertDatabaseCount('entity_managers', 0);",
         "        $this->assertDatabaseCount('entity_claims', 1);\n        $this->assertDatabaseHas('entity_managers', [\n            'entity_id' => $personId,\n            'user_id' => $user->id,\n            'role' => 'subject',\n            'status' => 'active',\n        ]);\n        $this->assertDatabaseHas('account_entity_links', [\n            'entity_id' => $personId,\n            'user_id' => $user->id,\n            'link_type' => 'person',\n            'status' => 'active',\n            'verification_state' => 'claimed',\n            'source' => 'claim',\n        ]);\n        $this->assertDatabaseHas('entities', [\n            'id' => $personId,\n            'verification_state' => 'self_claimed',\n        ]);", 'person db')
elif mode == 'self-rating':
    once("            ->assertJsonPath('data.verificationState', 'not_started');",
         "            ->assertJsonPath('data.verificationState', 'self_declared')\n            ->assertJsonPath('meta.selfDeclared', true)\n            ->assertJsonPath('meta.verificationPromptRequired', false);", 'self-rating response')
else:
    raise SystemExit('unknown mode')
p.write_text(s)
PY
  encoded="$(base64 -w0 <"$file")"
  GH_TOKEN="$TOKEN" gh api --method PUT "repos/${REPOSITORY}/contents/${path}" \
    -f message="$message" -f content="$encoded" -f sha="$sha" -f branch="$BRANCH" >/dev/null
  printf 'UPDATED=%s\n' "$path"
}

patch_file "apps/api/tests/Feature/Entity/EntityBridgeApiTest.php" persona "test: prove self-added persona is self-declared"
patch_file "apps/api/tests/Feature/Entity/PublicPersonEntityBridgeTest.php" person "test: prove self-added person is self-declared"
patch_file "apps/api/tests/Feature/Entity/RateUrMatePublicIdentitySelfRatingTest.php" self-rating "test: expect self-declared owned gamer identity"

final="$(GH_TOKEN="$TOKEN" gh api "repos/${REPOSITORY}/git/ref/heads/${BRANCH}" --jq '.object.sha')"
printf 'RUM153_SELF_DECLARATION_TEST_FIX_HEAD=%s\n' "$final"
