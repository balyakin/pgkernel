# KERN-001 — Preemption Model

## Why it matters

Kernel preemption mode directly affects PostgreSQL contention behavior on high-core systems.

## Detection

1. `/sys/kernel/debug/sched/preempt`
2. `uname -v` fallback
3. `/boot/config-$(uname -r)` fallback

## Status logic

- `none` -> pass
- `voluntary` -> pass
- `lazy` -> warn
- `full` / `preempt` -> crit
- unknown -> info

## Fix

```bash
echo none > /sys/kernel/debug/sched/preempt
```

Safety: `safe-runtime`

Reference: https://www.phoronix.com/news/Linux-7.0-AWS-PostgreSQL-Drop
