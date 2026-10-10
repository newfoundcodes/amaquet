---
title: "Engine internals"
---

This page describes the behavior implemented by `internal/core`, including constraints that are not visible in the short command reference.

## Sharding and locking

The engine creates 256 shards. FNV-1a over the complete key selects a shard, and every shard owns a Go map protected by an `RWMutex`. Independent shards can proceed concurrently. Key writes acquire one shard's write lock; reads acquire its read lock and atomically update access telemetry.

Composite values have their own locks. The dispatcher obtains the value, performs its operation through that object, and then updates the engine's version and size metadata. Blocking queues, barriers, and semaphores therefore wait on a request context without pinning the keyspace shard lock.

## Entry metadata and versions

Each entry records:

- the stable `DataType` plus its physical Go value;
- creation, update, and optional expiration timestamps;
- a monotonically increasing per-key version;
- atomic access count and last-access time;
- approximate stored bytes used by memory governance.

Replacing an existing key preserves `CreatedAt` and increments its version. A deleted then recreated key starts again at version 1. Successful `EXPIRE` and `PERSIST` also increment the version. `CAS` requires a positive expected version and atomically replaces only that version.

## Expiration algorithm

TTL is enforced in two ways:

1. Reads and writes lazily remove an expired entry encountered on its shard.
2. One background goroutine maintains an indexed min-heap ordered by expiration time.

There is at most one heap node per currently expiring key. Updating TTL fixes that node in place; `PERSIST` removes it. The heap item includes the entry version, so the expirer cannot delete a newer replacement based on an obsolete deadline.

`TTL` returns `-1` for an existing persistent key and `NOT_FOUND` for a missing/expired key. A non-positive relative expiration is rejected. During deterministic replay, an already-passed absolute expiration leaves the key absent.

## Scans

`SCAN` encodes the current shard number and last returned key into an opaque base64url cursor. For each visited shard it copies the live matching keys under a read lock, sorts that shard-local page, and continues lexicographically after the cursor key. It does not create one globally sorted snapshot.

Consequences of a live, non-snapshot scan:

- concurrent insertions/deletions can change later pages;
- ordering is deterministic within the observed shard pages, not a global lexical ordering across all shards;
- callers must treat the cursor as opaque and stop when it becomes empty;
- `count` is clamped to 10,000 and defaults to 100.

`KEYS` repeatedly consumes `SCAN`, defaults to 1,000, and is also capped at 10,000.

## Approximate memory accounting

The engine estimates each entry from the key, a fixed metadata allowance, and either an implementation-provided approximate size, a snapshot JSON size, or a conservative fallback. This counter is designed for admission and eviction decisions; it is not allocator accounting or process RSS.

Writers reserve estimated positive growth before making a mutation visible. Reservations participate in concurrent capacity checks. The exact estimated delta is applied when metadata is committed, and the reservation is then released. Memory-reducing operations require no reservation.

When `max_memory_bytes` is non-zero, the Go runtime soft memory limit is set to 125% of that value. This is headroom, not a hard operating-system limit.

## Eviction

Supported policies are:

| Policy         | Candidate comparison                                       |
| -------------- | ---------------------------------------------------------- |
| `noeviction`   | Reject growth with `MEMORY_LIMIT`                          |
| `allkeys-lru`  | Oldest sampled last-access time                            |
| `allkeys-lfu`  | Lowest sampled access count                                |
| `volatile-ttl` | Nearest sampled expiration; persistent keys are ineligible |

Victim selection is approximate. One eviction attempt visits up to all 256 shards but samples at most two eligible map entries per shard. Go map iteration provides the spread; it is not a strict global LRU/LFU ordering. The key currently being mutated is excluded from victim selection.

## Engine transactions

The internal `Engine.Transaction` API sorts and locks all involved shard IDs to prevent lock-order deadlocks. It provides staged copies of entry metadata and permits only non-pointer/non-map/non-slice composite payloads, because mutating a shared object would defeat rollback. An error from the callback commits nothing.

The public `BATCH` command is different: it reduces round trips but executes requests sequentially and does not roll back earlier successes.

## Compression boundary

Compression changes only physical `Value.Data`; the stable type ID stays unchanged. Eligible values are converted to canonical bytes before codec selection and reconstructed before normal reads/operations. Compression statistics are updated incrementally as entries are installed, replaced, or removed. See [Adaptive compression](compression.md).

## Shutdown

Closing the engine stops and joins the expiration goroutine. Process shutdown separately stops the admin server, closes protocol connections, aborts incomplete upload state, flushes the AOF, and closes control-plane state.
