# Shared ChatGPT Integration

Workbench is one execution plane with two ChatGPT transports. Ordinary project chats should use whichever transport their ChatGPT plan can actually authorize; the execution ownership and safety policy are the same either way. The user should not open OpenClaw or carry prompts/results between systems.

This integration guide is subordinate to `docs/GOVERNANCE.md`, `docs/DECISIONS.md` and `SECURITY.md`.

## Transport selection

### Personal ChatGPT plans

Use the **private Git relay as the primary write/mutation transport** when direct custom-MCP writes are unavailable. A fresh chat with connected GitHub can locate the user's private repository whose name contains `workbench-relay`, read `WORKBENCH_CAPABILITIES.json` and `WORKBENCH_CHATGPT.md`, then submit bounded Workbench controls through `relay/control`.

This path supports repository safe-hands plus direct host/cluster inspection/mutation because GitHub is the transport and Workbench performs the already-authorised operation on the runner. It does not invoke another AI model.

A Secure MCP Tunnel may still be connected for MCP capabilities the current ChatGPT plan permits, but Workbench must not make personal-plan cluster writes depend on full custom-MCP write support.

### ChatGPT plans with full custom MCP actions

When the ChatGPT workspace supports full custom MCP write/modify actions, the same Workbench tools can be exposed directly through the Secure MCP Tunnel/custom ChatGPT app. In that environment MCP can be the preferred low-latency transport while the private Git relay remains a recovery/bootstrap path.

## Architecture

```text
                         +---------------------------+
ChatGPT project chat --->| transport selector        |
                         +-------------+-------------+
                                       |
                    +------------------+------------------+
                    |                                     |
          personal-plan write path               full-MCP-capable path
                    |                                     |
                    v                                     v
       private Workbench Git relay             OpenAI Secure MCP Tunnel
                    |                                     |
                    +------------------+------------------+
                                       v
                              Workbench execution core
                                       |
                    +------------------+------------------+
                    |                  |                  |
             repository hands   direct machine ops   durable state/memory
                                       |
                              optional autonomous
                                operator capacity
```

The MCP server itself remains loopback-only. When Secure MCP Tunnel is used, its client makes the outbound connection; no inbound firewall port is opened for Workbench.

## Execution ownership

ChatGPT is the development brain. It owns reasoning, source code, Git/GitHub, pull requests, releases, CI and GitHub Actions orchestration.

Routine host/cluster operations are also driven by ChatGPT directly through bounded Workbench controls:

- `inspect_machine` executes one read-only allowlisted program + argv command on the configured Workbench operator host;
- `run_machine_command` executes one explicitly allowlisted mutating program + argv command;
- neither invokes a shell or a second AI model;
- external AI/operator availability is not a prerequisite for ordinary bounded infrastructure diagnostics/lifecycle work.

`delegate_operation` remains optional autonomous operator capacity when an outcome cannot reasonably be decomposed into the direct structured surface and such capacity is intentionally available. It is not the default route.

## Shared tool model

Both transports map onto the same Workbench control model. The MCP surface can include:

- workspace/context/memory reads and writes;
- bounded repository list/search/read;
- exact patch application;
- allowlisted safe commands;
- bounded machine inspection/mutation;
- task/dependency/attention state;
- optional explicit autonomous operations.

The private relay exposes corresponding safe-hands/direct-machine controls through `relay/control`. `relay/inbox` is reserved for optional autonomous machine-side work rather than routine deterministic execution.

## Direct machine command boundary

The direct executor is intentionally not a generic remote shell.

- Requests contain an exact executable basename and argv array; Workbench uses direct process execution rather than shell composition.
- Executable/subcommand are allowlisted.
- Read-only inspection and mutation are separate actions so authority is explicit.
- Alternate credential targets, credential-bearing arguments, secret reads, arbitrary scripts and high-risk primitives are refused by policy unless a separate exact trusted operation exists.
- Potentially unbounded streams are refused.
- Arguments that resemble secret material are refused; output that resembles secret material is withheld.
- Repository `run_safe_command` remains a separate development/test/build/status surface.
- A configured K3s control host may internally reuse its existing non-interactive privileged `k3s kubectl` boundary only when the validated operation requires it; generic privilege escalation is not exposed as a model-callable program.

This split lets ChatGPT operate authorised infrastructure without making Workbench an unrestricted shell endpoint.

## Windows host boundary

Windows bridge operations are separately outbound and typed/allowlisted. Do not treat generic cluster `run_machine_command` as permission to create a generic Windows shell endpoint. Windows capabilities such as Blender/Unreal operations require their own typed contracts.

## Personal-plan bootstrap

No OpenAI tunnel secret is required for the private relay path.

1. Keep the private Workbench relay running on the operator host.
2. A fresh ChatGPT project chat uses connected GitHub to locate the private repository whose name contains `workbench-relay`.
3. Read `WORKBENCH_CAPABILITIES.json` first, then `WORKBENCH_CHATGPT.md` for the transport/behavioural contract.
4. Use unique request IDs under `relay/control/<id>.json`; read the matching `relay/control-outbox/<id>.json` result.
5. Prefer deterministic bounded controls for routine work. Do not use autonomous relay paths merely because an autonomous harness exists.

The private bootstrap documents are transport contracts. They do not override the target project's canonical repository requirements/decision record.

## Optional Secure MCP Tunnel setup

The OpenAI-side tunnel/app identities are user/workspace-specific and deliberately not committed to this public repository.

1. Install/start the loopback MCP service using the generic Workbench installer.
2. Configure the approved outbound tunnel/runtime identity in the relevant provider/workspace.
3. Store runtime credentials only in protected local configuration; never paste them into ChatGPT or commit them.
4. Connect the custom Workbench app/tool surface as supported by the plan/workspace.
5. Keep the private relay as bootstrap/recovery transport where useful.

Do not describe optional MCP tunnel setup as a prerequisite for personal-plan Workbench writes.

## Security boundaries

- Private relay report mode is allowed only on transport Workbench verifies as private.
- Runtime/tunnel and MCP credentials remain separate protected local secrets.
- The public repository contains no runtime key, bearer token or workspace-specific app ID.
- Tunnel and Windows transports are outbound-only.
- Direct machine execution does not accept arbitrary shell text/credentials/unrestricted executables.
- Optional autonomous operator delegation remains separate from deterministic execution.
- Project/session activity metadata is not individual job-execution authority; Operations semantics are governed by `docs/operations-dashboard-contract.md`.

## North-star experience

Any project chat with the user's connected Workbench transport should be able to receive an intent, inspect/edit the relevant repository, operate authorised machines through bounded controls, continue through non-human waits, and return a verified result without requiring OpenClaw/model credit or human message shuttling. Transport differences should be an implementation detail, not something the human has to manage.
