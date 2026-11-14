# RAT PR170 — isolated DEV email-capture admission review

**Status: PROPOSED / NOT DEPLOYED.** This is infrastructure, not a RAT application change. Independent maintainer review/admission is required before invoking the mutating operation.

## Exact purpose

Provide a short-lived, operator-controlled mailbox for an existing **synthetic** `rat169qa*` test account, solely to finish PR #170's claimed-Thing DEV acceptance. Existing unclaimed editing and connection acceptance already passes. Do not change application CI/images, mail middleware, database verification flags, LIVE, or the owner's real data.

The approved synthetic account must have a reserved example-domain email address matching its username; operators must validate that mapping using canonical application state **without printing the email address or verification link**. The recipient pattern is restricted to `^rat169qa[a-z0-9]{6,24}@example[.]com$`, never a real address. The operator is authorised to receive mail sent to these reserved test identities **inside isolated DEV**; it is not evidence of ownership of an externally registered mailbox.

## Proposed bounded infrastructure

Reviewed operation: `scripts/ops/rat170-isolated-dev-mailcapture.sh`.

- Restricted to the currently deployed exact canonical DEV API and isolated RAT digests, verified ingress identities, and unchanged LIVE baseline.
- Requires a **reviewed immutable official Mailpit image digest**. A tag, latest, or unverified digest is refused.
- Creates exactly three owned DEV resources: one ephemeral Deployment, SMTP-only ClusterIP Service on TCP 1025, and a restrictive NetworkPolicy. No new Ingress, UI Service, host port, volume claim, or credential-bearing Secret.
- Runs a single Mailpit pod with memory-only `emptyDir` storage, a non-root, read-only root filesystem, dropped capabilities, no service-account token, and strict resource limits.
- Binds its HTTP/API listener to `127.0.0.1:8025` *inside the pod*; there is no publicly accessible UI/API. API access, if approved, is via the authenticated Workbench operator's bounded `kubectl` connection only, not a browser route.
- Captures only the allowlisted synthetic example-domain recipients, without SMTP relay/forwarding or external egress. Reverse DNS and external version checks are disabled. The NetworkPolicy permits SMTP ingress only from the canonical DEV API pod and denies capture-pod egress.
- Leaves canonical DEV's `MAIL_MAILER=log` configuration and all running application deployment specifications unchanged. Later **reviewed** notification operation may use Laravel's standard notification service with an invocation-local SMTP override so authentication notifications are sent to the controlled sink instead of application logs.
- `remove` mode deletes only the three owned capture resources after verifying the expected image and ownership. Deleting the pod destroys the in-memory messages. If any preflight or resource identity fails, the operation refuses to proceed.

## Required security review before execution

1. An independent authorised maintainer reviews the Kubernetes security, Mailpit image digest, NetworkPolicy, recipient restriction, mailbox custody, rollback/cleanup path and exact DEV-only scope. Approve the dedicated infrastructure PR through normal review; do not treat a Git branch head as security sign-off.
2. The maintainer confirms an existing approved synthetic `rat169qa*` user and exact `@example.com` reserved test alias may be received by the isolated operator mailbox. Real email, person, business and ownership assertions remain forbidden.
3. The reviewer confirms the subsequent notification/password-recovery verifier executes the normal Laravel signed-link and password-broker flows, keeps links/tokens/passwords in operator-only memory or short-lived mode-0600 files, and returns only boolean/provenance-safe results. It **must not** read application logs, create signatures, set `email_verified_at`, override a verification middleware, expose links through relay results, or persist credentials in GitHub.
4. Approve a concrete immutable Mailpit image digest, then run the guarded `install` operation from an admitted reviewed Workbench origin HEAD. Verify original Workbench outbox, actual pod imageID, NetworkPolicy and absence of public Service/Ingress.
5. Verify/recover **one existing** synthetic claimant through the standard signed-link route using the approved controlled mailbox; read back the account's verified status before using the existing guarded synthetic claimed-business fixture.
6. Run the deployed claimant/outsider correction browsers and read-only provenance/claim moderation checks. Repeat no previously passing RAT tests unless materially affected.
7. Invoke the guarded `remove` operation and independently confirm all three capture resources are gone, the recipient messages are gone, app images unchanged, and LIVE unchanged. Preserve only sanitised receipts.

## Non-goals and blocking gates

- Do not use the original `log` mail transport's token-bearing application output as mailbox access.
- Do not route real accounts through this sink, alter normal production email delivery, use actual DNS/business ownership, or manually set account/claim verification flags.
- Do not start this operation while this proposal is unreviewed; repository admission is necessary but not sufficient for security approval.
- No claim that actual mail capture, authenticated verification, claimed-Thing acceptance or OWNER_REVIEW_READY succeeded until the respective independent receipts exist.
- If review, immutable image selection or protected notification handling remains unavailable, leave the current isolated candidate intact and report the precise approval requirement.

Associated application source remains PR #170 at `e3bb818f1825d778f5084460057ba7ea8bfb422b`, exact-head CI 9/9 passing, with no application PR merge/deployment required.
