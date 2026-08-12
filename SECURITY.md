# Security

Workbench is powerful software. It exists to give AI systems controlled access to developer machines and agent harnesses, so the default posture must be restrictive.

## Current protections

- Private MCP servers bind loopback only.
- MCP uses bearer authentication.
- Unexpected HTTP `Origin` values are rejected.
- `run_safe_command` uses an allowlist and rejects shell composition, push/deploy/network/destructive patterns.
- `apply_patch` performs `git apply --check` before writing and rejects secret-looking content.
- Notes sent through MCP are rejected if they look like credentials.
- Repository read/search tools filter credential-like, binary, generated, oversized and probable-secret files.
- Vault plaintext is not exposed through MCP.
- Autonomous worker prompts prohibit remote push, deploy, production changes, credential disclosure and destructive operations.
- Metered API routing is disabled by default.
- Scarce agentic usage is protected by default.

## Public repository boundary

This public repository contains **generic product code and documentation only**. Private dogfood/deployment data must not be committed to code, issues, pull requests, release notes, examples, public relay messages or screenshots.

Do not publish machine names, tailnet names, local usernames/home paths, private addresses, internal service URLs/topology, provider-account inventory or entitlement state, credentials/tokens/private keys, tunnel/runtime secrets, vault contents, or private project/task content and logs.

Deployment-specific values belong in local protected configuration or a private authenticated repository. Public examples use placeholders only.

## Credential handling

Private transports keep independent authorization boundaries. Local MCP credentials and remote tunnel/runtime credentials are stored in local protected files and should not be printed, committed, or placed in model context.

A public relay is suitable only for deliberately harmless synthetic dogfood messages. Real workloads require a private authenticated relay transport.

## Important limitations

Provider CLIs and external agent harnesses are separate trust boundaries. A sufficiently capable coding agent can execute commands inside its provider-defined sandbox/permission system. Workbench prompts reduce accidental overreach but do not replace operating-system isolation.

Configurable harness and notification commands are deliberately powerful and should be treated like shell configuration.

Do not point an early Workbench build directly at production infrastructure without an external permission layer.

## Reporting vulnerabilities

Do not include live credentials, private topology or other users' private data in a public report. For sensitive disclosures, contact the repository owner privately before publishing details.
