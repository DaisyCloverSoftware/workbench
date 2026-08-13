# Roadmap

## v0.3 — native dogfood foundation

- Native Windows desktop application
- provider discovery and subscription-first router
- persistent autonomous tasks
- MCP hands (`apply_patch`, `run_safe_command`)
- task delegation/polling/attention loop
- Windows DPAPI vault
- OpenClaw/harness command adapter
- external interrupt command hook

## v0.5 foundation

- bidirectional Git relay for request/result/attention transport
- public-safe status-only relay mode and private report mode
- durable project/global memory store
- compact project checkpoints for fresh-conversation resume
- bounded context packs instead of replaying full chat history
- reusable routine/code registry with scoped upsert/deduplication
- stable Git-backed project identity with local fallback
- automatic compact successful-task outcome memory
- secret-like content rejection on model-facing memory writes

## Next

- personal-plan Git-relay control envelopes for memory/checkpoint/routine writes when direct custom-MCP actions are unavailable
- semantic retrieval and automatic memory consolidation
- explicit memory archive/forget/edit controls and provenance/usage telemetry
- reusable artefact references for files/components too large for inline routine code
- project knowledge graph, parked ideas and searchable decisions layered over durable memory
- first-class Workbench Runner protocol (remote and local)
- real OpenClaw adapter with structured jobs rather than a command template
- provider entitlement/allowance telemetry where providers expose it
- automatic cross-model review policies
- Git worktree/branch isolation per autonomous task
- richer retry/recovery and resumable sessions
- desktop/tablet/phone preview/test targets
- screenshot and Playwright result surfaces
- WhatsApp/Signal/Telegram/Slack adapters as human-interrupt channels
- OS keychain support on macOS/Linux
- signed installers and automatic updates
- mobile companion for approvals and read-only task status

## Later

- synchronised/encrypted private knowledge backends for multi-machine continuity
- Workbench Runner capable enough to replace a general-purpose external harness for many local/cluster workflows
- distributed runner registry and capability advertisement
- policy packs for production, regulated environments and team use
- community adapter registry
