# Shared ChatGPT Integration

Workbench is one execution plane with two ChatGPT transports. Ordinary project chats should use whichever transport their ChatGPT plan can actually authorize; the execution ownership and safety policy are the same either way. The user should not open OpenClaw or carry prompts/results between systems.

## Transport selection

### Personal ChatGPT plans

Use the **private Git relay as the primary write/mutation transport**. A fresh chat with connected GitHub can locate the user's private repository whose name contains `workbench-relay`, read `WORKBENCH_CAPABILITIES.json` and `WORKBENCH_CHATGPT.md`, then submit bounded Workbench controls through `relay/control`.

This path supports repository safe-hands plus direct host/cluster inspection and mutation because GitHub is the transport and Workbench performs the already-authorised operation on the runner. It does not invoke another AI model.

A Secure MCP Tunnel may still be connected for MCP capabilities the current ChatGPT plan permits, but Workbench must not make personal-plan cluster writes depend on full custom-MCP write support.

### ChatGPT plans with full custom MCP actions

When the ChatGPT workspace supports full custom MCP write/modify actions, the same Workbench tools can be exposed directly through the Secure MCP Tunnel and custom ChatGPT app. In that environment the MCP app can become the preferred low-latency transport while the private Git relay remains a recovery/bootstrap path.

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
                                operator fallback
```

The MCP server itself remains loopback-only at `127.0.0.1:8765/mcp`. When Secure MCP Tunnel is used, `tunnel-client` makes the outbound connection to OpenAI; no inbound firewall port is opened for Workbench.

## Execution ownership

ChatGPT is the development brain. It owns reasoning, source code, Git/GitHub, pull requests, releases, CI, and GitHub Actions.

Routine host and cluster operations are also driven by ChatGPT directly through Workbench:

- `inspect_machine` executes one read-only allowlisted program + argv command on the configured Workbench operator host.
- `run_machine_command` executes one explicitly allowlisted mutating program + argv command.
- Neither tool invokes a shell or a second AI model.
- External AI/operator availability is therefore not a prerequisite for ordinary Kubernetes, Helm, systemd, journal, Docker/runtime, or host diagnostics and bounded lifecycle operations.

`delegate_operation` remains available only as optional autonomous operator capacity when a machine-side outcome cannot reasonably be decomposed into the direct structured command surface and such capacity is intentionally available.

## Shared tool model

Both transports map onto the same Workbench operations model. The MCP surface includes:

- `get_workspace`
- `get_context`
- `search_memory`
- `list_files`
- `search_text`
- `read_file`
- `save_memory`
- `save_context`
- `apply_patch`
- `run_safe_command`
- `inspect_machine`
- `run_machine_command`
- `save_note`
- `delegate_operation`
- `await_operation`
- `get_task`
- `list_tasks`
- `resolve_attention`

The private relay exposes the corresponding safe-hands/direct-machine actions through `relay/control`. `relay/inbox` is reserved for optional autonomous machine-side work rather than routine command execution.

## Direct machine command boundary

The direct executor is intentionally not a generic remote shell.

- Requests contain an exact executable basename and an argv array; Workbench uses direct process execution rather than `bash -c`, `sh -c`, PowerShell, command substitution, pipes or redirects.
- The executable and subcommand are allowlisted.
- Read-only inspection and mutation are separate actions so authority is explicit.
- Alternate cluster/host credential targets, credential-bearing arguments, Kubernetes Secret reads, arbitrary scripts and high-risk primitives such as delete/exec/remove are refused by the initial policy.
- Potentially unbounded streams such as journal follow or Docker stats without `--no-stream` are refused.
- Machine command arguments that resemble secret material are refused; output that resembles secret material is withheld from ChatGPT.
- Repository `run_safe_command` stays separate and remains limited to development/test/build/status/diff work.
- On the DaisyClover K3s control host, a validated `kubectl` request may internally reuse the cluster's existing non-interactive `sudo -n k3s kubectl` boundary only when direct kubectl fails specifically because the root-owned K3s kubeconfig cannot be read. `sudo` is not exposed as a Workbench-callable program.

This split lets ChatGPT operate the cluster without making Workbench an unrestricted shell endpoint.

## Personal-plan bootstrap

No OpenAI tunnel secret is required for the private relay path.

1. Keep the private Workbench relay running on the operator host.
2. A fresh ChatGPT project chat uses connected GitHub to locate the private repository whose name contains `workbench-relay`.
3. Read `WORKBENCH_CAPABILITIES.json` first, then `WORKBENCH_CHATGPT.md` for the behavioural contract.
4. Use unique request IDs under `relay/control/<id>.json`; read the matching `relay/control-outbox/<id>.json` result.
5. Prefer `inspect_machine` / `run_machine_command` for routine host/cluster work. Do not use `relay/inbox` merely because OpenClaw exists.

## Optional Secure MCP Tunnel setup

The OpenAI-side tunnel/app identities are user/workspace-specific and are deliberately not committed to this public repository.

1. Install/start the loopback MCP service on the Workbench host with `scripts/install-cluster-mcp.sh`.
2. In OpenAI Platform tunnel settings, create/choose a tunnel associated with the ChatGPT workspace and create a restricted runtime key with the tunnel permissions required to run it.
3. On the Workbench host, set `WORKBENCH_TUNNEL_ID=tunnel_...` (or enter the non-secret tunnel ID when prompted) and run `scripts/install-openai-tunnel.sh`. Enter the runtime key only into that local terminal prompt. Do not paste it into ChatGPT or commit it to Git.
4. In ChatGPT developer/app settings, create the custom Workbench app using the tunnel and scan its available tools.
5. If the local plugin packaging flow is being used, bind the resulting technical app ID with `scripts/package-chatgpt-plugin.sh` or `scripts/package-chatgpt-plugin.ps1`.

Do not describe this optional MCP setup as a prerequisite for personal-plan Workbench writes.

## Security boundaries

- Private relay report mode is allowed only on a transport Workbench verifies as private.
- The OpenAI runtime API key, when a tunnel is used, stays only on the Workbench host in a mode-0600 file.
- The Workbench MCP bearer is stored separately in a mode-0600 file and is injected only on the local MCP hop by `tunnel-client`.
- The public repository contains no runtime key, bearer token, or workspace-specific app ID.
- Tunnel transport is outbound-only from the Workbench network.
- Direct machine execution does not accept arbitrary shell text, credentials, alternate cluster targets, or unrestricted executables.
- Optional autonomous operator delegation remains separate from direct deterministic execution.

## North-star experience

Any project chat with the user's connected Workbench transport should be able to receive an intent, inspect/edit the relevant repository, operate the host/cluster through bounded direct commands, and return a verified result without OpenClaw/model credit. Transport differences should be an implementation detail, not something the human has to manage.
