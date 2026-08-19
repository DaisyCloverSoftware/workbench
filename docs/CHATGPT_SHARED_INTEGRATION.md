# Shared ChatGPT Integration

Workbench's target integration is one private MCP connection that can be reused by ordinary ChatGPT project chats. A project chat should call Workbench tools directly; the user should not open OpenClaw or carry prompts/results between systems.

## Architecture

```text
ChatGPT project chat
        |
        | registered Workbench MCP app/plugin
        v
OpenAI Secure MCP Tunnel endpoint
        ^
        | outbound HTTPS only
        |
tunnel-client on the Workbench host
        |
        | loopback HTTP + static bearer header
        v
Workbench MCP 127.0.0.1:8765/mcp
        |
        +-- bounded repository hands
        +-- direct structured machine execution
        +-- durable Workbench state/memory
        +-- optional autonomous operator fallback
```

The Workbench MCP server stays loopback-only. `tunnel-client` makes the outbound connection to OpenAI; no inbound firewall port is opened for Workbench.

## Execution ownership

ChatGPT is the development brain. It owns reasoning, source code, Git/GitHub, pull requests, releases, CI, and GitHub Actions.

Routine host and cluster operations are also driven by ChatGPT directly through Workbench:

- `inspect_machine` executes one read-only allowlisted program + argv command on the configured Workbench operator host.
- `run_machine_command` executes one explicitly allowlisted mutating program + argv command.
- Neither tool invokes a shell or a second AI model.
- External AI/operator availability is therefore not a prerequisite for ordinary Kubernetes, Helm, systemd, journal, Docker/runtime, or host diagnostics and bounded lifecycle operations.

`delegate_operation` remains available only as optional autonomous operator capacity when a machine-side outcome cannot reasonably be decomposed into the direct structured command surface.

## What is shared

The shared ChatGPT connection exposes the current Workbench MCP surface:

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

The private Git relay exposes the same direct machine command model through `relay/control` when that transport is needed. `relay/inbox` is reserved for optional autonomous machine-side work rather than routine command execution.

## Direct machine command boundary

The direct executor is intentionally not a generic remote shell.

- Requests contain an exact executable basename and an argv array; Workbench uses direct process execution rather than `bash -c`, `sh -c`, PowerShell, command substitution, pipes or redirects.
- The executable and subcommand are allowlisted.
- Read-only inspection and mutation are separate tools so review metadata is explicit.
- Alternate cluster/host credential targets, credential-bearing arguments, Kubernetes Secret reads, arbitrary scripts and high-risk primitives such as delete/exec/remove are refused by the initial policy.
- Potentially unbounded streams such as journal follow or Docker stats without `--no-stream` are refused.
- Machine command arguments that resemble secret material are refused; output that resembles secret material is withheld from ChatGPT.
- Repository `run_safe_command` stays separate and remains limited to development/test/build/status/diff work.

This split lets ChatGPT operate the cluster without making Workbench an unrestricted shell endpoint.

## One-time connection setup

The OpenAI-side tunnel/app identities are user/workspace-specific and are deliberately not committed to this public repository.

1. Install/start the loopback MCP service on the Workbench host with `scripts/install-cluster-mcp.sh`.
2. In OpenAI Platform tunnel settings, create/choose a tunnel associated with the ChatGPT workspace that will use Workbench and create a restricted runtime key with the tunnel permissions required to run it.
3. On the Workbench host, set `WORKBENCH_TUNNEL_ID=tunnel_...` (or enter the non-secret tunnel ID when prompted) and run `scripts/install-openai-tunnel.sh`. Enter the runtime key only into that local terminal prompt. Do not paste it into ChatGPT or commit it to Git.
4. Enable ChatGPT developer mode, create a developer-mode app in ChatGPT Plugins, choose **Tunnel**, select/paste the same `tunnel_id`, and create the Workbench connection.
5. Copy the connection's technical ID from ChatGPT. It starts with `plugin_asdk_app`.
6. On the computer where the local Workbench plugin should be installed, run `scripts/package-chatgpt-plugin.sh <plugin_asdk_app...>` (POSIX) or `scripts/package-chatgpt-plugin.ps1 -AppId <plugin_asdk_app...>` (Windows). The helper creates a personal Workbench plugin bundle and personal marketplace entry without committing the workspace-specific app ID.
7. Refresh/restart ChatGPT desktop as required by the local marketplace flow, install/enable Workbench, and use it from a fresh project chat.

## Security boundaries

- The runtime API key is stored only on the Workbench host in a mode-0600 file.
- The Workbench MCP bearer is stored separately in a mode-0600 file and is injected only on the local MCP hop by `tunnel-client`.
- The public repository contains no runtime key, bearer token, or workspace-specific `plugin_asdk_app...` ID.
- Tunnel transport is outbound-only from the Workbench network.
- Direct machine execution does not accept arbitrary shell text, credentials, alternate cluster targets, or unrestricted executables.
- Mutating machine tools are advertised as destructive/open-world so the ChatGPT product can apply the appropriate review/permission boundary.
- Optional autonomous operator delegation remains separate from direct deterministic execution.

## Rebuilding the personal plugin binding

The generated personal plugin is disposable. If the ChatGPT connection is recreated and receives a new `plugin_asdk_app...` ID, rerun the packaging helper with the new ID. The helper replaces only an existing Workbench-owned personal plugin directory and leaves unrelated marketplace entries intact.
