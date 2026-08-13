# Private loop self-update

A private Personal Pro relay can refresh Workbench itself without turning the human into a shell-command courier.

The private control action is intentionally narrow:

```json
{
  "version": 1,
  "id": "<unique-id>",
  "action": "update_workbench",
  "args": {}
}
```

It is accepted only through the private relay control path. Public relay mode never processes private control requests.

The action does **not** accept a command, repository URL, branch, project path or deployment target. The relay reuses only its already-configured private Git origin and the configured Workbench source tree, then schedules the existing bootstrap out-of-process. The bootstrap itself requires clean clones, fast-forwards source, runs the full test suite, rebuilds the MCP/relay binaries and restarts their user services.

The update is delayed briefly so the relay can publish the control result before its own service is restarted.

This capability is for maintaining the Workbench control plane. It is not a general remote shell and must not be extended into an arbitrary deployment mechanism.
