# OpenClaw operations CLI contract

Workbench uses OpenClaw only for machine-side operations that ChatGPT cannot execute itself.

The supported scripted invocation is:

```text
openclaw agent --message <prompt>
```

`openclaw agent` is already a non-interactive/scripted command. Workbench must not append the browser-only `--headless` flag. If OpenClaw changes this contract, update the operations adapter and its regression tests together.

This does not change the ownership boundary: ChatGPT owns source code, Git/GitHub, pull requests, CI, GitHub Actions and release orchestration. OpenClaw owns only the delegated host/server/cluster/runtime operation.
