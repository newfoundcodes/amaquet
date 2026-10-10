---
title: "Observability"
---

This page describes the native and HTTP surfaces that expose Amaquet health, capacity, and runtime metrics.

## Amaquet `INFO`

`INFO` returns server identity, protocol version, live key count, uptime, type count, compression totals, approximate engine memory usage, configured memory limit, eviction policy, and eviction count.

`MEMORY {"key":"..."}` reports per-key version/access metadata, approximate bytes, and physical compression information.

## Health endpoints

The administration server exposes:

- `/api/health` for liveness;
- `/api/ready` for readiness plus key/memory statistics and persistence health;
- `/metrics` for Prometheus/OpenMetrics-style process and database metrics.

Readiness becomes unhealthy when AOF persistence reports a background flush/fsync failure. Memory statistics are informational: crossing the configured engine-memory budget does not by itself make this endpoint return HTTP 503.

## Metrics

The exact metrics are:

- `amaquet_keys`;
- `amaquet_memory_bytes` and `amaquet_memory_limit_bytes`;
- `amaquet_evicted_keys_total`;
- `amaquet_compressed_keys` and `amaquet_compression_saved_bytes`;
- `amaquet_go_heap_bytes` and `amaquet_go_goroutines`;
- `amaquet_ready` (`1` or `0`).

The endpoint uses Prometheus text format version 0.0.4. It requires `system.read` by default and can be made public with `admin.metrics_require_auth:false`.

The engine-memory value is an application estimate. Monitor process RSS/working-set separately because Go runtime pages, temporary buffers, TLS, connections, filesystem buffers, and native/runtime overhead are outside that counter.

## Logs

The process uses `github.com/charmbracelet/log` with structured key/value fields for listener startup, persistence replay/open status, identity information, configuration failures, and shutdown events. Production deployments should collect these logs with timestamps and retain persistence or authentication errors as security/availability signals.
