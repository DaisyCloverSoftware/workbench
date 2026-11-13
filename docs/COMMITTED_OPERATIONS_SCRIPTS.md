# Committed Operations Scripts

`run_operations_script` is Workbench's direct path for reviewed multi-step Bash operations that already live in source control. It exists to cover deployment/runbook scripts without exposing arbitrary remote shell text and without invoking an external AI worker.

## Contract

A runnable operations script must:

- belong to a project already inside Workbench's authorised project scope;
- have a canonical relative path beneath `scripts/ops/`;
- end in `.sh`;
- exist as a regular Git blob at the selected commit; by default that is local `HEAD`, or callers may supply a full 40-character `commit` currently advertised by a branch head on the project's credential-free `github.com` `origin`;
- not be a symlink;
- receive only bounded literal argv entries that do not resemble secret material.

Workbench normally resolves the full local `HEAD` SHA. If `commit` is supplied, Workbench verifies the SHA is currently advertised by an `origin` branch head, fetches that exact branch into a separate disposable Git repository, verifies the fetched commit matches exactly, and leaves the registered checkout untouched. It then creates a detached worktree at the selected commit, hashes the committed script, and executes:

```text
bash --noprofile --norc <committed-script-path> <literal argv...>
```

Workbench never accepts `bash -c`, pipes, redirects, command substitutions, or arbitrary shell source through this action. Dirty edits in the developer checkout are not copied into the detached worktree and therefore cannot alter the operation. Remote-commit mode also avoids switching, pulling, resetting, merging, or fast-forwarding the registered checkout merely to obtain the reviewed script.

## Result evidence

The result includes:

- relative script path;
- literal argv;
- exact 40-character Git commit SHA;
- SHA-256 of the executed script bytes;
- bounded stdout/stderr;
- exit code;
- whether output was truncated;
- transport marker `git-worktree-bash` for local HEAD or `github-origin-commit-worktree-bash` for an exact advertised origin commit.

Secret-like output is withheld rather than returned to ChatGPT.

### Lifecycle is separate from output availability

Implementing the truthful-evidence requirements of WB-DEC-003 and WB-DEC-014, result contract version 1 adds executor-owned `execution_status`. The private relay preserves the same typed information as top-level `operation_status`, even when the existing result payload is withheld or the operation returns an error. The envelope request ID and executor source commit/script digest provide correlation. The executor version identifies the installed implementation.

`started` means process creation succeeded. `completion` is `not_started`, `exited`, `timed_out`, `interrupted` or `unknown`. `process_result_observed` records whether the executor obtained an OS process state. `process_exit_code` is an integer only when a normal exit code is available, otherwise null. The legacy `exit_code` remains for callers; unavailable results use -1, not invented success. No persisted record after executor interruption means UNKNOWN, not proof that a process never started.

`output_withheld` and `output_truncated` are independent of execution. Existing protection of stdout/stderr and the private result payload is retained. A failed envelope can therefore contain a known process exit 0 with protected output withheld. Neither exit 0 nor a complete report proves an application's acceptance, rollback correctness or email delivery. Conversely, nonzero exit, timeout and withholding do not prove that no mutation occurred.

Dropping executor status merely because an operation returns an error is superseded. Do not reintroduce that behaviour. Historical outboxes are not rewritten or reinterpreted by this change.

### Optional closed, forward-only disposition

A reviewed operation may provide bounded maintenance evidence through the executor-created file identified by `WORKBENCH_OPERATION_STATUS_FILE`, when `WORKBENCH_OPERATION_STATUS_VERSION=1`. This is an optional extension: old operations need not write it and keep their existing execution/error behaviour. Their application disposition remains unreported.

The executor exclusively creates a fresh file before starting the process, retains its handle and identity, and reads only that file after process termination. It rejects path/inode replacement, symlinks, non-regular files, oversized files and unstable reads. There is no caller-selected reader, historical audit retrieval, output-parser fallback or exemption for a script/path. The file is removed with the existing disposable execution directory. Write the provided file in place; replacing it by rename is rejected.

The maximum size is 2048 bytes. Exactly these ten fields are required; duplicate/unknown/missing/null keys, trailing objects, wrong types and arbitrary values are rejected without returning their contents:

| Field | Allowed values |
| --- | --- |
| `version` | integer 1 |
| `terminal` | boolean |
| `stage` | `unknown`, `initialisation`, `preflight`, `update`, `readback`, `restart`, `pod_consistency`, `authentication`, `health`, `readiness`, `startup_logs`, `preservation`, `source_readback`, `rollback_precheck`, `rollback_update`, `rollback_readback`, `rollback_restart`, `rollback_verification`, `complete` |
| `reason` | `none`, `unknown`, `management`, `baseline`, `configuration`, `concurrency`, `command_failed`, `timeout`, `interrupted`, `invalid_response`, `pod_changed`, `authentication_rejected`, `tls_failed`, `connection_failed`, `provider_rejected`, `health_failed`, `readiness_failed`, `startup_log_missing`, `startup_log_incomplete`, `preservation_failed`, `audit_failed`, `reporting_failed`, `internal_failure` |
| `update` | `unknown`, `not_attempted`, `not_applied`, `confirmed`, `unexpected` |
| `recovery` | `unknown`, `not_attempted`, `confirmed`, `unconfirmed`, `blocked_concurrent_change`, `not_attempted_update_unresolved` |
| `authentication` | `unknown`, `passed`, `failed`, `not_attempted` |
| `health`, `readiness` | `unknown`, `200`, `503`, `other`, `not_attempted` |
| `audit` | `unknown`, `complete`, `incomplete` |

Only the validated enum/boolean values are reserialised as `script_reported_disposition`. They are assertions from the reviewed script, not independently derived executor facts. Free text, identifiers, credentials, hashes, raw errors, authentication exchanges and extra fields cannot be transported by this schema. Source/destination names belong in reviewed source, not in this status file. A script must never derive this report from protected historical results.

`reporting` is `not_reported`, `complete`, `incomplete`, `invalid` or `unavailable`. A missing file/report does not imply update not attempted. A non-terminal report or one with incomplete/unknown audit remains incomplete. Invalid input is discarded; no partial free-text recovery occurs. A reporting failure must not overwrite a known process result. Required audit/recovery behaviour remains the operation's responsibility; this status mechanism does not make required audit optional.

`evidence_complete` is false whenever process evidence is missing, the process did not exit normally, output was withheld/truncated, or no complete terminal disposition was supplied. It describes evidence availability, not application success. Consumers may use valid categorical findings while explicitly retaining UNKNOWN for facts not observed. A report with `audit=incomplete` must not be represented as complete evidence even if some fields are useful.

The raw stdout/stderr protection decision is unchanged. Lifecycle fields and the closed-schema report are generated/validated separately rather than extracted from a withheld stream. Negative protection tests must remain green. Actual-interface synthetic tests are required before relying on a new reporting contract for privileged maintenance; installing a version alone is not sufficient acceptance.

## Authority

This is a mutating/open-world operations path. A committed script can still perform significant host/cluster actions, so source-control provenance is not a substitute for user authority. Use it only when the script's intended effect is already within the current permission scope. Do not use it to bypass a production, destructive, credential, or irreversible-action boundary.

If a deployment can be expressed cleanly through `inspect_machine` and `run_machine_command`, those smaller structured primitives remain preferable. Use `run_operations_script` when the repository already contains the reviewed orchestration logic and reproducing it as a long chat-generated shell block would be less safe or less auditable.
