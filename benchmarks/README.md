# Benchmarks

This directory contains reproducible benchmark artifacts for pgkernel release gates.

## Goal

- Runtime SLO: `p50 <= 1s`, `p95 <= 2s`

## Suggested layout

- `profiles/` — host profile metadata (CPU, RAM, kernel, PostgreSQL)
- `raw/` — raw benchmark outputs
- `summary/` — release-ready markdown summaries

## Repro protocol

1. Capture baseline before tuning.
2. Apply one remediation at a time.
3. Re-run workload and `pgkernel check`.
4. Save raw data and generated report JSON.
5. Publish summary with commands, environment, and confidence notes.
