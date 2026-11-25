# RUM / RAT: DEV data freedom and LIVE data protection

Status: OWNER APPROVED — MANDATORY, 25 November 2026.

## DEV

DEV is owner-controlled development, experimentation and testing data. An explicit owner instruction authorises permanent deletion of the identified application data and its exclusively dependent data: entities, identities, relationships/history, accounts, ratings/events/reasons/votes/reviews, images/media, historical/audit records and projections. Production historical-preservation policy does not override this instruction. Do not substitute hiding, disqualification, withdrawal, archiving, soft deletion or Sin Bin placement. No repeated owner approval is required for an already explicit request.

## LIVE

LIVE retains canonical evidence protection, historical integrity, auditability and production privacy/security/retention/deletion governance. DEV flexibility must not weaken LIVE schema, triggers, application controls or operating policy. DEV acceptance is not permission to merge or deploy to LIVE.

## Mandatory tooling boundary

Owner-directed deletion is an operational capability, not a product HTTP endpoint and not a per-cleanup application migration. Reusable, reviewed DEV-only maintenance commands must have no selectable LIVE/namespace/database/host target. Establish the authorised isolated namespace, workload and database endpoint identities before any mutation; validate the actual PostgreSQL database/role/server on the same connection used to delete. Fail closed if identity or exact target scope cannot be established.

Use exact canonical IDs, read-only dependency plans and deterministic scope fingerprints. Handle exclusively dependent rows and affected projections/caches transactionally. Preserve unrelated data. Foreign keys stay enabled. Where current evidence guards require a privileged deletion scope, use the established exact-ID transaction-private mechanism, restore and hash-verify all original function definitions before commit, and roll back on any failure. Never leave an application-wide deletion bypass or permanently disabled guard behind.

An external operational receipt may record IDs, counts, hashes and environment identity without retaining the deleted application content. Verify physical absence in the database, not merely UI disappearance. Report separately from application releases.

Current bounded adapter: `scripts/ops/rum-dev-delete-rating.sh` plans/applies/verifies exact Rating Event deletions with dependency inventory. Unknown/shared dependency structures require the relevant adapter to be extended safely, not a soft-delete substitute or a claim that DEV data is historically undeletable. This rule applies to future entity/media/account/relationship adapters as well.

Current explicit deletion scope: Rating Event `01m7x799g070af1g4je8hh468v`, entity `01m7x6ez57zpsgbv486y9t0y9h`. All other rating IDs, entities and relationships are protected. This policy is not blanket permission to delete unspecified data.
