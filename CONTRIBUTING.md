# Contributing

Workbench is deliberately early. Contributions are welcome, especially when they make the system more provider-neutral, safer or less annoying to supervise.

Good first contribution areas include provider detection, entitlement/usage reporting, new harness adapters, safe-command policy tests, notification transports, mobile companion concepts and native preview/test surfaces.

Before opening a pull request:

```bash
gofmt -w .
go test ./...
```

Keep provider-specific assumptions inside adapters. Core routing should reason about capabilities, trust and cost classes rather than brands.

Do not add browser automation for consumer AI chat products as a shortcut around unsupported integration paths.
