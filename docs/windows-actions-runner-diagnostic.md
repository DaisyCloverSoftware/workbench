# Read-only Windows Actions runner diagnostic

## Contract

`inspect_windows_actions_runner` accepts only `args.host_id`, never a project,
path, URL, command, service name or script. It submits a typed
`actions_runner_diagnostic / inspect_v1` host job only after a fresh Windows
heartbeat advertises diagnostic capability version `1`. That capability means
this read-only implementation is present, not that an Actions runner is
installed or ready. Both server and Windows client need the extension.
Old clients fail closed before submission. Use `get_windows_host_job` for the
terminal nested job result; a submission acknowledgement is not proof.

The native collector starts no program or shell. It uses local Windows service
and process query APIs. It requests neither service start/control rights nor
process termination/memory rights. It selects only Actions-named services and
Runner.Listener/Runner.Worker/RunnerService images. Service accounts, raw service
commands, process command lines, local absolute paths and unrelated process
records are not returned. Installation/service IDs are opaque path/name hashes.

Candidate roots come from those service/image paths and three conventional
locations: the system drive's actions-runner, the current user's actions-runner,
and the current user's work/actions-runner. No recursive search, other-user
profile search, network share or arbitrary path is supported. At most 8 roots,
32 service/process records, 4 bounded service pages and 16384 process snapshot
entries are considered. Deadline checks are cooperative between local Windows
calls; an OS API call itself is not forcibly interrupted.

Only .runner is read as bounded metadata. .service, config.cmd and two runner
images are checked for presence. Credential files, logs, environment files,
source files and the configured work directory are never read. Reads reject
reparse points, path aliases, non-fixed drives, hard-linked metadata, oversized
files, credential-bearing/non-GitHub targets and duplicate JSON keys. Parent
directories are held against rename while reading; Git worktrees are excluded.
A presence marker is not a verified or signed runner binary.

## Meaning of the result

.runner contains a local agent ID/target, not authoritative registration or
labels. Labels must be read from GitHub, never inferred from host OS.
ClassifyRegistration requires a freshly completed inventory of the same scope
before comparing IDs. An ID absent from that scope is an orphaned local
configuration observation, not permission to delete or re-register anything.
A matching ID proves registration presence only, not online/idle availability.

Services can be silently omitted by Windows when query permission is missing.
A process snapshot can race exits and omit inaccessible image paths. Missing
conventional locations do not establish machine-wide absence:
`machine_wide_absence_established` remains false. Do not propose provisioning on
a partial, denied or scoped-empty result. No record promotes a heartbeat,
inventory, service state or binary presence to Unreal proof.

## Delivery and acceptance

The zero-argument workbench-actions-runner-inspect binary exposes the same local
collector for isolated native validation. It prints the bounded report and
cannot start/register runners or accept a target. Tests use synthetic temp
fixtures and include a query-only hosted-Windows smoke without printing machine
inventory. Hosted Windows evidence is not evidence from an owner's workstation.

Deploy the reviewed client/relay through the approved delivery path. This
tranche adds no upgrade operation and grants no forced-restart authority.
Do not masquerade this operation as Blender/Unreal, smuggle scripts into smoke
jobs, bypass installed-client capability checks or publish fake heartbeats.
After client delivery, obtain the actual host job result. Keep this first
extension read-only and preserve existing worktrees.

## Subsequent gates: design only, no recovery mutation implemented

- Registered but stopped: recheck same-scope agent ID, GitHub labels, exact local
  service/image identity and absence of a worker. Design a separate typed
  start-existing-service operation pinned to that observed identity, verified
  binary, local config digest and safe work directory. Refuse ambiguity,
  disabled services, a running worker or changed/missing registration. Starting
  a runner may immediately execute an already-queued job; assess that first.
- Locally configured but orphaned: require complete matching-scope GitHub
  absence and a quiet installation. Preserve configuration/work, verify the
  official runner package and safe dedicated work directory, then design the
  official remove/reconfigure flow with a short-lived registration token using
  a non-relay secret channel. No tokens in Git/logs/caller arguments, no generic
  shell, no --replace against another registration. This diagnostic authorizes
  no deletion. Exact steps depend on observed state.
- No installation observed: report checked scope and permission gaps; resolve
  uncovered plausible locations before proposing provisioning.

Only a compatible registered online/idle Windows runner with verified
repository/group access justifies allowing the existing exact-head job to
proceed. Never retry because a diagnostic or its CI passed.

## Primary references

- https://learn.microsoft.com/windows/win32/api/winsvc/nf-winsvc-enumservicesstatusexw
- https://learn.microsoft.com/windows/win32/api/winsvc/nf-winsvc-queryserviceconfigw
- https://learn.microsoft.com/windows/win32/api/winbase/nf-winbase-queryfullprocessimagenamew
- https://github.com/actions/runner/blob/main/src/Runner.Common/ConfigurationStore.cs
- https://docs.github.com/actions/how-tos/manage-runners/self-hosted-runners/remove-runners
- https://docs.github.com/actions/how-tos/manage-runners/self-hosted-runners/configure-the-application
