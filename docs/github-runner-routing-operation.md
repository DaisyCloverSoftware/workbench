# Read-only GitHub runner-routing operation

`scripts/ops/github-runner-routing.sh OWNER/REPO` is a bounded diagnostic for a Windows x64 self-hosted Actions queue. It is not a generic GitHub CLI or shell capability.

Use the existing `run_operations_script` control with the exact reviewed full commit advertised by an origin branch head. Keep project/repository arguments and returned runner metadata in private authenticated transport. No installed Workbench binary update or registered-checkout change is required.

## Contract

The operation validates one repository argument and performs exactly two GitHub.com GET requests: the first page of repository runners and the first page of organization runners for the repository owner. Each page is capped at 100 entries; each subprocess has a 25-second timeout. It uses the existing local GitHub CLI authentication without inspecting or printing tokens. Only IDs, operating systems, online/busy states and labels are returned; debug output, raw errors, runner names and unrelated response fields are excluded.

Missing CLI, denied access, malformed responses, duplicate IDs and incomplete first pages are not evidence of no registered runner. They produce unavailable/partial state and a nonzero exit. Visibility of an organization runner is not proof of repository access, runner-group eligibility, engine installation or ability to execute a particular workflow.

The operation never registers, relabels, starts, stops or deletes runners; never retries a workflow; never changes Actions permissions; and never reads secrets or arbitrary endpoints. No historical cause of runner loss or native Unreal/build success is inferred.

## Verification

Run `bash -n scripts/ops/github-runner-routing.sh` and `python3 scripts/tests/test_github_runner_routing.py` on a POSIX host with Bash and Python 3. Tests use a synthetic GitHub CLI fixture, not live credentials or network calls. Live output must be interpreted separately and retained only in private operational evidence.
