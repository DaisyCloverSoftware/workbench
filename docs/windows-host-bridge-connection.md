# Windows host-bridge connection correction

## Scope and acceptance

The desktop-owned outbound Windows bridge must use the current **saved** Workbench Runner SSH target without requiring a desktop restart. This is a connection-lifecycle correction, not a new Windows execution surface.

A desktop started with an empty or invalid target must remain able to connect after a valid target is saved. Clearing the saved target suspends new polls. A subsequent valid target resumes them. Unsaved settings are not authority. A failed persistence attempt must not change the bridge target even if the engine already changed its in-memory preferences. One synchronized target holder publishes only after the actual save succeeds; concurrent saves are serialized without blocking target reads during persistence. Each poll still validates the target using the existing SSH-target validator.

Read the saved target between complete poll / execute / report cycles. Never launch concurrent bridge cycles, cancel an in-flight job merely because routing settings change, or report a claimed job to a different server. Desktop shutdown retains the existing cancellation behaviour. Existing network timeouts, polling cadence, local operation allowlists and fixed `host-json` transport remain unchanged.

A source-level regression existed because startup captured the target once, and saving settings or rescanning providers did not start or redirect the bridge. That defect is reproducible from the source contract; it is not proof of the cause of any particular disconnected workstation.

## Verification boundaries

Portable loop tests cover initially missing/invalid settings, target changes, clearing and restoring a setting, non-overlapping cycles, in-flight target pinning, cancellation and invalid callback wiring. The Windows startup and persistence contracts verify that startup supplies the committed-target holder and routing saves publish only after persistence succeeds. Regression tests cover failed changes, failed clearing, concurrent save ordering and readers during a blocked save. Existing SSH validation and host-operation tests remain required.

Cross-compilation establishes Windows build compatibility only. Native acceptance still requires the candidate desktop running, a real heartbeat, a bounded successful host operation, and evidence that changing saved routing affects subsequent polls without disrupting an active cycle. Tests, network presence and a remembered version string must not be reported as restored connectivity.

This change does not start a closed application, provision credentials, create an inbound listener, register a GitHub Actions runner, extend Unreal operations or establish any project build/visual proof. It does not authorize an autonomous provider, alter publication policy or permit a forced desktop restart.

Run `bash scripts/ops/verify-host-bridge-reconnect.sh` in an isolated checkout for focused connection tests, relay/MCP tests and a Windows cross-build. The full `go test ./...` gate remains mandatory in ordinary CI: governance fixtures intentionally refuse local test origins under `WORKBENCH_OPERATION_SCRIPT=1`. Do not unset that guard or weaken origin checks to make an operations-run test suite green. This script does not deploy or update the installed desktop.
