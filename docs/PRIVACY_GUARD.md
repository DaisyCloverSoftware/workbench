# Automated public-source privacy guard

`go test ./...` scans tracked source for common accidental environment leaks such as absolute user-home paths, private/Tailscale addresses, tailnet hostnames and shell user-at-host prompts.

The guard is intentionally generic. It does not contain maintainer-specific deny-list values, because publishing those values would defeat the purpose of the check.

If a legitimate example trips the guard, prefer documentation address ranges, placeholders and environment variables rather than weakening the rule around a real deployment value.
