# Private MCP failure diagnostics

This is a diagnostic correction to the existing private control transport, not an execution capability or policy expansion.

## Contract

A failed private control request keeps `status: failed`, its existing envelope `id`, and a concise top-level `error`. When the failure crossed MCP, `result.diagnostic` also preserves a versioned, sanitized diagnostic with the same request identifier. A command error does not become a completed operation merely because a diagnostic is available.

The relay previously discarded `structuredContent.error` and `structuredContent.output` when MCP returned `isError`. Its private outbox then discarded the returned result on errors. Both loss points are corrected. Already-published generic failures are not rewritten: new instrumentation cannot recover output which was never retained.

The diagnostic identifies the observed layer: relay preflight, request validation, MCP preflight, transport, HTTP response, response reading/size/decoding/shape, JSON-RPC, or MCP tool failure. It forwards only documented error text and combined output from failed tools, not arbitrary structured data, RPC error data, HTTP response bodies, request arguments or environment dumps.

`execution_started`, `exit_code` and `timed_out` are nullable evidence fields. Null means **unknown**, not false or zero. The current safe-command MCP reports an error string and combined output, not structured process metadata. Therefore a tool error retains null process fields even when its human-readable error mentions an exit status. The relay does not parse prose into authoritative process facts. Rejection known to occur before invocation records `execution_started: false`. A transport timeout records `timed_out: true` but leaves execution and exit status unknown. A received response, HTTP error or timeout must never cause an automatic replay.

Successful structured results retain their existing shape and existing secret/size guards. A missing or unusable response cannot be reported as a successfully observed command result.

## Bounds and privacy

Failure messages are capped at 4 KiB and combined output at 16 KiB, with explicit independent truncation flags. The existing 1 MiB response bound is enforced with an explicit response-limit diagnostic. The existing 20-second MCP HTTP deadline is unchanged.

Redaction happens before truncation. Sensitive lines, authorization/cookie values, probable credentials, environment-assignment lines, credential-bearing HTTP URLs, private-key blocks and known sensitive relay-environment values are withheld. Sensitive environment values are used only for masking; no environment is serialized. The local MCP authentication value is also masked if reflected in an error. UTF-8 is preserved at the output boundary. Redaction is reported independently from truncation. Existing whole-result secret and private-outbox size checks remain enabled as a further boundary.

These diagnostics are bounded output from the requested operation, not arbitrary log/filesystem inspection. They do not authorize reading unrelated files or printing an environment. Public source/tests use only synthetic fixtures; private runtime evidence stays in the private transport.

## Verification and rollout

Focused tests cover validation/policy/project/launch/nonzero-error fixtures, preserved successful output, JSON-RPC validation, transport timeout/cancellation, unusable and oversized responses, masking, truncation and private-outbox correlation. Fixtures deliberately verify that missing process metadata remains unknown. No prohibited operation is executed by these tests.

Normal source review, required CI, release and checksum-verified runtime update gates still apply. A source PR or passing fixture test is not evidence that a deployed relay runs the correction. Verify the live relay revision and one permitted project operation through the normal `run_safe_command` interface after rollout, preserving the project changes before and after. An application error with useful retained diagnostics is an acceptable diagnostic verification result; application correctness is a separate task.

No command allowlist, project-root authorization, deployment guard, model/provider authority or product approval rule changes as part of this correction.
