# Public source privacy policy

Workbench's public repository contains generic product code, documentation and examples only.

Do not publish maintainer- or user-specific deployment data, including machine or tailnet names, local usernames or absolute home-directory paths, private addresses, private service topology, provider-account inventory or entitlement state, credentials, runtime secrets, private task content, or dogfood logs.

Use placeholders, documentation address ranges and generic capability descriptions in public examples. Real deployment configuration belongs in local protected storage or a private authenticated repository.

`go test ./...` includes a conservative public-source privacy guard for common accidental environment leaks. A failing guard is a reason to generalize the public material, not to add a private value to an allowlist.
