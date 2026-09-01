# Shared ChatGPT Integration

Workbench is one execution plane with two ChatGPT transports. Ordinary project chats should use whichever transport their ChatGPT plan can actually authorize; the execution ownership and safety policy are the same either way. The user should not have to carry prompts/results between systems.

This integration guide is subordinate to `docs/GOVERNANCE.md`, `docs/DECISIONS.md` and `SECURITY.md`.

## Transport selection

### Personal ChatGPT plans

Use the **private Git relay as the primary write/mutation transport** when direct custom-MCP writes are unavailable. A fresh chat with connected GitHub locates the user's private repository whose name contains `workbench-relay`, reads `WORKBENCH_CAPABILITIES.json` and `WORKBENCH_CHATGPT.md`, then submits bounded Workbench controls through `relay/control`.

This path supports repository safe-hands plus direct host/cluster inspection/mutation because GitHub is the transport and Workbench performs the already-authorised operation on the runner. Routine direct execution does not invoke another AI model.

A Secure MCP Tunnel may still be connected for MCP capabilities the current ChatGPT plan permits, but Workbench must not make personal-plan cluster writes depend on full custom-MCP write support.

### ChatGPT plans with full custom MCP actions

When the ChatGPT workspace supports full custom MCP write/modify actions, the same Workbench tools can be exposed directly through the Secure MCP Tunnel/custom ChatGPT app. In that environment MCP can be the preferred low-latency transport while the private Git relay remains a recovery/bootstrap path.

Transport selection does not change authority. **OpenClaw is owner-opt-in only on every transport.**

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
```

OpenClaw is deliberately absent from the normal execution graph. It is a separately owner-selected mode that is unavailable to automatic routing.

The MCP server itself remains loopback-only. When Secure MCP Tunnel is used, its client makes the outbound connection; no inbound firewall port is opened for Workbench.

## Execution ownership

ChatGPT is the development brain. It owns reasoning, source code, Git/GitHub, pull requests, reviews, releases, CI, GitHub Actions orchestration and subsequent engineering decisions.

Routine host/cluster operations are also driven by ChatGPT directly through bounded Workbench controls:

- `inspect_machine` executes one read-only allowlisted program + argv command on the configured Workbench operator host;
- `inspect_machine_batch` executes a bounded ordered set of read-only inspections under the same single-command policy;
- `run_machine_command` executes one explicitly allowlisted mutating program + argv command;
- `run_operations_script` executes a reviewed Git-tracked `scripts/ops/*.sh` operation under its exact bounded contract;
- none of these actions requires OpenClaw or another AI worker.

If the current direct surface cannot express an operation, ChatGPT must first attempt safe decomposition, use an existing reviewed operation, or implement an appropriate bounded capability/reviewed operation within the authorised engineering scope. If none applies, report the exact capability/authority blocker. **Do not convert a direct-capability miss into autonomous/OpenClaw delegation.**

## OpenClaw authorization boundary

OpenClaw is an owner-selected execution mode. ChatGPT and Workbench MUST NOT select, invoke, suggest or use it automatically.

Only an explicit owner instruction naming OpenClaw for the applicable operation authorizes OpenClaw. The following do not authorize it:

- difficulty or duration;
- an allowlist miss or missing Workbench capability;
- CI/deployment failure;
- Kubernetes, Docker, systemd or Helm trouble;
- Bash or multi-step troubleshooting requirements;
- previous OpenClaw use or historical task state;
- OpenClaw being installed, healthy, configured or otherwise available.

Availability does not constitute authorization. Unless the owner explicitly asks for OpenClaw by name, authorization is denied.

On the private relay, `relay/inbox` may preserve deliberate explicit-use functionality, but it is not a fallback route. The inbox requires the manifest-advertised owner-authorization signal in addition to the `[workbench:operations]` routing marker. That operations marker by itself is metadata, not owner consent, and normal routing must never synthesize the authorization signal.

## Shared tool model

Both transports map onto the same Workbench control model. The normal execution surfaces can include:

- workspace/context/memory reads and writes;
- bounded repository list/search/read;
- exact patch application;
- allowlisted safe commands;
- bounded machine inspection/mutation;
- reviewed committed operations scripts;
- task/dependency/attention state.

The private relay exposes corresponding safe-hands/direct-machine controls through `relay/control`. Autonomous relay paths are excluded from automatic selection.

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

## Reviewed operations scripts

When a safe multi-step Bash operation is inappropriate as individual direct commands, use a reviewed Git-tracked `scripts/ops/*.sh` operation through `run_operations_script` when the current capability manifest exposes it. Caller-supplied arbitrary shell text is not a substitute.

A missing reviewed operation is a capability/design question for ChatGPT to solve within scope, or a precise capability boundary to report. It is not permission to route to OpenClaw.

## Windows host boundary

Windows bridge operations are separately outbound and typed/allowlisted. Do not treat generic cluster `run_machine_command` as permission to create a generic Windows shell endpoint. Windows capabilities such as Blender/Unreal operations require their own typed contracts.

## Personal-plan bootstrap

No OpenAI tunnel secret is required for the private relay path.

1. Keep the private Workbench relay running on the operator host.
2. A fresh ChatGPT project chat uses connected GitHub to locate the private repository whose name contains `workbench-relay`.
3. Read `WORKBENCH_CAPABILITIES.json` first, then `WORKBENCH_CHATGPT.md` for the transport/behavioural contract.
4. Use unique request IDs under `relay/control/<id>.json`; read the matching `relay/control-outbox/<id>.json` result.
5. Prefer deterministic bounded controls and reviewed operations for routine work.
6. Treat OpenClaw as denied unless the current owner instruction explicitly names OpenClaw.

Historical conversations, prior OpenClaw tasks and old routing descriptions do not override this bootstrap.

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
- OpenClaw/autonomous execution requires explicit owner authorization and remains separate from deterministic execution.
- Project/session activity metadata is not individual job-execution authority; Operations semantics are governed by `docs/operations-dashboard-contract.md`.

## North-star experience

Any project chat with the user's connected Workbench transport should be able to receive an intent, inspect/edit the relevant repository, operate authorised machines through bounded controls, continue through non-human waits, and return a verified result without requiring OpenClaw/model credit or human message shuttling. Transport differences should be an implementation detail, not something the human has to manage.
