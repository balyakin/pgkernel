# Contributing to pgkernel

Thanks for contributing.

## Development workflow

1. Fork repository and create a feature branch.
2. Implement change with tests.
3. Run local checks:

```bash
go test ./...
go vet ./...
```

4. Submit pull request with benchmark notes for behavior-changing checks.

## New check proposal template

When proposing a new check, include:

- **ID:** stable identifier (example: `NUMA-001`)
- **Category:** kernel, memory, io, network, postgresql
- **What is detected:** file path, sysctl key, or config field
- **Status logic:** pass/warn/crit rules
- **Fix command:** copy-paste command
- **Safety level:** safe-runtime, reboot-required, high-risk
- **Reference URLs:** kernel docs, PostgreSQL docs, LKML, or vendor documentation
- **Confidence model:** when confidence should be downgraded

## Coding conventions

- Keep checks deterministic and testable.
- Do not hardcode environment-specific paths in tests.
- Preserve JSON schema compatibility in minor/patch versions.
