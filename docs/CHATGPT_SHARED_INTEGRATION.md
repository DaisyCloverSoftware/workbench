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
        +-- durable Workbench task state
        +-- supervised OpenClaw machine operations
```

The Workbench MCP server stays loopback-only. `tunnel-client` makes the outbound connection to OpenAI; no inbound firewall port is opened for Workbench.

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
- `save_note`
- `delegate_operation`
- `await_operation`
- `get_task`
- `list_tasks`
- `resolve_attention`

ChatGPT owns reasoning, source code, Git/GitHub, pull requests, releases, CI, and GitHub Actions. `delegate_operation` is only for host/server/cluster/runtime work ChatGPT cannot execute directly.

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
- Machine operations remain behind Workbench's durable task and attention model; installing the shared connection does not broaden user authority.

## Rebuilding the personal plugin binding

The generated personal plugin is disposable. If the ChatGPT connection is recreated and receives a new `plugin_asdk_app...` ID, rerun the packaging helper with the new ID. The helper replaces only an existing Workbench-owned personal plugin directory and leaves unrelated marketplace entries intact.
