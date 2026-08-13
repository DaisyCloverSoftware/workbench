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
- Durable memory, checkpoints and routine/code writes reject probable secret material.
- Durable knowledge is stored in local protected application state, not in the public source tree.
- Project memory is scoped by stable project identity; cross-project/global memory must be selected explicitly.
- Context packs are bounded and contain only selected durable records rather than the complete knowledge database.
- Vault plaintext is not exposed through MCP or copied into durable memory.
- Autonomous worker prompts prohibit remote push, deploy, production changes, credential disclosure and destructive operations.
- Metered API routing is disabled by default.
- Scarce agentic usage is protected by default.

## Durable-memory boundary

The knowledge store is **not a second vault**. It is model-facing project context. Anything persisted there may later be supplied to another eligible model/worker when relevant to a task.

Keep credentials, private keys, access tokens and other raw secrets in the encrypted vault or external secret stores. Secret-like memory/routine writes are refused, and automatic task-outcome capture skips secret-looking output.

Project scope is the default for model-written memory and routines. Global scope is deliberately explicit because global records can be recalled while working on unrelated projects. Global memory should contain general engineering knowledge, reusable procedures and deliberately shareable patterns—not customer/project confidential detail.

A Git remote is used only as a project identity signal when available; the knowledge database remains local. Repositories without a usable remote receive a machine-local identity.

## Public repository boundary

This public repository contains **generic product code and documentation only**. Private dogfood/deployment data must not be committed to code, issues, pull requests, release notes, examples, public relay messages or screenshots.

Do not publish machine names, tailnet names, local usernames/home paths, private addresses, internal service URLs/topology, provider-account inventory or entitlement state, credentials/tokens/private keys, tunnel/runtime secrets, vault contents, private project/task content, durable private memory, or private logs.

Deployment-specific values belong in local protected configuration or a private authenticated repository. Public examples use placeholders only.

## Credential and relay handling

Private transports keep independent authorization boundaries. Local MCP credentials and remote tunnel/runtime credentials are stored in local protected files and should not be printed, committed, or placed in model context.

A public relay is suitable only for deliberately harmless synthetic dogfood messages and status-only output. Real workloads and any relay-backed durable-memory/control traffic require a private authenticated relay transport.

## Important limitations

Provider CLIs and external agent harnesses are separate trust boundaries. A sufficiently capable coding agent can execute commands inside its provider-defined sandbox/permission system. Workbench prompts reduce accidental overreach but do not replace operating-system isolation.

Secret detection is intentionally conservative and pattern-based. It reduces accidental credential retention but cannot prove that arbitrary prose is non-sensitive. Scope private/confidential knowledge appropriately and use a private transport for real workloads.

Configurable harness and notification commands are deliberately powerful and should be treated like shell configuration.

Do not point an early Workbench build directly at production infrastructure without an external permission layer.

## Reporting vulnerabilities

Do not include live credentials, private topology, durable private memory, or other users' private data in a public report. For sensitive disclosures, contact the repository owner privately before publishing details.
