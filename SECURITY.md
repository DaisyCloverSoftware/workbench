# Security

Workbench is powerful software. It exists to give AI systems controlled access to developer machines and agent harnesses, so the default posture must be restrictive.

Security decisions that materially affect product behaviour are also recorded in `docs/DECISIONS.md`.

## Current protections

- Private MCP servers bind loopback only.
- MCP uses bearer authentication.
- Unexpected HTTP `Origin` values are rejected.
- `run_safe_command` uses an allowlist and rejects shell composition, push/deploy/network/destructive patterns outside explicitly authorised paths.
- `apply_patch` performs `git apply --check` before writing and rejects secret-looking content.
- Notes sent through MCP are rejected if they look like credentials.
- Repository read/search tools filter credential-like, binary, generated, oversized and probable-secret files.
- Vault plaintext is not exposed through MCP.
- Autonomous worker prompts prohibit remote push, deploy, production changes, credential disclosure and destructive operations unless a separate explicit trusted operation authorises the action.
- Metered API routing is disabled by default.
- Scarce agentic usage is protected by default.
- Public/privacy-minimal inventory prefers opaque project/runner references over host filesystem paths.

## ChatGPT and autonomous-worker boundary

ChatGPT is the primary reasoning/coding path. Direct ChatGPT development operations MUST NOT silently instantiate or route through OpenClaw or another autonomous coding harness.

Autonomous delegation is a separate explicit capability. OpenClaw is optional operator/autonomous capacity, not a required trust hop for ordinary bounded operations.

The private-relay continuation path is also distinct: it can carry an explicitly authenticated durable development continuation across a dependency wait, but that does not reopen implicit ChatGPT → OpenClaw delegation.

## Authenticated private continuation

Private continuation authority is HMAC-bound to the relay correlation ID, project and original continuation body using the local MCP credential as key material. The credential itself is not persisted into task intent.

When a Workbench-owned external dependency becomes terminal, Workbench may append its exact dependency-update suffix after the proof. The validator accepts that Workbench-owned suffix shape while arbitrary appended content remains fail-closed.

Do not weaken this boundary to make transport/debugging more convenient.

## Windows host boundary

Windows host access is **outbound and typed/allowlisted**.

- The Windows host initiates/maintains its outbound bridge; do not add an inbound listener as a convenience shortcut.
- Remote operations are explicit typed actions (for example bounded Blender/Unreal operations), not a generic Windows command shell.
- New Windows capabilities require explicit argument/result bounds, privacy-safe errors and tests for rejection of unsupported/unsafe input.
- Generic shell authority MUST NOT be introduced simply to avoid adding a typed operation.

## Control-plane boundary

Long-running execution and bounded control/status operations should be logically separate even when they share transport. A long operation must not force unrelated health/status/cancellation reads to wait behind it.

Where durable execution is asynchronous, control operations should refer to bounded job identifiers/status rather than keeping a privileged synchronous session open.

## Public repository boundary

This public repository contains **generic product code and documentation only**. Private dogfood/deployment data must not be committed to code, issues, pull requests, release notes, examples, public relay messages or screenshots.

Do not publish machine names, tailnet names, local usernames/home paths, private addresses, internal service URLs/topology, provider-account inventory or entitlement state, credentials/tokens/private keys, tunnel/runtime secrets, vault contents, or private project/task content and logs.

Deployment-specific values belong in local protected configuration or a private authenticated repository. Public examples use placeholders only.

`PUBLIC_SOURCE_POLICY.md` and `docs/PRIVACY_GUARD.md` are normative for public-source hygiene.

## Credential handling

Private transports keep independent authorization boundaries. Local MCP credentials and remote tunnel/runtime credentials are stored in local protected files and should not be printed, committed, copied into public governance records or placed into ordinary model-visible task content.

Private relay payloads must not contain raw credentials. Authentication material stays in protected local configuration; relay messages carry only the bounded authority/proof material the protocol requires.

A public relay is suitable only for deliberately harmless synthetic dogfood messages. Real workloads require a private authenticated relay transport.

## Operations and result privacy

Machine/repository operations should return the minimum bounded result needed to make the next decision. Prefer categorical/privacy-safe diagnostics over raw logs when raw output could reveal paths, private project content, machine identity or credentials.

Live operational evidence used in public governance documentation must be generalised. The fact that a host/service was verified may be recorded without publishing its private identifier/address/topology.

## Important limitations

Provider CLIs and external agent harnesses are separate trust boundaries. A sufficiently capable coding agent can execute commands inside its provider-defined sandbox/permission system. Workbench prompts reduce accidental overreach but do not replace operating-system isolation.

Configurable harness and notification commands are deliberately powerful and should be treated like shell configuration.

Typed/allowlisted access reduces authority but does not prove the target application itself is safe or correct. Blender/Unreal smoke success, for example, is an acceptance question separate from the bridge security model.

Do not point an early Workbench build directly at production infrastructure without an external permission layer appropriate to that environment.

## Reporting vulnerabilities

Do not include live credentials, private topology or other users' private data in a public report. For sensitive disclosures, contact the repository owner privately before publishing details.
