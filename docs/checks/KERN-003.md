# KERN-003 — RSEQ Support

## Why it matters

Kernel support for restartable sequences (rseq) is a foundation for future PostgreSQL optimizations that reduce scheduling overhead.

## Detection

- `/sys/kernel/debug/rseq`
- `/proc/sys/kernel/rseq`
- `CONFIG_RSEQ` in boot config fallback

## Status logic

- rseq supported -> info
- rseq unsupported on kernel `<7.0` -> info
- rseq unsupported on kernel `>=7.0` -> warn

## Fix

Use a kernel build with `CONFIG_RSEQ=y`.

Safety: `reboot-required`

Reference: https://lore.kernel.org/lkml/
