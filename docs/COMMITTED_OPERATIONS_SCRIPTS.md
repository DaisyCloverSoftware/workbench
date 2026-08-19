# Committed Operations Scripts

`run_operations_script` is Workbench's direct path for reviewed multi-step Bash operations that already live in source control. It exists to cover deployment/runbook scripts without exposing arbitrary remote shell text and without invoking an external AI worker.

## Contract

A runnable operations script must:

- belong to a project already inside Workbench's authorised project scope;
- have a canonical relative path beneath `scripts/ops/`;
- end in `.sh`;
- exist as a regular Git blob at the project's current `HEAD` commit;
- not be a symlink;
- receive only bounded literal argv entries that do not resemble secret material.

Workbench resolves the full `HEAD` SHA, creates a disposable detached Git worktree at that exact commit, hashes the committed script, and executes:

```text
bash --noprofile --norc <committed-script-path> <literal argv...>
```

Workbench never accepts `bash -c`, pipes, redirects, command substitutions, or arbitrary shell source through this action. Dirty edits in the developer checkout are not copied into the detached worktree and therefore cannot alter the operation.

## Result evidence

The result includes:

- relative script path;
- literal argv;
- exact 40-character Git commit SHA;
- SHA-256 of the executed script bytes;
- bounded stdout/stderr;
- exit code;
- whether output was truncated;
- transport marker `git-worktree-bash`.

Secret-like output is withheld rather than returned to ChatGPT.

## Authority

This is a mutating/open-world operations path. A committed script can still perform significant host/cluster actions, so source-control provenance is not a substitute for user authority. Use it only when the script's intended effect is already within the current permission scope. Do not use it to bypass a production, destructive, credential, or irreversible-action boundary.

If a deployment can be expressed cleanly through `inspect_machine` and `run_machine_command`, those smaller structured primitives remain preferable. Use `run_operations_script` when the repository already contains the reviewed orchestration logic and reproducing it as a long chat-generated shell block would be less safe or less auditable.
