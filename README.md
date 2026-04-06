# pgkernel

> **One command. Full diagnosis. Zero guesswork.**

`pgkernel` checks the intersection of **Linux kernel behavior** and **PostgreSQL tuning** in one run, then prints copy-paste remediation commands.

[![Go Version](https://img.shields.io/badge/go-1.22%2B-00ADD8.svg)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Latest Release](https://img.shields.io/github/v/release/ebalyakin/pgkernel)](https://github.com/ebalyakin/pgkernel/releases)

## Demo

![pgkernel terminal demo](docs/assets/pgkernel-demo.gif)

The demo records `pgkernel check` with colorized output and actionable fixes.

## Quick install

```bash
# Homebrew
brew install pgkernel

# Go install
go install github.com/ebalyakin/pgkernel/cmd/pgkernel@latest

# Download binary
curl -sSL https://github.com/ebalyakin/pgkernel/releases/latest/download/pgkernel-linux-amd64 -o pgkernel
chmod +x pgkernel
```

## Quick start

```bash
sudo pgkernel check
```

For CI-friendly output:

```bash
pgkernel check --format json --quiet
```

## Why this exists

In 2026, Linux 7.0 preemption behavior changes were linked to serious PostgreSQL throughput drops on some workloads. Most tools check either PostgreSQL parameters or kernel hardening, but not their interaction. `pgkernel` closes that gap.

- HN discussion: https://news.ycombinator.com/item?id=47644864
- Phoronix report: https://www.phoronix.com/news/Linux-7.0-AWS-PostgreSQL-Drop
- LKML threads: https://lore.kernel.org/lkml/

## Command interface

```text
pgkernel check [flags]
pgkernel version
pgkernel --help
```

Core flags:

- `--format pretty|json|markdown`
- `--pg-config /path/to/postgresql.conf`
- `--only kernel,memory` or `--only KERN-001,PG-001`
- `--exclude IO-001`
- `--fail-on warn|crit`
- `--baseline` and `--compare-with` for drift-aware policy checks
- `--severity all|warn|crit`
- `--share` for copy-ready markdown snippet
- `--quiet` for exit-code-only CI mode

## Full check reference

| ID | Area | Risk class | Typical impact | Confidence notes | Safety level |
|---|---|---|---:|---|---|
| KERN-001 | Kernel preemption model | warn/crit | 0-92 | downgraded when fallback detection is used | safe-runtime |
| KERN-002 | Kernel 7.0+ interaction | warn | 0-68 | lower confidence when preemption model unknown | safe-runtime |
| KERN-003 | RSEQ support | info/warn | 10-35 | based on debugfs/proc/boot config availability | reboot-required |
| MEM-001 | Transparent Huge Pages | warn/crit | 0-91 | high confidence on Linux sysfs | safe-runtime |
| MEM-002 | Static huge pages | warn/crit | 0-85 | depends on pg config + meminfo completeness | high-risk |
| MEM-003 | vm.swappiness | warn/crit | 0-86 | high confidence from proc sysctl | safe-runtime |
| MEM-004 | vm.overcommit_memory | warn/crit | 0-88 | high confidence from proc sysctl | safe-runtime |
| MEM-005 | OOM score protection | warn/info | 0-58 | lower confidence when PID detection falls back | safe-runtime |
| IO-001 | Storage scheduler | warn | 0-48 | reduced confidence when mount-to-device mapping fails | safe-runtime |
| IO-002 | Dirty write-back ratios | warn | 0-45 | high confidence from proc sysctl | safe-runtime |
| NET-001 | TCP keepalive | info | 0-18 | high confidence from proc sysctl | safe-runtime |
| PG-001 | shared_buffers sizing | warn/crit | 0-90 | depends on parsable memory units and RAM visibility | reboot-required |
| PG-002 | work_mem headroom | warn | 0-60 | lower confidence when parser cannot compute bytes | reboot-required |
| PG-003 | WAL sanity | warn | 0-45 | parser fallback lowers confidence | reboot-required |

Detailed pages for each check are available in `docs/checks/`.

## CI/CD integration

### GitHub Actions example

```yaml
name: pgkernel-policy
on: [push, pull_request]

jobs:
  audit:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install pgkernel
        run: go install github.com/ebalyakin/pgkernel/cmd/pgkernel@latest
      - name: Run policy gate
        run: pgkernel check --format json --fail-on crit --quiet
```

### Baseline and regression mode

```bash
pgkernel check --format json --baseline .ci/pgkernel-baseline.json --compare-with .ci/last-good.json --quiet
```

## Docker usage

```bash
docker run --rm --privileged \
  -v /proc:/host/proc:ro \
  -v /sys:/host/sys:ro \
  ghcr.io/ebalyakin/pgkernel:latest \
  pgkernel check
```

## Share workflow

Generate a markdown report with compact top-risks section:

```bash
pgkernel check --format markdown --share > pgkernel-report.md
```

The output is optimized for GitHub issues, Slack, and incident postmortems.

## Benchmark evidence

`benchmarks/` stores the reproducible protocol and data layout:

- target SLO: `p50 <= 1s`, `p95 <= 2s`
- before/after benchmark profiles per kernel/preemption mode
- raw snapshots and summary markdown for release gates

See `benchmarks/README.md`.

## Competitive comparison

| Tool | Focus | What it misses |
|---|---|---|
| pgtune / PGTuner | PostgreSQL settings by hardware | no kernel + scheduler + sysctl interaction |
| kernel-hardening-checker | kernel security posture | no PostgreSQL-specific diagnostics |
| pgcenter | runtime activity monitoring | no host-level remediation advice |
| pgbench | benchmark generator | no root-cause analysis |
| **pgkernel** | **kernel + PostgreSQL interaction with fix commands** | — |

## Public check pages

- `docs/checks/INDEX.md`
- `docs/checks/KERN-001.md`
- `docs/checks/KERN-002.md`
- `docs/checks/KERN-003.md`
- `docs/checks/MEM-001.md`
- `docs/checks/MEM-002.md`
- `docs/checks/MEM-003.md`
- `docs/checks/MEM-004.md`
- `docs/checks/MEM-005.md`
- `docs/checks/IO-001.md`
- `docs/checks/IO-002.md`
- `docs/checks/NET-001.md`
- `docs/checks/PG-001.md`
- `docs/checks/PG-002.md`
- `docs/checks/PG-003.md`

## Telemetry and privacy

Current release has no forced telemetry. If telemetry is added in future releases, it is strictly opt-in and documented.

## Contributing

Contributions are welcome:

1. Open an issue with environment details and report snippet.
2. For new checks, provide rationale, reference links, and remediation safety level.
3. Add tests for parser logic and policy behavior.

## License

MIT, (c) Evgeny Balyakin
