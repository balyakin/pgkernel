# KERN-002 — Kernel 7.0+ Warning

## Why it matters

Linux 7.0 changed default preemption behavior on modern architectures, which can impact PostgreSQL throughput when `preemption != none`.

## Detection

- Kernel version from `uname -r`
- Preemption model from KERN-001 detector

## Status logic

- Kernel `< 7.0` -> pass
- Kernel `>= 7.0` and preemption `none` -> pass
- Kernel `>= 7.0` and preemption not `none` -> warn

## Fix

```bash
echo none > /sys/kernel/debug/sched/preempt
```

Safety: `safe-runtime`

Reference: https://www.phoronix.com/news/Linux-Restrict-Preempt-Modes
