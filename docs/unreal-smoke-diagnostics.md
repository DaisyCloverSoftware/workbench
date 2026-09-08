# Unreal smoke diagnostic correction

This is a diagnostic correction within S0-006, not a new Unreal execution capability or a change to startup acceptance. The existing governance, security, outbound host bridge and review/deployment gates apply.

## Evidence contract

The fixed disposable-project smoke must distinguish observations from inferred causes. A Zen failure label requires a Zen-related failure in the same log record. Healthy Zen output plus an unrelated loader/profiler failure elsewhere, including on another stream, must not produce `class=zen`. A disabled optional shared cache is not a failure of the local cache. This explicitly rejects the old whole-capture substring conjunction.

A bounded streaming collector must retain only fixed categorical signals after each record. It reports observed Zen service readiness, local cache readiness, Zen errors, engine exit requests and video-memory warnings independently. Positive readiness and a later error may both be observed; neither may erase the other. These observations are not proof of the root cause or a successful job.

Timeout-stage reporting must distinguish historical keyword presence from evidence near the end of an output stream. `shader-work` requires an actual shader-progress phrase rather than the `LogShaderCompilers` category name alone. The returned diagnostic includes the recognised stage closest to the tail of either individual stream plus the bounded number of later records in that stream, and separately the bounded number of records after the closest observed shader-progress record. Stdout and stderr are asynchronous and MUST NOT be merged into an invented global record order. These record distances are diagnostic recency evidence only; they do not prove useful progress, a deadlock, or a causal subsystem failure.

The native 8 September observation established healthy local Zen, no recognised shader-progress record, and a latest recognised `asset-discovery` stage with later output before the five-minute timeout. The smoke therefore must no longer depend only on a startup console `Quit` reaching a late editor command-processing point. In addition to retaining the fixed `-ExecCmds=Quit` fallback, the fixed invocation must use Unreal's built-in `-TestExit` mechanism keyed to the privacy-safe engine-initialisation sentinel `Engine is initialized`. This does not shorten or bypass normal engine initialisation: it asks the process to exit once that existing startup milestone is actually emitted. If the process still times out after emitting the sentinel, diagnostics must report that independently rather than treating it as success.

No raw log line, path, hostname, address, command, credential, project data or caller-selected diagnostic text may be returned. Keep at most 8 KiB of an unfinished record per output stream. Discard an oversized record completely, report that coverage limitation, and resume at the next newline. Do not join records or streams or classify a retained prefix of an oversized record. Preserve categorical signals throughout the stream, including records in the middle omitted by the old 8 KiB head-and-rolling-tail capture. Empty or unrecognised output remains inconclusive.

## Unchanged authority and acceptance

Keep the same validated executable selection, Workbench-owned disposable project, cleanup, five-minute timeout, typed job submission/claim/completion protocol and Windows-side allowlist. The fixed argv may add only the engine-initialisation `-TestExit` sentinel described above; callers still cannot supply any project, executable, script, commandlet, console command, exit pattern, argument, environment variable or URL. Do not add an arbitrary log reader, generic Windows command, inbound listener, project automation, cache repair, security change, longer timeout or new remote parameter.

Successful startup still requires the real smoke process to exit successfully and its correlated host-job result to be retrieved. Error classification, engine-initialisation observation and other diagnostic flags never override the process outcome. Manual editor success, installed metadata, a green build or corrected diagnostics alone are not automated Unreal acceptance.

## Regression and target verification

Cover healthy Zen plus unrelated failures on the same and different streams, metadata substrings, disabled shared cache, genuine Zen errors, normal failure precedence, late and omitted-middle records beyond 8 KiB total output, write-chunk boundaries, CRLF, unterminated final records, oversized records, privacy, stage-recency distances, shader-category false positives, engine-initialisation sentinel observation, fixed `-TestExit` argv confinement and successful-output compatibility. Test the actual Windows wiring as well as the portable collector. Run the full ordinary test/build gates on the exact candidate.

Delivery is a PR/preview Windows build until separately approved through the existing release/install process. Verify the exact executable on the affected Windows host before using new diagnostic results. Do not launch another Unreal process alongside an owner-observed memory-constrained editor session or interrupt the owner's editor. A source-only fix does not close S0-006.
