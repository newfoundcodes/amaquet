---
title: "Command reference"
description: "Complete reference for Amaquet top-level commands, arguments, validation rules, result shapes, conditional writes, scans, batches, and chunked binary transfers."
---

Amaquet commands are JSON requests carried in a `COMMAND` protocol frame. The command name is case-insensitive; JSON argument names are case-sensitive. A request has this shape:

```json
{
  "command": "GET",
  "args": { "key": "profile:42" }
}
```

The response is a protocol envelope. Successful commands return `ok: true` and a command-specific `result`; failures return `ok: false` and an error object:

```json
{ "ok": true, "result": { "key": "profile:42", "type": "json", "value": {}, "version": 1 } }
```

```json
{ "ok": false, "error": { "code": "NOT_FOUND", "message": "key not found" } }
```

Native protocol authorization classifies commands as `data.read` or `data.write`. `PING`, introspection, key reads, `BLOB_READ`, and read-only `OP` operations require `data.read`; writes require `data.write`. See [authentication and identity](authentication.md) and [errors](../reference/errors.md).

## Argument conventions

These conventions apply across the top-level commands. Type-specific operation arguments can have their own defaults and units, so those are documented separately in the `OP` section and type reference.

| Argument form | Meaning |
| --- | --- |
| `key` | A non-empty string in the single global keyspace. Logical database paths are not supported. |
| `type` | One of the stable names returned by `TYPES`, such as `integer`, `json`, or `fifo_queue`. |
| `value` | A direct value whose JSON representation matches that type, or a nested wire-value object when used inside a composite type. |
| `ttl_ms` | A relative lifetime accepted as numeric milliseconds or a Go duration string such as `"2h"`. For `SET`, `CREATE`, `CAS`, and `BLOB_BEGIN`, only a positive parsed duration creates expiry; omitted, zero, negative, or unparsable values result in no expiry. `EXPIRE` is stricter and rejects non-positive values. |
| `expires_at_unix_nano` | Advanced absolute expiry timestamp used by replayed journal requests. A positive value takes precedence over `ttl_ms`. |
| `nx` | Conditional write: succeed only when the key does not exist as a live key. Defaults to `false`. |
| `xx` | Conditional write: succeed only when the key already exists as a live key. Defaults to `false`. |
| `cursor` | Opaque `SCAN` continuation token. Treat it as an uninterpreted string. |
| `count` / `limit` | Requested result size. Relevant commands cap values at 10,000 unless a command documents a smaller limit. |

Direct values use the top-level `type` and `value` fields:

```json
{
  "command": "SET",
  "args": {
    "key": "counter",
    "type": "integer",
    "value": 41,
    "ttl_ms": 60000
  }
}
```

Nested values use an explicit envelope:

```json
{
  "type": "json",
  "value": { "job_id": "job-1", "attempt": 1 }
}
```

The complete direct-value encoding table and all type-specific `OP` arguments are documented in [wire values](wire-values.md) and [type construction and operations](../data-types/operations.md).

## Command summary

Use this table to choose the command and identify its required fields before reading the detailed sections below.

| Command | Required arguments | Main optional arguments | Result |
| --- | --- | --- | --- |
| `PING` | none | none | `"PONG"` |
| `SET` | `key`, `type`, `value` | `ttl_ms`, `nx`, `xx` | `true` |
| `CREATE` | `key`, `type` | `options`, `ttl_ms`, `nx`, `xx` | `true` |
| `CAS` | `key`, `expected_version`, `type`, `value` | `ttl_ms` | `true` |
| `BATCH` | `requests` | `continue_on_error` | Ordered per-request result objects |
| `GET` | `key` | none | Key, type, value, version, optional expiry |
| `DEL` | `keys` | none | Number of deleted keys |
| `EXISTS` | `key` | none | Boolean |
| `TYPE` | `key` | none | Stable type name |
| `TTL` | `key` | none | Remaining milliseconds or `-1` |
| `EXPIRE` | `key`, `ttl_ms` | none | `true` |
| `PERSIST` | `key` | none | `true` |
| `KEYS` | none | `prefix`, `limit` | Bounded key array |
| `SCAN` | none | `cursor`, `prefix`, `count` | `{keys,cursor}` |
| `BLOB_BEGIN` | `key`, `upload_id`, `total_size` | `ttl_ms`, `nx`, `xx`, `connection_scoped` | Upload metadata |
| `BLOB_CHUNK` | `upload_id`, `offset`, `data` | none | Received and total byte counts |
| `BLOB_COMMIT` | `upload_id` | none | `true` |
| `BLOB_READ` | `key` | `offset`, `length` | Base64 binary range |
| `BLOB_ABORT` | `upload_id` | none | Boolean |
| `TYPES` | none | none | Array of 91 type names |
| `INFO` | none | none | Runtime metadata and statistics |
| `MEMORY` | `key` | none | Approximate per-key memory metadata |
| `OP` | `key`, `operation` | nested `args` | Operation-specific result |

## Liveness and introspection

These commands inspect connectivity, the registered type catalog, and runtime accounting without changing stored values.

### `PING`

`PING` takes no arguments and returns the string `PONG`. It is a lightweight command-path check, not a persistence or readiness check.

```json
{ "command": "PING", "args": {} }
```

For HTTP health and AOF readiness, use `/api/health` and `/api/ready` on the administration listener.

### `TYPES`

`TYPES` takes no arguments and returns all registered stable type names. The current implementation returns 91 names. Use this to discover the type spelling accepted by `SET` and `CREATE`.

```json
{ "command": "TYPES", "args": {} }
```

### `INFO`

`INFO` takes no arguments and returns a runtime snapshot:

```json
{
  "name": "Amaquet",
  "protocol_version": 1,
  "keys": 12,
  "uptime_seconds": 184,
  "types": 91,
  "compression": { "compressed_keys": 2, "original_bytes": 4096, "stored_bytes": 1200 },
  "memory": { "used_bytes": 8192, "max_bytes": 0, "policy": "noeviction", "evicted_keys": 0 }
}
```

The memory and compression objects contain additional implementation-defined counters. `used_bytes` is an approximate engine accounting value, not process RSS.

### `MEMORY`

`MEMORY` requires `key` and returns per-entry accounting and access metadata:

```json
{ "command": "MEMORY", "args": { "key": "counter" } }
```

The result includes `key`, `type`, `version`, `access_count`, `approx_bytes`, and `compressed`. Compressed values add codec and original/stored-size fields. The estimate excludes some Go runtime and process-level overhead.

## Key lifecycle commands

These commands create, read, replace, conditionally update, expire, and delete keys in the single global keyspace.

### `SET`

`SET` replaces or creates a directly encoded value. Its required arguments are `key`, a registered `type`, and a JSON `value` matching that type. A positive `ttl_ms` applies expiry; numeric milliseconds and Go duration strings are accepted. `nx` and `xx` implement conditional creation/replacement. A successful replacement of a live key advances its version; creating a new key starts at version 1. A failed conditional write returns an error and does not replace the value.

```json
{
  "command": "SET",
  "args": {
    "key": "counter",
    "type": "integer",
    "value": 41,
    "nx": true,
    "ttl_ms": 300000
  }
}
```

Use `SET` for scalar and directly representable values. Use `CREATE` for composite values that need constructor options, then use `OP`.

### `CREATE`

`CREATE` constructs a value through the registered type factory. It requires `key` and `type`; `options` is an optional object whose fields are type-specific. It also accepts `ttl_ms`, `nx`, and `xx`.

```json
{
  "command": "CREATE",
  "args": {
    "key": "jobs",
    "type": "fifo_queue",
    "options": {},
    "nx": true
  }
}
```

An example with constructor options:

```json
{
  "command": "CREATE",
  "args": {
    "key": "events",
    "type": "ring_buffer",
    "options": { "capacity": 1024 }
  }
}
```

The accepted option names and defaults are type-specific; consult the corresponding type page before constructing specialized values.

### `GET`

`GET` requires `key` and returns the logical value snapshot:

```json
{
  "key": "counter",
  "type": "integer",
  "value": 42,
  "version": 2,
  "expires_at": "2026-10-23T12:00:00Z"
}
```

`expires_at` is omitted when the key has no expiry. Mutable values are returned as public snapshots rather than internal pointers. Large chunked binary values above 32 MiB return metadata instead of a large inline value; use `BLOB_READ` for ranges.

### `CAS`

`CAS` is an optimistic replacement. It requires the current positive `expected_version`, plus the same `key`, `type`, and `value` fields used by `SET`. It optionally accepts `ttl_ms`.

```json
{
  "command": "CAS",
  "args": {
    "key": "counter",
    "expected_version": 2,
    "type": "integer",
    "value": 43,
    "ttl_ms": 60000
  }
}
```

The command returns `true` only when the stored version still matches. A stale version returns `CONFLICT`. The version is read with `GET`; successful TTL changes and successful mutations also advance it.

### `DEL`

`DEL` requires `keys`, an array of key strings, and returns the number of live keys removed. Missing keys do not cause an error.

```json
{
  "command": "DEL",
  "args": { "keys": ["temporary", "old-session", "missing"] }
}
```

### `EXISTS` and `TYPE`

Both commands require `key`. `EXISTS` returns a boolean. `TYPE` returns the stable type name and fails with `NOT_FOUND` when the key is absent.

```json
{ "command": "EXISTS", "args": { "key": "counter" } }
```

```json
{ "command": "TYPE", "args": { "key": "counter" } }
```

### `TTL`

`TTL` requires `key` and returns remaining whole milliseconds. It returns `-1` for a live key without an expiry and returns `NOT_FOUND` for a missing or already-expired key.

```json
{ "command": "TTL", "args": { "key": "counter" } }
```

### `EXPIRE` and `PERSIST`

`EXPIRE` requires `key` and a positive `ttl_ms` (numeric milliseconds or a Go duration string); it rejects zero or negative durations, replaces the current expiry, and increments the key version. `PERSIST` requires `key`, removes an existing expiry, and also increments the version. Both fail with `NOT_FOUND` for a missing or expired key.

```json
{ "command": "EXPIRE", "args": { "key": "counter", "ttl_ms": 60000 } }
```

```json
{ "command": "PERSIST", "args": { "key": "counter" } }
```

Expiry is enforced both lazily during access and by a background expiry worker. A TTL does not survive a restart unless the mutation is journaled through AOF.

## Key enumeration

Enumeration is bounded and best-effort: neither command provides a point-in-time snapshot of the keyspace.

### `KEYS`

`KEYS` accepts optional `prefix` and `limit` arguments. `limit` defaults to 1000, non-positive values use the default, and values above 10,000 are capped. It returns a bounded array of matching live keys.

```json
{
  "command": "KEYS",
  "args": { "prefix": "user:", "limit": 50 }
}
```

Use `KEYS` for bounded administrative inspection. It is not a point-in-time snapshot and its result is not globally lexical.

### `SCAN`

`SCAN` is the incremental alternative for larger keyspaces. Its arguments are optional: `cursor` is empty for the first page, `prefix` filters keys, and `count` defaults to 100 and is capped at 10,000.

```json
{
  "command": "SCAN",
  "args": { "cursor": "", "prefix": "session:", "count": 100 }
}
```

The result is `{ "keys": [...], "cursor": "..." }`. Continue until `cursor` is empty. Cursors are opaque and do not represent a snapshot: inserts, deletes, and expirations during iteration can affect later pages. Keys are ordered within shard-local pages; no global ordering is promised.

## Batches and conditional execution

Use these commands to reduce round trips or protect a replacement against a stale version; neither provides a general multi-command rollback transaction.

### `BATCH`

`BATCH` requires `requests`, an array of ordinary Amaquet request objects. It accepts at most 1,024 nested requests. `continue_on_error` defaults to `false`.

```json
{
  "command": "BATCH",
  "args": {
    "continue_on_error": true,
    "requests": [
      { "command": "SET", "args": { "key": "a", "type": "integer", "value": 1 } },
      { "command": "INEXISTENT", "args": {} },
      { "command": "GET", "args": { "key": "a" } }
    ]
  }
}
```

The result is an ordered array of `{ok,result}` or `{ok,error}` objects. With `continue_on_error:false`, execution stops after the first failed nested request. With `true`, later requests are attempted. `BATCH` is not atomic and does not roll back earlier successful requests; each nested mutation has its own ordinary persistence boundary.

## Chunked binary commands

Protocol frames are limited to 64 MiB. Chunked uploads support a logical binary value up to 1 GiB while spooling incoming chunks to temporary files. Chunks use standard padded base64. A complete upload is committed as the `binary_string` type.

### `BLOB_BEGIN`

`BLOB_BEGIN` requires `key`, a non-empty `upload_id`, and `total_size` from 0 through 1 GiB. Optional arguments are `ttl_ms`, `nx`, `xx`, and `connection_scoped`.

```json
{
  "command": "BLOB_BEGIN",
  "args": {
    "key": "archive",
    "upload_id": "archive-2026-01",
    "total_size": 10485760,
    "ttl_ms": 86400000,
    "nx": true,
    "connection_scoped": true
  }
}
```

`ttl_ms` controls the expiry of the final stored value; unfinished uploads also receive the configured inactivity TTL. `connection_scoped:true` binds later chunk, commit, and abort requests to the exact native connection that created the upload. The result contains `upload_id`, `total_size`, `expires_in_ms`, and `connection_scoped`. Pending uploads consume the global `max_pending_upload_bytes` budget.

### `BLOB_CHUNK`

`BLOB_CHUNK` requires `upload_id`, a non-negative byte `offset`, and non-empty base64 `data`. The decoded range must remain within `total_size`, cannot overlap an earlier chunk, and each upload is limited to 65,536 spans.

```json
{
  "command": "BLOB_CHUNK",
  "args": {
    "upload_id": "archive-2026-01",
    "offset": 0,
    "data": "AAECAwQ="
  }
}
```

The result reports `{ "received": N, "total_size": M }`. `received` counts unique decoded bytes, not the highest offset.

### `BLOB_COMMIT`

`BLOB_COMMIT` requires `upload_id`. It succeeds only when every byte from offset 0 through `total_size-1` has been received. It returns `true`, creates or replaces the destination `binary_string`, applies the upload's `nx`/`xx` and final TTL settings, and removes the temporary spool file.

```json
{ "command": "BLOB_COMMIT", "args": { "upload_id": "archive-2026-01" } }
```

Commit reads the spool in bounded blocks and reserves engine capacity before installing the value. A pre-installation failure leaves the upload available for retry until its inactivity deadline. Once the value has been installed, a later AOF commit failure can return an error while the in-memory mutation remains applied; that persistence failure is handled by the server's degraded fail-stop path and the upload is no longer retryable.

### `BLOB_READ`

`BLOB_READ` requires `key`, which must contain a `binary_string`. Optional `offset` defaults to 0 and `length` defaults to 1 MiB. Both must be non-negative; `length` cannot exceed 32 MiB per request. Reads at the end of the object may return fewer bytes.

```json
{
  "command": "BLOB_READ",
  "args": { "key": "archive", "offset": 1048576, "length": 262144 }
}
```

The result is:

```json
{
  "key": "archive",
  "version": 3,
  "offset": 1048576,
  "size": 10485760,
  "data": "...base64..."
}
```

### `BLOB_ABORT`

`BLOB_ABORT` requires `upload_id`, removes the unfinished upload, and returns `true` when it existed or `false` when it was already absent. It cannot abort an upload while it is committing.

```json
{ "command": "BLOB_ABORT", "args": { "upload_id": "archive-2026-01" } }
```

## `OP`: type-specific operations

`OP` requires a stored `key` and an `operation` string. Its optional nested `args` object contains operation-specific arguments:

```json
{
  "command": "OP",
  "args": {
    "key": "jobs",
    "operation": "ENQUEUE",
    "args": {
      "value": { "type": "json", "value": { "id": "job-1" } }
    }
  }
}
```

Operation names are case-insensitive, but nested argument names are case-sensitive. The operation must be supported by the stored type. Read operations return a type-specific result and do not increment the key version; successful mutations increment it. Blocking operations such as blocking-queue dequeue, semaphore acquire, and barrier waits observe the request context and the server `command_timeout_ms` deadline.

Common operation argument patterns include:

| Pattern | Examples |
| --- | --- |
| Nested typed value | `value: {"type":"utf8_string","value":"build"}` for lists, queues, sets, maps, and records |
| Numeric delta | `delta` for integer, unsigned-integer, float, and atomic-counter `ADD` |
| Range | `start`, `stop`, `from`, `to`, or `count`, depending on the collection or log |
| Time | Unix milliseconds for samples/availability; Unix nanoseconds are used by replay-oriented timestamp fields |
| Vector search | `vector` plus optional `k` and type-specific metric/configuration |
| Lease/lock ownership | `owner`, `token`, and optional positive `ttl_ms`; ownership is token-based, not inferred from authentication |

Examples:

```json
{
  "command": "OP",
  "args": { "key": "counter", "operation": "ADD", "args": { "delta": 1 } }
}
```

```json
{
  "command": "OP",
  "args": {
    "key": "scores",
    "operation": "ADD",
    "args": {
      "member": { "type": "utf8_string", "value": "alice" },
      "score": 97.5
    }
  }
}
```

```json
{
  "command": "OP",
  "args": {
    "key": "timeline",
    "operation": "ADD",
    "args": {
      "fields": {
        "kind": { "type": "utf8_string", "value": "login" },
        "user_id": { "type": "integer", "value": 42 }
      }
    }
  }
}
```

The [type construction and operation reference](../data-types/operations.md) is the authoritative operation-by-operation argument and result catalog. It includes all 91 registered types, constructor options, read/write behavior, defaults, and constraints.

## Execution, cancellation, and persistence

Normal command execution is bounded by `limits.command_timeout_ms`, which defaults to 30 seconds. A client can send a `CANCEL` frame for an active request, but cancellation is cooperative: the operation must observe the context. Request IDs are independent, so multiplexed commands can complete out of order.

When AOF is enabled, mutating top-level commands and journalable mutating `OP` operations are written through prepare/commit records. Read operations are not journaled. Ephemeral coordination mutations—such as Pub/Sub, leases, locks, barriers, semaphores, and rate-limit buckets—are intentionally not persisted because restoring ownership state could recreate stale coordination. See [persistence](../concepts/persistence.md) for replay and durability behavior.

`BATCH` does not create a multi-command transaction. A successful command that later encounters an AOF commit failure may already have changed memory while the server enters persistence-degraded fail-stop behavior; clients should reconcile state rather than assuming a non-OK mutation was necessarily absent.
