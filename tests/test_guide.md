# pgkernel test guide

## Purpose

This guide describes fast validation scenarios for policy logic and high-risk checks.

## Commands

```bash
go test ./...
go run ./cmd/pgkernel check --format json --severity warn
```

## Expected markers

- Policy tests must verify exit code precedence (runtime error > critical > warning > pass).
- Check tests must validate semantic edge cases:
  - `KERN-001` returns `warn` for `PREEMPT_LAZY`.
  - `MEM-001` returns `crit` when THP mode is `always`.
  - `PG-001` returns `crit` when `shared_buffers=128MB` on high-RAM hosts.

## Regression workflow

1. Save a baseline report:

```bash
pgkernel check --format json > .ci/pgkernel-baseline.json
```

2. Compare current run with baseline:

```bash
pgkernel check --format json --compare-with .ci/pgkernel-baseline.json --quiet
```

3. Non-zero exit code indicates regression according to `--fail-on` threshold.
