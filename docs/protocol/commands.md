---
title: "Command reference"
---

Commands use opcode `COMMAND` and a JSON request:

```json
{ "command": "GET", "args": { "key": "example" } }
```

## Common commands

These commands operate on the keyspace, server metadata, or execution envelope rather than on one type-specific implementation. Command names are case-insensitive; argument property names remain case-sensitive.

| Command       | Important arguments                                                                  | Result                                                                   |
| ------------- | ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------ |
| `PING`        | none                                                                                 | `PONG`                                                                   |
| `SET`         | `key`, `type`, `value`; optional `ttl_ms`, `nx:false`, `xx:false`                    | `true`                                                                   |
| `CREATE`      | `key`, `type`; optional `options:{}`, `ttl_ms`, `nx:false`, `xx:false`               | `true`                                                                   |
| `CAS`         | `key`, positive `expected_version`, `type`, `value`; optional `ttl_ms`               | `true` or `CONFLICT`                                                     |
| `BATCH`       | `requests`; optional `continue_on_error:false`                                       | ordered `{ok,result}` / `{ok,error}` items                               |
| `GET`         | `key`                                                                                | `{key,type,value,version}` plus `expires_at` when present                |
| `DEL`         | `keys`                                                                               | deleted count                                                            |
| `EXISTS`      | `key`                                                                                | boolean                                                                  |
| `TYPE`        | `key`                                                                                | stable wire type                                                         |
| `TTL`         | `key`                                                                                | remaining whole milliseconds or `-1` for no expiration                   |
| `EXPIRE`      | `key`, positive `ttl_ms`                                                             | boolean                                                                  |
| `PERSIST`     | `key`                                                                                | boolean                                                                  |
| `KEYS`        | optional `prefix:""`, `limit:1000`                                                   | bounded list; use `SCAN` for large keyspaces                             |
| `SCAN`        | optional opaque `cursor:""`, `prefix:""`, `count:100`                                | `{keys,cursor}` page                                                     |
| `BLOB_BEGIN`  | `key`, `total_size`; optional `upload_id`, `ttl_ms`, `nx`, `xx`, `connection_scoped` | upload ID, size, upload TTL, scope flag                                  |
| `BLOB_CHUNK`  | `upload_id`, `offset`, base64 `data`                                                 | received bytes                                                           |
| `BLOB_COMMIT` | `upload_id`                                                                          | boolean                                                                  |
| `BLOB_ABORT`  | `upload_id`                                                                          | boolean                                                                  |
| `BLOB_READ`   | `key`; optional `offset:0`, `length:1048576`                                         | key/version/offset/total size and base64 range                           |
| `TYPES`       | none                                                                                 | registered type names                                                    |
| `INFO`        | none                                                                                 | server, compression, and memory information                              |
| `MEMORY`      | `key`                                                                                | key/type/version/access count/approximate bytes and compression metadata |
| `OP`          | `key`, `operation`, `args`                                                           | type-specific result                                                     |

## Server command deadline

Normal command execution is bounded by `limits.command_timeout_ms`. The default is 30 seconds. A protocol `CANCEL` frame can cancel a compatible in-flight request earlier.

Blocking data-type operations therefore cannot hold a server request forever even if the caller requests a longer wait.

## Optimistic `CAS`

Read `version` with `GET`, then replace only that version:

```json
{
  "command": "CAS",
  "args": {
    "key": "account:1",
    "expected_version": 4,
    "type": "json",
    "value": { "balance": 120 },
    "ttl_ms": 60000
  }
}
```

A stale version returns a conflict. AOF replay stores the resolved absolute expiration, so restart does not extend a CAS TTL.

`SET`, `CREATE`, and `CAS` are full replacements. A positive relative TTL is converted to an absolute `expires_at_unix_nano` before journaling. `nx` requires the key to be absent; `xx` requires it to exist. Direct wire types and composite construction options are defined in [Type construction and operations](../data-types/operations.md).

## `BATCH`

A batch accepts up to 1024 normal Amaquet requests. It reduces round trips. It is **not** a rollback transaction. Earlier successful requests remain applied if a later request fails. With `continue_on_error:false`, the returned array ends at the first error; with `true`, later requests are attempted. Each nested mutation crosses the ordinary AOF transaction boundary independently.

## Cursor `SCAN`

`SCAN` uses an opaque cursor. Start with an empty cursor and continue until the returned cursor is empty. Do not parse or construct cursors in application code.

The implementation scans shard pages instead of materializing and sorting the complete keyspace for every page. Ordering is lexical within each of 256 shards, not globally lexical and not a point-in-time snapshot. Inserts, deletes, and expirations during iteration can affect later pages. `count` and `KEYS limit` are capped at 10,000; non-positive values select their defaults.

## Large binary values

A protocol frame is limited to 64 MiB. Chunked upload permits a `binary_string` value up to 1 GiB while keeping each request bounded.

Typical sequence:

1. `BLOB_BEGIN`
2. one or more non-empty, non-overlapping `BLOB_CHUNK` requests
3. `BLOB_COMMIT`, or `BLOB_ABORT`

`total_size` can be 0 through 1 GiB. If `upload_id` is omitted, the dispatcher generates a random 128-bit hexadecimal ID before persistence. Chunks are standard base64, must be non-empty, cannot overlap, and cannot extend beyond the declared size. An upload accepts at most 65,536 spans. Pending uploads have a global byte budget and inactivity TTL. A connection-scoped upload can only be modified by its exact owner connection; ordinary uploads are not owner-bound.

At commit, the object is read from the spool file into bounded 1 MiB in-memory blocks. It is not copied into one contiguous 1 GiB Go slice.

Use `BLOB_READ` for large values:

```json
{ "command": "BLOB_READ", "args": { "key": "archive", "offset": 0, "length": 1048576 } }
```

`length` is limited to 32 MiB per request and reads at end-of-object may be shorter. A normal `GET` returns large chunked-binary metadata for objects above 32 MiB instead of creating an oversized response.

The Go client exposes `UploadBlob` and `BlobRead` helpers.

## `OP`

`OP` dispatches operations implemented by the stored logical type. See the [data-type catalog](../data-types/index.md) for type-specific operations.

## Introspection results

`TYPES` returns all 91 stable type names. `INFO` returns `name`, `protocol_version`, live `keys`, `uptime_seconds`, registered `types`, aggregate `compression`, and engine `memory` statistics. `MEMORY` returns `compressed:false` for native storage; compressed payloads add codec, original/stored sizes, and saving information.

Top-level command names and operation names are normalized to uppercase. JSON argument property names are case-sensitive. Unknown commands, missing/invalid arguments, and unsupported type operations return an error rather than being ignored.
